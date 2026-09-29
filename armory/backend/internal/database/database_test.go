package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

const oldSchema = `
CREATE TABLE lockers (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, location TEXT, ip_address TEXT,
	kind TEXT NOT NULL DEFAULT 'rifle', capacity INTEGER NOT NULL DEFAULT 5) STRICT;
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, service_no TEXT NOT NULL UNIQUE,
	role TEXT NOT NULL, active INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL) STRICT;
CREATE TABLE slots (id INTEGER PRIMARY KEY, locker_id INTEGER NOT NULL REFERENCES lockers (id), slot_no INTEGER NOT NULL,
	sensor_id INTEGER NOT NULL UNIQUE, reading INTEGER, active INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE (locker_id, slot_no)) STRICT;
CREATE TABLE categories (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, active INTEGER NOT NULL DEFAULT 1) STRICT;
CREATE TABLE guns (id INTEGER PRIMARY KEY, slot_id INTEGER UNIQUE REFERENCES slots (id),
	category_id INTEGER NOT NULL REFERENCES categories (id), kind TEXT NOT NULL, notes TEXT,
	status TEXT NOT NULL DEFAULT 'in', created_at TEXT NOT NULL, updated_at TEXT NOT NULL) STRICT;
CREATE TABLE requests (id INTEGER PRIMARY KEY, requester_id INTEGER NOT NULL REFERENCES users (id),
	category_id INTEGER REFERENCES categories (id), gun_id INTEGER REFERENCES guns (id),
	status TEXT NOT NULL DEFAULT 'pending', reason TEXT, admin_note TEXT, decided_by INTEGER REFERENCES users (id),
	valid_until TEXT, expected_return_at TEXT, created_at TEXT NOT NULL, decided_at TEXT, collected_at TEXT, returned_at TEXT) STRICT;
CREATE TABLE events (id INTEGER PRIMARY KEY, occurred_at TEXT NOT NULL, type TEXT NOT NULL,
	user_id INTEGER REFERENCES users (id), gun_id INTEGER REFERENCES guns (id), slot_id INTEGER REFERENCES slots (id),
	request_id INTEGER REFERENCES requests (id), details TEXT) STRICT;
INSERT INTO lockers (id, name, ip_address) VALUES (1, 'A', '10.0.0.1');
INSERT INTO slots (id, locker_id, slot_no, sensor_id, reading, created_at, updated_at) VALUES (1, 1, 1, 1, 1, 't', 't');
INSERT INTO categories (id, name) VALUES (1, 'Rifle');
INSERT INTO guns (id, slot_id, category_id, kind, created_at, updated_at) VALUES (1, 1, 1, 'rifle', 't', 't');
INSERT INTO events (id, occurred_at, type, gun_id, slot_id) VALUES (1, '2026-09-29T10:00:00Z', 'gun_removed', 1, 1);
INSERT INTO events (id, occurred_at, type, gun_id, slot_id) VALUES (2, '2026-09-29T10:05:00Z', 'gun_returned', 1, 1);
`

func TestOpenDropsGunsAndKeepsHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(oldSchema); err != nil {
		t.Fatal(err)
	}
	old.Close()

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	for _, table := range []string{"guns", "categories", "events_old", "requests_old"} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name = ?", table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("table %s still exists", table)
		}
	}

	events, err := NewStore(db).ListEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Type != "gun_returned" || events[0].LockerName != "A" {
		t.Fatalf("events after upgrade = %+v", events)
	}

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatalf("reopen after upgrade: %v", err)
	}
	again.Close()
}
