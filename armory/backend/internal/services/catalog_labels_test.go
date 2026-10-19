package services

import (
	"context"
	"testing"

	"armory/internal/models"
)

func catalogSlots(t *testing.T, f flow) []CatalogSlot {
	t.Helper()
	items, err := f.svc.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.ID == f.locker.ID {
			return item.Slots
		}
	}
	t.Fatal("locker not in the catalog")
	return nil
}

func TestCatalogSaysRequestedWhileTheGunIsStillThereAndTakenOnlyOnceItIsGone(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	f.store.MarkLockerSeen(ctx, "10.0.0.5")
	a := askFor(t, f, f.user.ID, 1, 2, 3)
	if _, err := f.svc.Approve(ctx, f.admin.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	who := f.user.Name

	t.Run("all three requested and still in place", func(t *testing.T) {
		for _, sl := range catalogSlots(t, f) {
			if sl.RequestedBy != who || sl.TakenBy != "" || sl.Reading != 1 || !sl.Available {
				t.Fatalf("slot %d = %+v, want requested by %q, not taken, green and selectable", sl.No, sl, who)
			}
		}
	})

	moveGun(t, f, firstSlotOf(f), 0)

	t.Run("one physically taken, the other two are still only requested", func(t *testing.T) {
		got := catalogSlots(t, f)
		if got[0].TakenBy != who || got[0].RequestedBy != "" || got[0].Reading != 0 || got[0].Available {
			t.Fatalf("slot 1 = %+v, want taken by %q and red", got[0], who)
		}
		for _, sl := range got[1:] {
			if sl.TakenBy != "" || sl.RequestedBy != who || sl.Reading != 1 || !sl.Available {
				t.Fatalf("slot %d = %+v, a gun still in place must not read as taken", sl.No, sl)
			}
		}
	})

	t.Run("returning the gun clears the label", func(t *testing.T) {
		moveGun(t, f, firstSlotOf(f), 1)
		if sl := catalogSlots(t, f)[0]; sl.TakenBy != "" {
			t.Fatalf("slot 1 = %+v, want no taken-by after it is back", sl)
		}
	})
}

func TestTakenByNeedsTheSensorToSayTheGunIsGone(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	f.store.MarkLockerSeen(ctx, "10.0.0.5")
	a := askFor(t, f, f.user.ID, 1)
	if _, err := f.svc.Approve(ctx, f.admin.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	moveGun(t, f, firstSlotOf(f), 0)
	if sl := catalogSlots(t, f)[0]; sl.TakenBy != f.user.Name {
		t.Fatalf("slot 1 = %+v, want taken once the sensor says it is gone", sl)
	}

	if err := f.store.RecordSensorEvent(ctx, firstSlotOf(f), 1, "gun_moved"); err != nil {
		t.Fatal(err)
	}
	if sl := catalogSlots(t, f)[0]; sl.TakenBy != "" || sl.Reading != 1 {
		t.Fatalf("slot 1 = %+v, the sensor says the gun is there so it must not read as taken", sl)
	}
}

func TestNewRequesterClaimsGunsThePreviousOneLeftInPlace(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	f.store.MarkLockerSeen(ctx, "10.0.0.5")
	a := askFor(t, f, f.user.ID, 1, 2, 3)
	if _, err := f.svc.Approve(ctx, f.admin.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	moveGun(t, f, firstSlotOf(f), 0)

	b := askFor(t, f, f.other.ID, 2, 3)
	if _, err := f.svc.Approve(ctx, f.admin.ID, b.ID); err != nil {
		t.Fatal(err)
	}

	got := catalogSlots(t, f)
	if got[0].TakenBy != f.user.Name {
		t.Fatalf("slot 1 = %+v, the gun that was really taken stays with the first person", got[0])
	}
	for _, sl := range got[1:] {
		if sl.RequestedBy != f.other.Name || sl.TakenBy != "" {
			t.Fatalf("slot %d = %+v, want requested by %q", sl.No, sl, f.other.Name)
		}
	}
}

func firstSlotOf(f flow) int64 { return f.locker.Slots[0].ID }

func askFor(t *testing.T, f flow, userID int64, slots ...int64) models.Request {
	t.Helper()
	r, err := f.svc.CreateForLocker(context.Background(), userID, f.locker.ID, slots, "")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func moveGun(t *testing.T, f flow, slotID int64, reading byte) {
	t.Helper()
	ctx := context.Background()
	if err := f.store.RecordSensorEvent(ctx, slotID, reading, "gun_moved"); err != nil {
		t.Fatal(err)
	}
	f.svc.SlotChanged(ctx, slotID, reading)
}
