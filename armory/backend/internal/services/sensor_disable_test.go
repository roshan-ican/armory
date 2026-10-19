package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"armory/internal/database"
	"armory/internal/models"
)

func TestDisabledSensor(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)

	locker, err := store.CreateLocker(ctx, models.Locker{Name: "A", IPAddress: "10.252.176.50", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	lockers := NewLockerService(store)
	sensors := NewSensorService(store, nil)
	activity := NewActivityService(store)

	if err := sensors.Sync(ctx, "10.252.176.50", 1, 1, "gun_returned"); err != nil {
		t.Fatal(err)
	}
	if err := sensors.Sync(ctx, "10.252.176.50", 2, 1, "gun_returned"); err != nil {
		t.Fatal(err)
	}

	l, err := lockers.SetSensorDisabled(ctx, locker.ID, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if !l.Slots[0].SensorDisabled || l.Slots[0].Reading != 2 || l.Slots[1].SensorDisabled {
		t.Fatalf("slots after disable = %+v", l.Slots)
	}

	if err := sensors.Record(ctx, "10.252.176.50", 1, 0, "gun_removed"); err != nil {
		t.Fatalf("record on disabled slot = %v, want nil", err)
	}
	if err := sensors.Sync(ctx, "10.252.176.50", 1, 0, "gun_removed"); err != nil {
		t.Fatalf("sync on disabled slot = %v, want nil", err)
	}
	if err := sensors.Record(ctx, "10.252.176.50", 2, 0, "gun_removed"); err != nil {
		t.Fatal(err)
	}

	events, err := activity.Recent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != "gun_removed" {
		t.Fatalf("events = %+v, want only slot 2's removal", events)
	}

	l, err = lockers.Get(ctx, locker.ID)
	if err != nil {
		t.Fatal(err)
	}
	if l.Slots[0].Reading != 2 || l.Slots[1].Reading != 0 {
		t.Fatalf("readings = %d, %d, want 2, 0", l.Slots[0].Reading, l.Slots[1].Reading)
	}

	l, err = lockers.SetSensorDisabled(ctx, locker.ID, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if l.Slots[0].SensorDisabled || l.Slots[0].Reading != -1 {
		t.Fatalf("slot after enable = %+v", l.Slots[0])
	}
	if err := sensors.Record(ctx, "10.252.176.50", 1, 1, "gun_returned"); err != nil {
		t.Fatal(err)
	}
	l, _ = lockers.Get(ctx, locker.ID)
	if l.Slots[0].Reading != 1 {
		t.Fatalf("reading after enable = %d, want 1", l.Slots[0].Reading)
	}

	if _, err := lockers.SetSensorDisabled(ctx, locker.ID, 4, true); !errors.Is(err, ErrInvalidSlot) {
		t.Fatalf("slot 4 err = %v, want ErrInvalidSlot", err)
	}
}

func TestSwapCarriesDisabledFlag(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)

	locker, err := store.CreateLocker(ctx, models.Locker{Name: "A", IPAddress: "10.252.176.50", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	lockers := NewLockerService(store)
	if _, err := lockers.SetSensorDisabled(ctx, locker.ID, 1, true); err != nil {
		t.Fatal(err)
	}
	l, err := lockers.SwapSensors(ctx, locker.ID, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if l.Slots[0].SensorDisabled || !l.Slots[1].SensorDisabled {
		t.Fatalf("after swap = %+v", l.Slots)
	}
}
