package database

import (
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
