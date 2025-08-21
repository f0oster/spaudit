package handlers

import (
	"database/sql"
)

// Helper functions to unwrap SQLC's nullable and bool-as-int fields.

// ns converts a sql.NullString to string, returning empty string if null.
func ns(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

// ib converts int64 to bool (non-zero is true).
func ib(i int64) bool {
	return i != 0
}

// ni converts a sql.NullInt64 to int64, returning 0 if null.
func ni(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0
}

// nb converts a sql.NullBool to bool, returning false if null.
func nb(n sql.NullBool) bool {
	if n.Valid {
		return n.Bool
	}
	return false
}
