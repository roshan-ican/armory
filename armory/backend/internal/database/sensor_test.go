package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"armory/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStore(db)
}

func TestFindSensorSlot(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	locker, err := st.CreateLocker(ctx, models.Locker{Name: "A", IPAddress: "10.252.176.50"})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := st.CreateCategory(ctx, "Rifle")
	if err != nil {
		t.Fatal(err)
	}
	slot2 := locker.Slots[1]
	gun, err := st.CreateGun(ctx, models.Gun{SlotID: slot2.ID, CategoryID: cat.ID, Serial: "SN1"})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("slot with gun", func(t *testing.T) {
		slotID, gunID, err := st.FindSensorSlot(ctx, "10.252.176.50", 2)
		if err != nil {
			t.Fatal(err)
		}
		if slotID != slot2.ID || gunID != gun.ID {
			t.Fatalf("got slot %d gun %d, want slot %d gun %d", slotID, gunID, slot2.ID, gun.ID)
		}
	})

	t.Run("empty slot", func(t *testing.T) {
		slotID, gunID, err := st.FindSensorSlot(ctx, "10.252.176.50", 1)
		if err != nil {
			t.Fatal(err)
		}
		if slotID != locker.Slots[0].ID || gunID != 0 {
			t.Fatalf("got slot %d gun %d, want slot %d gun 0", slotID, gunID, locker.Slots[0].ID)
		}
	})

	t.Run("unknown board", func(t *testing.T) {
		_, _, err := st.FindSensorSlot(ctx, "10.252.176.99", 1)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("slot number out of range", func(t *testing.T) {
		_, _, err := st.FindSensorSlot(ctx, "10.252.176.50", 9)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

type eventRow struct {
	typ    string
	gunID  int64
	slotID int64
}

func setupSensorFixture(t *testing.T) (*Store, models.Locker, models.Gun) {
	t.Helper()
	ctx := context.Background()
	st := newTestStore(t)
	locker, err := st.CreateLocker(ctx, models.Locker{Name: "A", IPAddress: "10.252.176.50"})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := st.CreateCategory(ctx, "Rifle")
	if err != nil {
		t.Fatal(err)
	}
	gun, err := st.CreateGun(ctx, models.Gun{SlotID: locker.Slots[1].ID, CategoryID: cat.ID, Serial: "SN1"})
	if err != nil {
		t.Fatal(err)
	}
	return st, locker, gun
}

func listEvents(t *testing.T, st *Store) []eventRow {
	t.Helper()
	rows, err := st.db.Query("SELECT type, COALESCE(gun_id, 0), slot_id FROM events ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []eventRow
	for rows.Next() {
		var e eventRow
		if err := rows.Scan(&e.typ, &e.gunID, &e.slotID); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

func gunStatus(t *testing.T, st *Store, id int64) string {
	t.Helper()
	g, err := st.GetGun(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return g.Status
}

func TestRecordSensorEventRemovesGun(t *testing.T) {
	st, locker, gun := setupSensorFixture(t)
	slotID := locker.Slots[1].ID

	if err := st.RecordSensorEvent(context.Background(), slotID, gun.ID, "gun_removed", "out"); err != nil {
		t.Fatal(err)
	}

	events := listEvents(t, st)
	want := eventRow{typ: "gun_removed", gunID: gun.ID, slotID: slotID}
	if len(events) != 1 || events[0] != want {
		t.Fatalf("events = %+v, want [%+v]", events, want)
	}
	if s := gunStatus(t, st, gun.ID); s != "out" {
		t.Fatalf("gun status = %q, want out", s)
	}
}

func TestRecordSensorEventReturnsGun(t *testing.T) {
	st, locker, gun := setupSensorFixture(t)
	ctx := context.Background()
	slotID := locker.Slots[1].ID

	if err := st.RecordSensorEvent(ctx, slotID, gun.ID, "gun_removed", "out"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordSensorEvent(ctx, slotID, gun.ID, "gun_returned", "in"); err != nil {
		t.Fatal(err)
	}

	if n := len(listEvents(t, st)); n != 2 {
		t.Fatalf("got %d events, want 2", n)
	}
	if s := gunStatus(t, st, gun.ID); s != "in" {
		t.Fatalf("gun status = %q, want in", s)
	}
}

func TestRecordSensorEventFaultKeepsStatus(t *testing.T) {
	st, locker, gun := setupSensorFixture(t)

	if err := st.RecordSensorEvent(context.Background(), locker.Slots[1].ID, gun.ID, "sensor_fault", ""); err != nil {
		t.Fatal(err)
	}

	if events := listEvents(t, st); len(events) != 1 || events[0].typ != "sensor_fault" {
		t.Fatalf("events = %+v, want one sensor_fault", events)
	}
	if s := gunStatus(t, st, gun.ID); s != "in" {
		t.Fatalf("gun status = %q, want unchanged in", s)
	}
}

func TestRecordSensorEventEmptySlot(t *testing.T) {
	st, locker, _ := setupSensorFixture(t)
	slotID := locker.Slots[0].ID

	if err := st.RecordSensorEvent(context.Background(), slotID, 0, "gun_returned", "in"); err != nil {
		t.Fatal(err)
	}

	events := listEvents(t, st)
	want := eventRow{typ: "gun_returned", gunID: 0, slotID: slotID}
	if len(events) != 1 || events[0] != want {
		t.Fatalf("events = %+v, want [%+v]", events, want)
	}
}

func TestRecordSensorEventRetiredGunUntouched(t *testing.T) {
	st, locker, gun := setupSensorFixture(t)
	ctx := context.Background()
	if _, err := st.RetireGun(ctx, gun.ID); err != nil {
		t.Fatal(err)
	}

	if err := st.RecordSensorEvent(ctx, locker.Slots[1].ID, gun.ID, "gun_returned", "in"); err != nil {
		t.Fatal(err)
	}
	if s := gunStatus(t, st, gun.ID); s != "retired" {
		t.Fatalf("gun status = %q, want retired", s)
	}
}

func TestRecordSensorEventRollsBackOnError(t *testing.T) {
	st, _, gun := setupSensorFixture(t)

	err := st.RecordSensorEvent(context.Background(), 9999, gun.ID, "gun_removed", "out")
	if err == nil {
		t.Fatal("want error for unknown slot, got nil")
	}
	if n := len(listEvents(t, st)); n != 0 {
		t.Fatalf("got %d events after failed call, want 0", n)
	}
	if s := gunStatus(t, st, gun.ID); s != "in" {
		t.Fatalf("gun status = %q, want in", s)
	}
}
