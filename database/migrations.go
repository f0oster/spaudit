package database

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migration represents a database migration
type Migration struct {
	Version string
	Name    string
	SQL     string
}

// getMigrations returns all available migrations sorted by version
func getMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		// Extract version from filename (e.g., "0001_init.sql" -> "0001")
		name := strings.TrimSuffix(entry.Name(), ".sql")
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 1 {
			continue
		}

		migrations = append(migrations, Migration{
			Version: parts[0],
			Name:    name,
			SQL:     string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// createMigrationsTable creates the migrations tracking table
func (d *Database) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := d.writeDB.Exec(query); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// getAppliedMigrations returns a set of applied migration versions
func (d *Database) getAppliedMigrations() (map[string]bool, error) {
	applied := make(map[string]bool)

	rows, err := d.readDB.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	return applied, nil
}

// applyMigration executes a single migration
func (d *Database) applyMigration(migration Migration) error {
	d.logger.Database("Applying migration",
		"version", migration.Version,
		"name", migration.Name)

	// Start transaction (use write connection for schema changes)
	tx, err := d.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
	}

	// Record migration as applied
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		migration.Version, migration.Name,
	); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", migration.Name, err)
	}

	d.logger.Database("Migration applied successfully",
		"version", migration.Version,
		"name", migration.Name)

	return nil
}

// runMigrations applies all pending migrations
func (d *Database) runMigrations() error {
	d.logger.Database("Checking for database migrations")

	// Create migrations table if it doesn't exist
	if err := d.createMigrationsTable(); err != nil {
		return err
	}

	// Get all available migrations
	migrations, err := getMigrations()
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		d.logger.Database("No migrations found")
		return nil
	}

	// Get applied migrations
	applied, err := d.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Apply pending migrations
	appliedCount := 0
	for _, migration := range migrations {
		if applied[migration.Version] {
			d.logger.Database("Migration already applied",
				"version", migration.Version,
				"name", migration.Name)
			continue
		}

		if err := d.applyMigration(migration); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.Name, err)
		}
		appliedCount++
	}

	if appliedCount > 0 {
		d.logger.Database("Database migrations completed",
			"applied", appliedCount,
			"total", len(migrations))
	} else {
		d.logger.Database("Database is up to date",
			"total_migrations", len(migrations))
	}

	return nil
}

// checkDatabaseExists checks if the database file exists
func checkDatabaseExists(path string) bool {
	if path == ":memory:" {
		return false // In-memory database always needs initialization
	}

	// For file paths, check if file exists
	if strings.HasPrefix(path, "file:") {
		// Extract file path from DSN
		path = strings.TrimPrefix(path, "file:")
		if idx := strings.Index(path, "?"); idx != -1 {
			path = path[:idx]
		}
	}

	// Check if file exists and is not empty
	if info, err := filepath.Abs(path); err == nil {
		if stat, err := os.Stat(info); err == nil && stat.Size() > 0 {
			return true
		}
	}

	return false
}
