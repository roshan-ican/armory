package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const preMaintainerSchema = `
CREATE TABLE users (
    id INTEGER PRIMARY KEY, name TEXT NOT NULL, service_no TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'requester')), password_hash TEXT,
    active INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
) STRICT;
CREATE TABLE maintenance_jobs (
    id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users (id),
    status TEXT NOT NULL DEFAULT 'assigned' CHECK (status IN ('assigned', 'active', 'finished', 'cancelled')),
    assigned_by INTEGER REFERENCES users (id), created_at TEXT NOT NULL, started_at TEXT, ended_at TEXT
) STRICT;
CREATE UNIQUE INDEX one_open_maintenance_per_user ON maintenance_jobs (user_id) WHERE status IN ('assigned', 'active');
CREATE TABLE face_enrollment_requests (
    id INTEGER PRIMARY KEY, name TEXT NOT NULL, service_no TEXT NOT NULL, face_refs TEXT NOT NULL,
    model_version TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    decided_by INTEGER REFERENCES users (id), created_at TEXT NOT NULL, decided_at TEXT
) STRICT;
INSERT INTO face_enrollment_requests (name, service_no, face_refs, model_version, created_at) VALUES ('Old Pending', 'OP', '[]', 'm', 't');
INSERT INTO users VALUES (1, 'Ada', 'A1', 'admin', 'hash', 1, 't', 't'), (2, 'Bob', 'B2', 'requester', NULL, 1, 't', 't');
INSERT INTO maintenance_jobs VALUES (1, 2, 'active', 1, 't', 't', NULL);
`

func TestOpenUpgradesOldMaintenanceSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(preMaintainerSchema); err != nil {
		t.Fatal(err)
	}
	old.Close()

	for i := 0; i < 2; i++ {
		db, err := Open(path)
		if err != nil {
			t.Fatalf("open %d: %v", i, err)
		}

		var users, jobs int
		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users)
		db.QueryRow("SELECT COUNT(*) FROM maintenance_jobs WHERE status = 'active'").Scan(&jobs)
		if users != 2 || jobs != 1 {
			t.Fatalf("open %d: users=%d active jobs=%d", i, users, jobs)
		}
		if _, err := db.Exec("INSERT INTO users (name, service_no, role, active, created_at, updated_at) VALUES ('Cy', 'C3', 'requester', 1, 't', 't')"); err != nil {
			t.Fatalf("extra user rejected: %v", err)
		}
		if _, err := db.Exec("INSERT INTO maintenance_jobs (user_id, status, created_at) VALUES (3, 'requested', 't')"); err != nil {
			t.Fatalf("requested status rejected: %v", err)
		}
		if _, err := db.Exec("INSERT INTO maintenance_jobs (user_id, status, created_at) VALUES (3, 'assigned', 't')"); err == nil {
			t.Fatal("second open job for one user was accepted")
		}
		var fk int
		rows, _ := db.Query("PRAGMA foreign_key_check")
		for rows.Next() {
			fk++
		}
		rows.Close()
		if fk != 0 {
			t.Fatalf("foreign key violations: %d", fk)
		}
		db.Exec("DELETE FROM maintenance_jobs WHERE user_id = 3")
		db.Exec("DELETE FROM users WHERE id = 3")
		db.Close()
	}
}

const leftoverSchema = `
CREATE TABLE users (
    id INTEGER PRIMARY KEY, name TEXT NOT NULL, service_no TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'maintainer', 'requester')), password_hash TEXT,
    active INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
) STRICT;
CREATE TABLE face_enrollment_requests (
    id INTEGER PRIMARY KEY, name TEXT NOT NULL, service_no TEXT NOT NULL, face_refs TEXT NOT NULL,
    model_version TEXT NOT NULL, requested_role TEXT NOT NULL DEFAULT 'requester' CHECK (requested_role IN ('requester', 'maintainer')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    decided_by INTEGER REFERENCES users (id), created_at TEXT NOT NULL, decided_at TEXT
) STRICT;
INSERT INTO users VALUES (1, 'Admin', 'AD', 'admin', 'h', 1, 't', 't'), (2, 'Fixer', 'FX', 'maintainer', NULL, 1, 't', 't');
`

func TestOpenKeepsWorkingOnADatabaseFromTheMaintainerRoleVersion(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "leftover.db")
	old, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(leftoverSchema); err != nil {
		t.Fatal(err)
	}
	old.Close()

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st := NewStore(db)

	if err := st.CreateFaceEnrollmentRequest(ctx, "New Person", "NEW PERSON", `["[1,2,3]"]`, "m"); err != nil {
		t.Fatal(err)
	}
	pending, err := st.ListPendingFaceEnrollmentRequests(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %v, %v", pending, err)
	}
	if err := st.DecideFaceEnrollmentRequest(ctx, pending[0].ID, 1, true); err != nil {
		t.Fatal(err)
	}
	var role string
	if err := db.QueryRow("SELECT role FROM users WHERE service_no = 'NEW PERSON'").Scan(&role); err != nil || role != "requester" {
		t.Fatalf("approved person role = %q, %v", role, err)
	}
	if _, err := st.GetUser(ctx, 2); err != nil {
		t.Fatalf("an old maintainer row must still load: %v", err)
	}
}
