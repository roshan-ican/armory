package database

import (
	"context"
	"testing"
)

func TestApprovingEnrollmentReactivatesDisabledPerson(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	create := func(name, key string) int64 {
		t.Helper()
		if err := store.CreateFaceEnrollmentRequest(ctx, name, key, `["[1,2,3]"]`, "m"); err != nil {
			t.Fatal(err)
		}
		pending, err := store.ListPendingFaceEnrollmentRequests(ctx)
		if err != nil || len(pending) == 0 {
			t.Fatalf("pending: %v %v", pending, err)
		}
		return pending[0].ID
	}

	first := create("r", "R")
	if err := store.DecideFaceEnrollmentRequest(ctx, first, 1, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE users SET active = 0 WHERE service_no = 'R'"); err != nil {
		t.Fatal(err)
	}

	second := create("r", "R")
	if err := store.DecideFaceEnrollmentRequest(ctx, second, 1, true); err != nil {
		t.Fatalf("approving a returning person: %v", err)
	}
	var active, users int
	store.db.QueryRowContext(ctx, "SELECT active FROM users WHERE service_no = 'R'").Scan(&active)
	store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE service_no = 'R'").Scan(&users)
	if active != 1 || users != 1 {
		t.Fatalf("active = %d, users = %d, want 1 and 1", active, users)
	}
}
