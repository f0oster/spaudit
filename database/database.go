package database

import (
	"database/sql"
	"fmt"
	"time"

	"spaudit/gen/db"
	"spaudit/logging"

	_ "modernc.org/sqlite"
)

// Config holds database configuration
type Config struct {
	Path              string        `env:"DB_PATH" default:"./spaudit.db"`
	MaxOpenConns      int           `env:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns      int           `env:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime   time.Duration `env:"DB_CONN_MAX_LIFETIME" default:"1h"`
	ConnMaxIdleTime   time.Duration `env:"DB_CONN_MAX_IDLE_TIME" default:"15m"`
	BusyTimeoutMs     int           `env:"DB_BUSY_TIMEOUT_MS" default:"5000"`
	EnableForeignKeys bool          `env:"DB_ENABLE_FOREIGN_KEYS" default:"true"`
	EnableWAL         bool          `env:"DB_ENABLE_WAL" default:"true"`
	StrictMode        bool          `env:"DB_STRICT_MODE" default:"true"`
	SerializeWrites   bool          `env:"DB_SERIALIZE_WRITES" default:"false"`
}

// Database wraps the SQL database connections and provides managed access
type Database struct {
	readDB       *sql.DB     // Connection pool for reads
	writeDB      *sql.DB     // Serialized connection for writes
	readQueries  *db.Queries // Queries using read connection
	writeQueries *db.Queries // Queries using write connection
	config       Config
	logger       *logging.Logger
}

// New creates a new Database instance with separate read/write connections
func New(config Config, logger *logging.Logger) (*Database, error) {
	dsn := buildDSN(config)

	// Check if database needs to be initialized
	dbExists := checkDatabaseExists(config.Path)

	logger.Database("Opening database connections",
		"path", config.Path,
		"exists", dbExists,
		"read_max_open_conns", config.MaxOpenConns,
		"write_max_open_conns", 1)

	// Create read connection pool
	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open read database: %w", err)
	}

	// Configure read connection pool for concurrency
	readDB.SetMaxOpenConns(config.MaxOpenConns)
	readDB.SetMaxIdleConns(config.MaxIdleConns)
	readDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	readDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Create write connection (serialized)
	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		readDB.Close()
		return nil, fmt.Errorf("failed to open write database: %w", err)
	}

	// Configure write connection for serialization
	writeDB.SetMaxOpenConns(1) // Single connection forces serialization
	writeDB.SetMaxIdleConns(1) // Keep the connection alive
	writeDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	writeDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	database := &Database{
		readDB:       readDB,
		writeDB:      writeDB,
		readQueries:  db.New(readDB),
		writeQueries: db.New(writeDB),
		config:       config,
		logger:       logger,
	}

	// Test connections and configure database
	if err := database.initialize(); err != nil {
		readDB.Close()
		writeDB.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Run migrations to ensure database schema is up to date
	if err := database.runMigrations(); err != nil {
		readDB.Close()
		writeDB.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	logger.Database("Database initialized successfully",
		"path", config.Path,
		"existed", dbExists,
		"wal_mode", config.EnableWAL,
		"read_connections", config.MaxOpenConns,
		"write_connections", 1)

	return database, nil
}

// buildDSN constructs the SQLite Data Source Name with proper parameters
func buildDSN(config Config) string {
	dsn := fmt.Sprintf("file:%s?", config.Path)

	// Add performance and reliability settings
	dsn += fmt.Sprintf("_busy_timeout=%d", config.BusyTimeoutMs)

	if config.EnableWAL {
		dsn += "&_journal_mode=WAL"
	}

	if config.EnableForeignKeys {
		dsn += "&_foreign_keys=on"
	}

	// Add other important SQLite settings for concurrent performance
	dsn += "&_cache_size=-64000"       // 64MB cache
	dsn += "&_temp_store=memory"       // Use memory for temp tables
	dsn += "&_synchronous=normal"      // Balance between safety and performance
	dsn += "&_wal_autocheckpoint=1000" // Checkpoint every 1000 pages
	dsn += "&_mmap_size=268435456"     // 256MB memory-mapped I/O

	if config.StrictMode {
		// Note: strict mode needs to be enabled per connection via PRAGMA
		// We'll handle this in initialize()
	}

	return dsn
}

// initialize configures both database connections after creation
func (d *Database) initialize() error {
	// Test read connection
	if err := d.readDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping read database: %w", err)
	}

	// Test write connection
	if err := d.writeDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping write database: %w", err)
	}

	// Configure both connections with same settings
	connections := []*sql.DB{d.readDB, d.writeDB}
	connectionTypes := []string{"read", "write"}

	for i, conn := range connections {
		connType := connectionTypes[i]

		// Enable strict mode if configured (per-connection setting)
		if d.config.StrictMode {
			if _, err := conn.Exec("PRAGMA strict=ON"); err != nil {
				d.logger.Warn("Failed to enable strict mode", "connection", connType, "error", err)
			}
		}

		// Set busy timeout
		if _, err := conn.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", d.config.BusyTimeoutMs)); err != nil {
			return fmt.Errorf("failed to set busy_timeout on %s connection: %w", connType, err)
		}
		d.logger.Info("Busy timeout configured", "connection", connType, "timeout_ms", d.config.BusyTimeoutMs)

		// Enable and verify WAL mode
		if d.config.EnableWAL {
			// Force enable WAL mode via PRAGMA
			var journalMode string
			err := conn.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode)
			if err != nil {
				return fmt.Errorf("failed to enable WAL mode on %s connection: %w", connType, err)
			}

			if journalMode != "wal" {
				d.logger.Warn("WAL mode not enabled", "connection", connType, "journal_mode", journalMode)
			} else {
				d.logger.Info("WAL mode enabled successfully", "connection", connType, "journal_mode", journalMode)
			}
		}
	}

	// Log connection pool stats for monitoring
	d.logPoolStats()

	return nil
}

