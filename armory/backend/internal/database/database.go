package database

import (
	"context"
	"database/sql"
	"fmt"

	"armory/migrations"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if _, err := db.Exec(migrations.Schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	if err := addSlotReading(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("add slots.reading: %w", err)
	}

	if err := migrateDropGuns(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("drop guns: %w", err)
	}

	return db, nil
}

// Databases created before slots.reading existed need the column added in place.
func addSlotReading(db *sql.DB) error {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slots') WHERE name = 'reading'").Scan(&n)
	if err != nil || n > 0 {
		return err
	}
	_, err = db.Exec("ALTER TABLE slots ADD COLUMN reading INTEGER CHECK (reading IN (0, 1, 2))")
	return err
}

// Databases from before guns were removed keep their events and requests; the guns and categories tables are dropped.
func migrateDropGuns(db *sql.DB) error {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'guns'").Scan(&n)
	if err != nil || n == 0 {
		return err
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	steps := []string{
		"ALTER TABLE events RENAME TO events_old",
		"ALTER TABLE requests RENAME TO requests_old",
		migrations.Schema,
		`INSERT INTO requests (id, requester_id, status, reason, admin_note, decided_by, valid_until, expected_return_at, created_at, decided_at, collected_at, returned_at)
		 SELECT id, requester_id, status, reason, admin_note, decided_by, valid_until, expected_return_at, created_at, decided_at, collected_at, returned_at FROM requests_old`,
		`INSERT INTO events (id, occurred_at, type, user_id, slot_id, request_id, details)
		 SELECT id, occurred_at, type, user_id, slot_id, request_id, details FROM events_old`,
		"DROP TABLE events_old",
		"DROP TABLE requests_old",
		"DROP TABLE guns",
		"DROP TABLE IF EXISTS categories",
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step); err != nil {
			return err
		}
	}
	return tx.Commit()
}
