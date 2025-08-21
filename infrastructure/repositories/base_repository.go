package repositories

import (
	"database/sql"
	"time"

	"spaudit/database"
	"spaudit/gen/db"
)

// BaseRepository provides common SQL type conversion methods and database access that can be embedded in all repositories.
type BaseRepository struct {
	db *database.Database
}

// NewBaseRepository creates a new BaseRepository with database access
func NewBaseRepository(database *database.Database) *BaseRepository {
	return &BaseRepository{
		db: database,
	}
}

// ReadQueries returns the read-optimized queries interface for SELECT operations
func (b *BaseRepository) ReadQueries() *db.Queries {
	return b.db.ReadQueries()
}

// WriteQueries returns the write-serialized queries interface for INSERT/UPDATE/DELETE operations
func (b *BaseRepository) WriteQueries() *db.Queries {
	return b.db.WriteQueries()
}

// WithTx executes a function within a write transaction
func (b *BaseRepository) WithTx(fn func(*db.Queries) error) error {
	return b.db.WithTx(fn)
}

// WithReadTx executes a function within a read transaction
func (b *BaseRepository) WithReadTx(fn func(*db.Queries) error) error {
	return b.db.WithReadTx(fn)
}

// FromNullString safely converts sql.NullString to string.
// Returns empty string if the SQL value is NULL.
func (b *BaseRepository) FromNullString(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

// FromNullBool safely converts sql.NullBool to bool.
// Returns false if the SQL value is NULL.
func (b *BaseRepository) FromNullBool(nb sql.NullBool) bool {
	if !nb.Valid {
		return false
	}
	return nb.Bool
}

// FromNullInt64 safely converts sql.NullInt64 to int64.
// Returns 0 if the SQL value is NULL.
func (b *BaseRepository) FromNullInt64(ni sql.NullInt64) int64 {
	if !ni.Valid {
		return 0
	}
	return ni.Int64
}

// FromNullTime safely converts sql.NullTime to *time.Time.
// Returns nil if the SQL value is NULL.
func (b *BaseRepository) FromNullTime(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

// ToNullString converts a string to sql.NullString.
// Empty string becomes NULL for database storage.
func (b *BaseRepository) ToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// ToNullBool converts a bool to sql.NullBool.
// Always valid since Go bool has no null state.
func (b *BaseRepository) ToNullBool(value bool) sql.NullBool {
	return sql.NullBool{Bool: value, Valid: true}
}

// ToNullInt64 converts an int64 to sql.NullInt64.
// Zero value becomes NULL for database storage.
func (b *BaseRepository) ToNullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}

// ToNullInt32 converts an int32 to sql.NullInt32.
// Zero value becomes NULL for database storage.
func (b *BaseRepository) ToNullInt32(i int32) sql.NullInt32 {
	if i == 0 {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: i, Valid: true}
}

// ToNullTime converts a *time.Time to sql.NullTime.
// Nil pointer becomes NULL for database storage.
func (b *BaseRepository) ToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// FromNullInt64ToPointer safely converts sql.NullInt64 to *int64.
// Returns nil if the SQL value is NULL.
func (b *BaseRepository) FromNullInt64ToPointer(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}