// Queries returns the read queries interface (for backwards compatibility)
func (d *Database) Queries() *db.Queries {
	return d.readQueries
}

// ReadQueries returns the read-optimized queries interface
func (d *Database) ReadQueries() *db.Queries {
	return d.readQueries
}

// WriteQueries returns the write-serialized queries interface
func (d *Database) WriteQueries() *db.Queries {
	return d.writeQueries
}

// DB returns the read database connection (for backwards compatibility)
func (d *Database) DB() *sql.DB {
	return d.readDB
}

// ReadDB returns the read database connection
func (d *Database) ReadDB() *sql.DB {
	return d.readDB
}

// WriteDB returns the write database connection
func (d *Database) WriteDB() *sql.DB {
	return d.writeDB
}

// Close closes both database connections
func (d *Database) Close() error {
	d.logger.Database("Closing database connections")

	if d.config.EnableWAL {
		d.logger.Info("checkpointing WAL...")
		if _, err := d.writeDB.Exec("PRAGMA wal_checkpoint(TRUNCATE);"); err != nil {
			d.logger.Warn("failed to checkpoint WAL", "error", err)
		}
	}

	var errs []error
	if err := d.readDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("read connection: %w", err))
	}
	if err := d.writeDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("write connection: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close connections: %v", errs)
	}
	return nil
}

// Health checks database connectivity and returns pool statistics for both connections
func (d *Database) Health() (map[string]interface{}, error) {
	if err := d.readDB.Ping(); err != nil {
		return nil, fmt.Errorf("read database ping failed: %w", err)
	}
	if err := d.writeDB.Ping(); err != nil {
		return nil, fmt.Errorf("write database ping failed: %w", err)
	}

	readStats := d.readDB.Stats()
	writeStats := d.writeDB.Stats()

	return map[string]interface{}{
		"read_pool": map[string]interface{}{
			"open_connections":    readStats.OpenConnections,
			"in_use":              readStats.InUse,
			"idle":                readStats.Idle,
			"wait_count":          readStats.WaitCount,
			"wait_duration":       readStats.WaitDuration.String(),
			"max_idle_closed":     readStats.MaxIdleClosed,
			"max_lifetime_closed": readStats.MaxLifetimeClosed,
			"max_open_conns":      d.config.MaxOpenConns,
		},
		"write_pool": map[string]interface{}{
			"open_connections":    writeStats.OpenConnections,
			"in_use":              writeStats.InUse,
			"idle":                writeStats.Idle,
			"wait_count":          writeStats.WaitCount,
			"wait_duration":       writeStats.WaitDuration.String(),
			"max_idle_closed":     writeStats.MaxIdleClosed,
			"max_lifetime_closed": writeStats.MaxLifetimeClosed,
			"max_open_conns":      1,
		},
	}, nil
}

// logPoolStats logs current connection pool statistics for both connections
func (d *Database) logPoolStats() {
	readStats := d.readDB.Stats()
	writeStats := d.writeDB.Stats()

	d.logger.Database("Read connection pool stats",
		"open_connections", readStats.OpenConnections,
		"in_use", readStats.InUse,
		"idle", readStats.Idle,
		"wait_count", readStats.WaitCount,
		"wait_duration", readStats.WaitDuration.String())

	d.logger.Database("Write connection pool stats",
		"open_connections", writeStats.OpenConnections,
		"in_use", writeStats.InUse,
		"idle", writeStats.Idle,
		"wait_count", writeStats.WaitCount,
		"wait_duration", writeStats.WaitDuration.String())
}

// WithTx executes a function within a database transaction (uses write connection)
func (d *Database) WithTx(fn func(*db.Queries) error) error {
	tx, err := d.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	qtx := d.writeQueries.WithTx(tx)

	if err := fn(qtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			d.logger.Error("Failed to rollback transaction", "error", rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithReadTx executes a function within a read-only transaction
func (d *Database) WithReadTx(fn func(*db.Queries) error) error {
	tx, err := d.readDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin read transaction: %w", err)
	}

	qtx := d.readQueries.WithTx(tx)

	if err := fn(qtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			d.logger.Error("Failed to rollback read transaction", "error", rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit read transaction: %w", err)
	}

	return nil
}
