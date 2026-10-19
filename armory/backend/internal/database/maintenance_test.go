package database

import (
	"context"
	"errors"
	"testing"

	"armory/internal/models"
)

func maintenanceFixture(t *testing.T) (*Store, models.User, models.User) {
	t.Helper()
	ctx := context.Background()
	st := newTestStore(t)
	roshan, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := st.CreateUser(ctx, models.User{Name: "Boss", ServiceNo: "AD-1", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	return st, roshan, admin
}

func eventTypes(t *testing.T, st *Store, userID int64) []string {
	t.Helper()
	rows, err := st.db.Query("SELECT type FROM events WHERE user_id = ? ORDER BY id", userID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func TestMaintenanceLifecycle(t *testing.T) {
	ctx := context.Background()
	st, roshan, admin := maintenanceFixture(t)

	job, err := st.AssignMaintenance(ctx, roshan.ID, admin.ID)
	if err != nil || job.Status != models.MaintenanceAssigned {
		t.Fatalf("assign = %+v, %v", job, err)
	}
	open, err := st.OpenMaintenanceFor(ctx, roshan.ID)
	if err != nil || open.ID != job.ID {
		t.Fatalf("open = %+v, %v", open, err)
	}

	if err := st.StartMaintenance(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.StartMaintenance(ctx, job.ID); err != nil {
		t.Fatalf("starting twice should be harmless: %v", err)
	}
	if got, _ := st.OpenMaintenanceFor(ctx, roshan.ID); got.Status != models.MaintenanceActive {
		t.Fatalf("status = %q, want active", got.Status)
	}

	if err := st.FinishMaintenance(ctx, roshan.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.OpenMaintenanceFor(ctx, roshan.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a finished job is still open: %v", err)
	}
	if err := st.FinishMaintenance(ctx, roshan.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("finishing twice: %v", err)
	}

	want := []string{"maintenance_assigned", "maintenance_started", "maintenance_finished"}
	got := eventTypes(t, st, roshan.ID)
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestMaintenanceRules(t *testing.T) {
	ctx := context.Background()
	st, roshan, admin := maintenanceFixture(t)

	t.Run("only one open job per person", func(t *testing.T) {
		if _, err := st.AssignMaintenance(ctx, roshan.ID, admin.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := st.AssignMaintenance(ctx, roshan.ID, admin.ID); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("second assign: %v", err)
		}
	})

	t.Run("an admin cannot be assigned", func(t *testing.T) {
		if _, err := st.AssignMaintenance(ctx, admin.ID, admin.ID); !errors.Is(err, ErrConflict) {
			t.Fatalf("assign admin: %v", err)
		}
	})

	t.Run("an unknown or removed person cannot be assigned", func(t *testing.T) {
		if _, err := st.AssignMaintenance(ctx, 9999, admin.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("unknown: %v", err)
		}
		gone, err := st.CreateUser(ctx, models.User{Name: "Gone", ServiceNo: "SN-9", Role: "requester"})
		if err != nil {
			t.Fatal(err)
		}
		if err := st.DeactivateRequester(ctx, gone.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := st.AssignMaintenance(ctx, gone.ID, admin.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("removed: %v", err)
		}
	})

	t.Run("an assigned job that never started cannot be finished", func(t *testing.T) {
		if err := st.FinishMaintenance(ctx, roshan.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("finish before start: %v", err)
		}
	})

	t.Run("cancelling frees the person for a new job", func(t *testing.T) {
		job, err := st.OpenMaintenanceFor(ctx, roshan.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.CancelMaintenance(ctx, job.ID); err != nil {
			t.Fatal(err)
		}
		if err := st.CancelMaintenance(ctx, job.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("cancelling twice: %v", err)
		}
		if _, err := st.AssignMaintenance(ctx, roshan.ID, admin.ID); err != nil {
			t.Fatalf("re-assign after cancel: %v", err)
		}
	})
}
