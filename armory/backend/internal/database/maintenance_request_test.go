package database

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"armory/internal/models"
)

func TestRequestMaintenance(t *testing.T) {
	ctx := context.Background()
	st, roshan, admin := maintenanceFixture(t)
	maint, err := st.CreateUser(ctx, models.User{Name: "Fixer", ServiceNo: "MT-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	l1, err := st.CreateLocker(ctx, models.Locker{Name: "L1", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	l2, err := st.CreateLocker(ctx, models.Locker{Name: "L2", Kind: "rifle", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}

	plan := []models.MaintenanceChange{
		{LockerID: l2.ID, Action: models.ChangeRemoveSensor},
		{LockerID: l1.ID, Action: models.ChangeReplaceSensor, SlotNo: 2},
	}
	job, err := st.RequestMaintenance(ctx, maint.ID, plan)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != models.MaintenanceRequested || len(job.Changes) != 2 || job.Changes[0].ID == 0 || job.Changes[1].JobID != job.ID {
		t.Fatalf("job = %+v", job)
	}

	rows, err := st.db.Query("SELECT locker_id, action, COALESCE(slot_no, 0) FROM maintenance_changes WHERE job_id = ? ORDER BY id", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got [][3]any
	for rows.Next() {
		var lid, slot int64
		var action string
		if err := rows.Scan(&lid, &action, &slot); err != nil {
			t.Fatal(err)
		}
		got = append(got, [3]any{lid, action, slot})
	}
	want := [][3]any{{l2.ID, "remove_sensor", int64(0)}, {l1.ID, "replace_sensor", int64(2)}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stored changes = %v, want %v", got, want)
	}

	if got := eventTypes(t, st, maint.ID); !reflect.DeepEqual(got, []string{"maintenance_requested"}) {
		t.Fatalf("events = %v", got)
	}

	if _, err := st.OpenMaintenanceFor(ctx, maint.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("requested job must not be startable yet, got %v", err)
	}

	if _, err := st.RequestMaintenance(ctx, maint.ID, plan); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("second open request = %v, want ErrDuplicate", err)
	}
	if _, err := st.RequestMaintenance(ctx, admin.ID, plan); !errors.Is(err, ErrConflict) {
		t.Fatalf("admin asking = %v, want ErrConflict", err)
	}
	if job, err := st.RequestMaintenance(ctx, roshan.ID, plan); err != nil || job.Status != models.MaintenanceRequested {
		t.Fatalf("a plain requester asking = %+v, %v", job, err)
	}
	if _, err := st.RequestMaintenance(ctx, 9999, plan); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown user = %v, want ErrNotFound", err)
	}

	other, err := st.CreateUser(ctx, models.User{Name: "Other", ServiceNo: "MT-2", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	bad := []models.MaintenanceChange{{LockerID: 9999, Action: models.ChangeAddSensor}}
	if _, err := st.RequestMaintenance(ctx, other.ID, bad); err == nil {
		t.Fatal("unknown locker was accepted")
	}
	var n int
	st.db.QueryRow("SELECT COUNT(*) FROM maintenance_jobs WHERE user_id = ?", other.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("failed request left %d job rows behind", n)
	}
}

func TestApproveAndRejectRequestedMaintenance(t *testing.T) {
	ctx := context.Background()
	st, _, admin := maintenanceFixture(t)
	maint, err := st.CreateUser(ctx, models.User{Name: "Fixer", ServiceNo: "MT-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	l1, err := st.CreateLocker(ctx, models.Locker{Name: "L1", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	plan := []models.MaintenanceChange{{LockerID: l1.ID, Action: models.ChangeAddSensor}}

	job, err := st.RequestMaintenance(ctx, maint.ID, plan)
	if err != nil {
		t.Fatal(err)
	}

	if err := st.ApproveMaintenance(ctx, 9999, admin.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("approve unknown = %v, want ErrNotFound", err)
	}
	if err := st.ApproveMaintenance(ctx, job.ID, admin.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.ApproveMaintenance(ctx, job.ID, admin.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("approve twice = %v, want ErrNotFound", err)
	}

	open, err := st.OpenMaintenanceFor(ctx, maint.ID)
	if err != nil || open.ID != job.ID || open.Status != models.MaintenanceAssigned {
		t.Fatalf("open after approve = %+v, %v", open, err)
	}
	var by int64
	st.db.QueryRow("SELECT assigned_by FROM maintenance_jobs WHERE id = ?", job.ID).Scan(&by)
	if by != admin.ID {
		t.Fatalf("assigned_by = %d, want %d", by, admin.ID)
	}
	if got := eventTypes(t, st, maint.ID); !reflect.DeepEqual(got, []string{"maintenance_requested", "maintenance_approved"}) {
		t.Fatalf("events = %v", got)
	}

	if err := st.CancelMaintenance(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	second, err := st.RequestMaintenance(ctx, maint.ID, plan)
	if err != nil {
		t.Fatalf("new request after cancel: %v", err)
	}
	if err := st.CancelMaintenance(ctx, second.ID); err != nil {
		t.Fatalf("reject a requested job: %v", err)
	}
	if err := st.CancelMaintenance(ctx, second.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cancel twice = %v, want ErrNotFound", err)
	}
	if err := st.StartMaintenance(ctx, second.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("start a cancelled job = %v, want ErrConflict", err)
	}
}

func TestStartMaintenanceTakesSnapshot(t *testing.T) {
	ctx := context.Background()
	st, _, admin := maintenanceFixture(t)
	maint, err := st.CreateUser(ctx, models.User{Name: "Fixer", ServiceNo: "MT-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	l1, err := st.CreateLocker(ctx, models.Locker{Name: "L1", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	l2, err := st.CreateLocker(ctx, models.Locker{Name: "L2", Kind: "rifle", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	st.db.Exec("UPDATE slots SET reading = 1 WHERE locker_id = ?", l1.ID)
	st.db.Exec("UPDATE slots SET reading = 1 WHERE locker_id = ? AND slot_no = 1", l2.ID)
	st.db.Exec("UPDATE slots SET reading = 0 WHERE locker_id = ? AND slot_no = 2", l2.ID)

	plan := []models.MaintenanceChange{
		{LockerID: l2.ID, Action: models.ChangeRemoveSensor},
		{LockerID: l1.ID, Action: models.ChangeReplaceSensor, SlotNo: 2},
	}
	job, err := st.RequestMaintenance(ctx, maint.ID, plan)
	if err != nil {
		t.Fatal(err)
	}
	if snap, err := st.ListMaintenanceSnapshot(ctx, job.ID); err != nil || len(snap) != 0 {
		t.Fatalf("snapshot before start = %v, %v", snap, err)
	}
	if err := st.ApproveMaintenance(ctx, job.ID, admin.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.StartMaintenance(ctx, job.ID); err != nil {
		t.Fatal(err)
	}

	st.db.Exec("UPDATE lockers SET capacity = 1 WHERE id = ?", l2.ID)
	st.db.Exec("UPDATE slots SET reading = 0 WHERE locker_id = ?", l1.ID)
	if err := st.StartMaintenance(ctx, job.ID); err != nil {
		t.Fatal(err)
	}

	snap, err := st.ListMaintenanceSnapshot(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []models.LockerSnapshot{
		{LockerID: l1.ID, Capacity: 3, GunsPresent: 3},
		{LockerID: l2.ID, Capacity: 2, GunsPresent: 1},
	}
	if !reflect.DeepEqual(snap, want) {
		t.Fatalf("snapshot = %+v, want %+v", snap, want)
	}

	changes, err := st.ListMaintenanceChanges(ctx, job.ID)
	for i := range job.Changes {
		job.Changes[i].LockerName = map[int64]string{l1.ID: "L1", l2.ID: "L2"}[job.Changes[i].LockerID]
	}
	if err != nil || !reflect.DeepEqual(changes, job.Changes) {
		t.Fatalf("changes = %+v, %v; want %+v", changes, err, job.Changes)
	}
	if none, err := st.ListMaintenanceChanges(ctx, 9999); err != nil || len(none) != 0 {
		t.Fatalf("changes for unknown job = %v, %v", none, err)
	}
}
