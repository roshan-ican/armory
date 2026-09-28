package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"armory/internal/database"
	"armory/internal/models"
)

func TestSensorServiceRecord(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)

	locker, err := store.CreateLocker(ctx, models.Locker{Name: "A", IPAddress: "10.252.176.50"})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := store.CreateCategory(ctx, "Rifle")
	if err != nil {
		t.Fatal(err)
	}
	gun, err := store.CreateGun(ctx, models.Gun{SlotID: locker.Slots[1].ID, CategoryID: cat.ID, Serial: "SN1"})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewSensorService(store)

	t.Run("removes gun in slot 2", func(t *testing.T) {
		if err := svc.Record(ctx, "10.252.176.50", 2, 0, "gun_removed", "out"); err != nil {
			t.Fatal(err)
		}
		g, err := store.GetGun(ctx, gun.ID)
		if err != nil {
			t.Fatal(err)
		}
		if g.Status != "out" {
			t.Fatalf("status = %q, want out", g.Status)
		}
	})

	t.Run("sync shows fault on slot 2", func(t *testing.T) {
		if err := svc.Sync(ctx, "10.252.176.50", 2, 2, "sensor_fault", ""); err != nil {
			t.Fatal(err)
		}
		g, err := store.GetGun(ctx, gun.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !g.SensorFault {
			t.Fatal("want SensorFault after sync with reading 2")
		}
	})

	t.Run("unknown board", func(t *testing.T) {
		err := svc.Record(ctx, "10.252.176.99", 1, 0, "gun_removed", "out")
		if !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
