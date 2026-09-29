package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"armory/internal/database"
	"armory/internal/models"
)

func TestSensorService(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)

	if _, err := store.CreateLocker(ctx, models.Locker{Name: "A", IPAddress: "10.252.176.50", Kind: "rifle", Capacity: 5}); err != nil {
		t.Fatal(err)
	}
	svc := NewSensorService(store, nil)
	activity := NewActivityService(store)

	t.Run("sync then record builds a history", func(t *testing.T) {
		if err := svc.Sync(ctx, "10.252.176.50", 2, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
		if err := svc.Record(ctx, "10.252.176.50", 2, 0, "gun_removed"); err != nil {
			t.Fatal(err)
		}
		if err := svc.Record(ctx, "10.252.176.50", 2, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
		events, err := activity.Recent(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 2 || events[0].Type != "gun_returned" || events[1].Type != "gun_removed" {
			t.Fatalf("events = %+v", events)
		}
	})

	t.Run("unknown board", func(t *testing.T) {
		err := svc.Record(ctx, "10.252.176.99", 1, 0, "gun_removed")
		if !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
