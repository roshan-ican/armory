package database

import (
	"database/sql"
	"errors"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	return errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func nullIfZero(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
