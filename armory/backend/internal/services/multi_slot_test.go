package services

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"armory/internal/models"
)

func (f flow) sensor(t *testing.T, slotNo int64, reading byte) {
	t.Helper()
	ctx := context.Background()
	slotID := f.locker.Slots[slotNo-1].ID
	if err := f.store.RecordSensorEvent(ctx, slotID, reading, "gun_moved"); err != nil {
		t.Fatal(err)
	}
	f.svc.SlotChanged(ctx, slotID, reading)
}

func (f flow) status(t *testing.T, id int64) (string, map[int64]string) {
	t.Helper()
	got, err := f.store.GetRequest(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	slots := map[int64]string{}
	for _, sl := range got.Slots {
		slots[sl.SlotNo] = sl.Status
	}
	return got.Status, slots
}

func TestChosenGunsAreTheOnesReserved(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)

	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{3}, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.Approve(ctx, f.admin.ID, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.SlotNos(), []int64{3}) {
		t.Fatalf("approved slots %v, want [3]", got.SlotNos())
	}
	if !reflect.DeepEqual(f.hw.unlocks, []string{"10.0.0.5#3"}) {
		t.Fatalf("unlocks %v", f.hw.unlocks)
	}

	f.sensor(t, 3, 0)
	status, slots := f.status(t, req.ID)
	if status != models.RequestCollected || slots[3] != models.SlotCollected {
		t.Fatalf("after taking gun 3: %s %v", status, slots)
	}
	if !reflect.DeepEqual(f.hw.signals, []string{"10.0.0.5:correct"}) {
		t.Fatalf("signals %v", f.hw.signals)
	}
	wrong, err := f.store.WrongSlots(ctx, req.ID)
	if err != nil || len(wrong) != 0 {
		t.Fatalf("wrong slots %v, %v", wrong, err)
	}
}

func TestSeveralGunsInOneRequest(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)

	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{3, 1, 3}, "drill")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(req.SlotNos(), []int64{1, 3}) || req.SlotList() != "1, 3" {
		t.Fatalf("pending slots %v %q", req.SlotNos(), req.SlotList())
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := f.store.CountAvailableInLocker(ctx, f.locker.ID); n != 3 {
		t.Fatalf("available after approval = %d, want 3 until a gun is collected", n)
	}

	f.sensor(t, 1, 0)
	if status, slots := f.status(t, req.ID); status != models.RequestApproved || slots[1] != models.SlotCollected || slots[3] != models.SlotChosen {
		t.Fatalf("after gun 1: %s %v", status, slots)
	}

	f.sensor(t, 2, 0)
	wrong, err := f.store.WrongSlots(ctx, req.ID)
	if err != nil || !reflect.DeepEqual(wrong, []int64{2}) {
		t.Fatalf("taking gun 2: wrong %v, %v", wrong, err)
	}
	f.sensor(t, 2, 1)

	f.sensor(t, 3, 0)
	if status, _ := f.status(t, req.ID); status != models.RequestCollected {
		t.Fatalf("after both guns: %s", status)
	}

	f.sensor(t, 3, 1)
	if status, slots := f.status(t, req.ID); status != models.RequestCollected || slots[3] != models.SlotReturned {
		t.Fatalf("after returning gun 3: %s %v", status, slots)
	}
	if n, _ := f.store.CountAvailableInLocker(ctx, f.locker.ID); n != 2 {
		t.Fatalf("available with one gun still out = %d, want 2", n)
	}

	f.sensor(t, 1, 1)
	if status, _ := f.status(t, req.ID); status != models.RequestReturned {
		t.Fatalf("after returning both: %s", status)
	}
	if open, _ := f.store.HasOpenRequest(ctx, f.user.ID); open {
		t.Fatal("request still open after everything was returned")
	}
}

func TestChosenGunRules(t *testing.T) {
	ctx := context.Background()

	t.Run("a gun number outside the locker is refused", func(t *testing.T) {
		f := newFlow(t)
		if _, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{4}, ""); !errors.Is(err, ErrInvalidSlot) {
			t.Fatalf("err = %v, want ErrInvalidSlot", err)
		}
	})

	t.Run("a missing gun cannot be chosen", func(t *testing.T) {
		f := newFlow(t)
		f.sensor(t, 2, 0)
		if _, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{2}, ""); !errors.Is(err, ErrGunUnavailable) {
			t.Fatalf("err = %v, want ErrGunUnavailable", err)
		}
		if open, _ := f.store.HasOpenRequest(ctx, f.user.ID); open {
			t.Fatal("a refused request was left behind")
		}
	})

	t.Run("two people ask for the same gun, approving the second replaces the first", func(t *testing.T) {
		f := newFlow(t)
		first, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{2}, "")
		if err != nil {
			t.Fatal(err)
		}
		second, err := f.svc.CreateForLocker(ctx, f.other.ID, f.locker.ID, []int64{2, 3}, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Approve(ctx, f.admin.ID, first.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Approve(ctx, f.admin.ID, second.ID); err != nil {
			t.Fatal(err)
		}
		if status, _ := f.status(t, first.ID); status != models.RequestExpired {
			t.Fatalf("first request status %s, want expired", status)
		}
		if status, _ := f.status(t, second.ID); status != models.RequestApproved {
			t.Fatalf("second request status %s, want approved", status)
		}
	})

	t.Run("a request without chosen guns still gets the first free one", func(t *testing.T) {
		f := newFlow(t)
		req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, nil, "")
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.svc.Approve(ctx, f.admin.ID, req.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.SlotNos(), []int64{1}) {
			t.Fatalf("slots %v, want [1]", got.SlotNos())
		}
	})

	t.Run("the catalog marks a collected gun unavailable", func(t *testing.T) {
		f := newFlow(t)
		req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{1}, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
			t.Fatal(err)
		}
		if err := f.store.MarkLockerSeen(ctx, "10.0.0.5"); err != nil {
			t.Fatal(err)
		}
		avail := func() map[int64]bool {
			items, err := f.svc.Catalog(ctx)
			if err != nil {
				t.Fatal(err)
			}
			got := map[int64]bool{}
			for _, sl := range items[0].Slots {
				got[sl.No] = sl.Available
			}
			return got
		}
		if got := avail(); !got[1] || !got[2] || !got[3] {
			t.Fatalf("available %v, want all three until the gun is collected", got)
		}
		f.sensor(t, 1, 0)
		if got := avail(); got[1] || !got[2] || !got[3] {
			t.Fatalf("available %v, want only 2 and 3", got)
		}
	})
}
