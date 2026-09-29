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

func newTestLocker(t *testing.T, st *Store, capacity int64) models.Locker {
	t.Helper()
	l, err := st.CreateLocker(context.Background(), models.Locker{Name: "A", IPAddress: "10.252.176.50", Kind: "rifle", Capacity: capacity})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func slotReading(t *testing.T, st *Store, slotID int64) int64 {
	t.Helper()
	var r int64
	if err := st.db.QueryRow("SELECT COALESCE(reading, -1) FROM slots WHERE id = ?", slotID).Scan(&r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestFindSensorSlot(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	locker := newTestLocker(t, st, 3)

	t.Run("known slot", func(t *testing.T) {
		got, err := st.FindSensorSlot(ctx, "10.252.176.50", 2)
		if err != nil {
			t.Fatal(err)
		}
		if got != locker.Slots[1].ID {
			t.Fatalf("got slot %d, want %d", got, locker.Slots[1].ID)
		}
	})

	t.Run("unknown board", func(t *testing.T) {
		if _, err := st.FindSensorSlot(ctx, "10.252.176.99", 1); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("slot beyond capacity", func(t *testing.T) {
		if _, err := st.FindSensorSlot(ctx, "10.252.176.50", 4); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("slot removed by shrinking", func(t *testing.T) {
		locker.Capacity = 2
		if _, err := st.UpdateLocker(ctx, locker); err != nil {
			t.Fatal(err)
		}
		if _, err := st.FindSensorSlot(ctx, "10.252.176.50", 3); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestRecordSensorEvent(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	locker := newTestLocker(t, st, 3)
	slot := locker.Slots[1].ID

	if err := st.RecordSensorEvent(ctx, slot, 0, "gun_removed"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordSensorEvent(ctx, slot, 1, "gun_returned"); err != nil {
		t.Fatal(err)
	}
	if got := slotReading(t, st, slot); got != 1 {
		t.Fatalf("reading = %d, want 1", got)
	}

	events, err := st.ListEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Type != "gun_returned" || events[1].Type != "gun_removed" {
		t.Fatalf("order = %q, %q; want newest first", events[0].Type, events[1].Type)
	}
	if events[0].LockerName != "A" || events[0].SlotNo != 2 {
		t.Fatalf("got locker %q slot %d", events[0].LockerName, events[0].SlotNo)
	}
}

func TestRecordSensorEventUnknownSlotSavesNothing(t *testing.T) {
	st := newTestStore(t)
	if err := st.RecordSensorEvent(context.Background(), 9999, 0, "gun_removed"); err == nil {
		t.Fatal("want error for unknown slot")
	}
	events, err := st.ListEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestSyncSensorSlot(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	slot := newTestLocker(t, st, 3).Slots[0].ID

	count := func() int {
		events, err := st.ListEvents(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		return len(events)
	}

	t.Run("first reading is stored without an event", func(t *testing.T) {
		if err := st.SyncSensorSlot(ctx, slot, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
		if got := slotReading(t, st, slot); got != 1 {
			t.Fatalf("reading = %d, want 1", got)
		}
		if n := count(); n != 0 {
			t.Fatalf("got %d events, want 0", n)
		}
	})

	t.Run("same reading adds nothing", func(t *testing.T) {
		if err := st.SyncSensorSlot(ctx, slot, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
		if n := count(); n != 0 {
			t.Fatalf("got %d events, want 0", n)
		}
	})

	t.Run("changed reading is logged", func(t *testing.T) {
		if err := st.SyncSensorSlot(ctx, slot, 0, "gun_removed"); err != nil {
			t.Fatal(err)
		}
		events, err := st.ListEvents(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 || events[0].Type != "gun_removed" || events[0].Details == "" {
			t.Fatalf("events = %+v", events)
		}
	})

	t.Run("unknown slot", func(t *testing.T) {
		if err := st.SyncSensorSlot(ctx, 9999, 1, "gun_returned"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
