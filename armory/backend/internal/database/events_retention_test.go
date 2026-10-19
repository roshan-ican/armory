package database

import (
	"context"
	"testing"
)

func TestDeleteEventsBefore(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	for _, at := range []string{"2026-08-01T10:00:00Z", "2026-09-06T23:59:59Z", "2026-09-07T00:00:00Z", "2026-10-01T10:00:00Z"} {
		if _, err := st.db.Exec("INSERT INTO events (occurred_at, type) VALUES (?, 'gun_removed')", at); err != nil {
			t.Fatal(err)
		}
	}

	n, err := st.DeleteEventsBefore(ctx, "2026-09-07T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("deleted %d events, want 2", n)
	}

	var left int
	if err := st.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 2 {
		t.Fatalf("%d events left, want 2", left)
	}
}
