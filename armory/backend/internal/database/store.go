package database

import (
	"database/sql"
	"errors"
	"sync/atomic"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type Store struct {
	db      *sql.DB
	faceGen atomic.Int64
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) FaceGeneration() int64 {
	return s.faceGen.Load()
}

func (s *Store) faceChanged() {
	s.faceGen.Add(1)
}

func (s *Store) commitFaces(tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		return err
	}
	s.faceChanged()
	return nil
}

func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	return errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullIfZero(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}
