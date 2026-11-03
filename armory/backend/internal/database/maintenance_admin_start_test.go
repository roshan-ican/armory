package database

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"armory/internal/models"
)

func TestStartMaintenanceByAdmin(t *testing.T) {
	ctx := context.Background()
	st, carl, admin := maintenanceFixture(t)
	if _, err := st.CreateLocker(ctx, models.Locker{Name: "L1", Kind: "rifle", Capacity: 3}); err != nil {
		t.Fatal(err)
	}

	job, err := st.StartMaintainceByAdmin(ctx, carl.ID, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != models.MaintenanceActive {
		t.Fatalf("status = %s, want active", job.Status)
	}
	if active, err := st.MaintenanceActive(ctx); err != nil || !active {
		t.Fatalf("MaintenanceActive = %v, %v", active, err)
	}
	if got := eventTypes(t, st, carl.ID); !reflect.DeepEqual(got, []string{"maintenance_started"}) {
		t.Fatalf("events = %v", got)
	}
	snap, err := st.ListMaintenanceSnapshot(ctx, job.ID)
	if err != nil || len(snap) != 1 {
		t.Fatalf("snapshot = %v, %v", snap, err)
	}

	if _, err := st.StartMaintainceByAdmin(ctx, carl.ID, admin.ID); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("second start = %v, want ErrDuplicate", err)
	}
	if _, err := st.StartMaintainceByAdmin(ctx, admin.ID, admin.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("admin as maintainer = %v, want ErrConflict", err)
	}

	list, err := st.ListActiveMaintenance(ctx)
	if err != nil || len(list) != 1 || list[0].UserName != carl.Name {
		t.Fatalf("active list = %+v, %v", list, err)
	}

	if err := st.FinishMaintenanceByID(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.FinishMaintenanceByID(ctx, job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second finish = %v, want ErrNotFound", err)
	}
	if active, _ := st.MaintenanceActive(ctx); active {
		t.Fatal("maintenance still active after finish")
	}
	if got := eventTypes(t, st, carl.ID); !reflect.DeepEqual(got, []string{"maintenance_started", "maintenance_finished"}) {
		t.Fatalf("events = %v", got)
	}
}
