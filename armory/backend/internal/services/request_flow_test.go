package services

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/models"
)

type fakeHardware struct {
	unlocks []string
	signals []string
	alarms  []string
}

func (f *fakeHardware) Unlock(ctx context.Context, lockerIP string, slotNo int64) error {
	f.unlocks = append(f.unlocks, lockerIP+"#"+string(rune('0'+slotNo)))
	return nil
}

func (f *fakeHardware) OpenDoor(ctx context.Context) error { return nil }

func (f *fakeHardware) Signal(ctx context.Context, lockerIP string, signal Signal) error {
	f.signals = append(f.signals, lockerIP+":"+string(signal))
	return nil
}

func (f *fakeHardware) Alarm(ctx context.Context, lockerIP string, on bool) error {
	f.alarms = append(f.alarms, lockerIP+":"+strconv.FormatBool(on))
	return nil
}

type flow struct {
	svc    *RequestService
	store  *database.Store
	hw     *fakeHardware
	hub    *live.Hub
	user   models.User
	other  models.User
	admin  models.User
	locker models.Locker
}

func newFlow(t *testing.T) flow {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)

	mk := func(name, no, role string) models.User {
		u, err := store.CreateUser(ctx, models.User{Name: name, ServiceNo: no, Role: role})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	locker, err := store.CreateLocker(ctx, models.Locker{Name: "East", IPAddress: "10.0.0.5", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, sl := range locker.Slots {
		if err := store.RecordSensorEvent(ctx, sl.ID, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
	}
	hw := &fakeHardware{}
	hub := live.NewHub()
	return flow{
		svc: NewRequestService(store, hw, hub), store: store, hw: hw, hub: hub,
		user: mk("Roshan Sahani", "SN-1", "requester"), other: mk("Amit", "SN-2", "requester"),
		admin: mk("Admin", "AD-1", "admin"), locker: locker,
	}
}

func TestRequestApproveRules(t *testing.T) {
	ctx := context.Background()

	t.Run("an admin approves, a slot is chosen and the locker is unlocked", func(t *testing.T) {
		f := newFlow(t)
		req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.svc.Approve(ctx, f.admin.ID, req.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != models.RequestApproved || firstSlotNo(got) != 1 || got.LockerName != "East" {
			t.Fatalf("got %+v", got)
		}
		if !reflect.DeepEqual(f.hw.unlocks, []string{"10.0.0.5#1"}) {
			t.Fatalf("unlocks = %v", f.hw.unlocks)
		}
	})

	t.Run("a requester cannot approve or reject", func(t *testing.T) {
		f := newFlow(t)
		req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Approve(ctx, f.other.ID, req.ID); !errors.Is(err, ErrNotAdmin) {
			t.Fatalf("approve: err = %v, want ErrNotAdmin", err)
		}
		if err := f.svc.Reject(ctx, f.user.ID, req.ID); !errors.Is(err, ErrNotAdmin) {
			t.Fatalf("reject: err = %v, want ErrNotAdmin", err)
		}
		if _, err := f.svc.Approve(ctx, 9999, req.ID); !errors.Is(err, ErrNotAdmin) {
			t.Fatalf("unknown admin: err = %v, want ErrNotAdmin", err)
		}
		if len(f.hw.unlocks) != 0 {
			t.Fatalf("nothing may unlock, got %v", f.hw.unlocks)
		}
	})

	t.Run("database errors become readable ones", func(t *testing.T) {
		f := newFlow(t)
		if _, err := f.svc.Approve(ctx, f.admin.ID, 9999); !errors.Is(err, ErrRequestNotFound) {
			t.Fatalf("missing: err = %v", err)
		}
		req, err := f.svc.Create(ctx, f.user.ID, "pistol", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); !errors.Is(err, ErrNoneAvailable) {
			t.Fatalf("no pistol locker: err = %v, want ErrNoneAvailable", err)
		}
		if err := f.svc.Reject(ctx, f.admin.ID, req.ID); err != nil {
			t.Fatal(err)
		}
		if err := f.svc.Reject(ctx, f.admin.ID, req.ID); !errors.Is(err, ErrRequestClosed) {
			t.Fatalf("second reject: err = %v, want ErrRequestClosed", err)
		}
	})

	t.Run("only the owner can cancel", func(t *testing.T) {
		f := newFlow(t)
		req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := f.svc.Cancel(ctx, f.other.ID, req.ID); !errors.Is(err, ErrRequestClosed) {
			t.Fatalf("other user: err = %v", err)
		}
		if err := f.svc.Cancel(ctx, f.user.ID, req.ID); err != nil {
			t.Fatal(err)
		}
		if err := f.svc.Cancel(ctx, f.user.ID, 9999); !errors.Is(err, ErrRequestNotFound) {
			t.Fatalf("missing: err = %v", err)
		}
	})

	t.Run("the lists split waiting from decided", func(t *testing.T) {
		f := newFlow(t)
		first, _ := f.svc.Create(ctx, f.user.ID, "rifle", "")
		if _, err := f.svc.Create(ctx, f.other.ID, "rifle", ""); err != nil {
			t.Fatal(err)
		}
		if err := f.svc.Reject(ctx, f.admin.ID, first.ID); err != nil {
			t.Fatal(err)
		}
		pending, err := f.svc.Pending(ctx)
		if err != nil || len(pending) != 1 || pending[0].UserName != "Amit" {
			t.Fatalf("pending = %+v, %v", pending, err)
		}
		recent, err := f.svc.Recent(ctx)
		if err != nil || len(recent) != 1 || recent[0].ID != first.ID {
			t.Fatalf("recent = %+v, %v", recent, err)
		}
	})
}

func TestRequestViews(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)

	if _, err := f.svc.Open(ctx, f.user.ID); !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("no request yet: err = %v", err)
	}
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("pending has no slots", func(t *testing.T) {
		v, err := f.svc.Open(ctx, f.user.ID)
		if err != nil || v.Request.ID != req.ID || len(v.Slots) != 0 {
			t.Fatalf("got %+v, %v", v, err)
		}
	})

	t.Run("someone else cannot look at it", func(t *testing.T) {
		if _, err := f.svc.View(ctx, f.other.ID, req.ID); !errors.Is(err, ErrRequestNotFound) {
			t.Fatalf("err = %v, want ErrRequestNotFound", err)
		}
	})

	t.Run("approved shows the locker's slots", func(t *testing.T) {
		if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
			t.Fatal(err)
		}
		v, err := f.svc.View(ctx, f.user.ID, req.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(v.Slots) != 3 || firstSlotNo(v.Request) != 1 || len(v.Wrong) != 0 {
			t.Fatalf("got %d slots, target %d, wrong %v", len(v.Slots), firstSlotNo(v.Request), v.Wrong)
		}
	})

	t.Run("availability still offers a gun nobody has collected yet", func(t *testing.T) {
		got, err := f.svc.Availability(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got["rifle"] != 3 || got["pistol"] != 0 {
			t.Fatalf("got %v", got)
		}
	})
}

func TestSlotChangedFollowsTheGun(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	approved, err := f.svc.Approve(ctx, f.admin.ID, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	target := firstSlotID(approved)
	wrongSlot := f.locker.Slots[2].ID

	statusOf := func() string {
		r, err := f.store.GetRequest(ctx, req.ID)
		if err != nil {
			t.Fatal(err)
		}
		return r.Status
	}
	setReading := func(slotID int64, reading byte) {
		if err := f.store.RecordSensorEvent(ctx, slotID, reading, "gun_removed"); err != nil {
			t.Fatal(err)
		}
		f.svc.SlotChanged(ctx, slotID, reading)
	}

	t.Run("a wrong gun leaves the request approved and raises the wrong alarm", func(t *testing.T) {
		setReading(wrongSlot, 0)
		if got := statusOf(); got != models.RequestApproved {
			t.Fatalf("status = %q, want approved", got)
		}
		if !reflect.DeepEqual(f.hw.alarms, []string{"10.0.0.5:true"}) {
			t.Fatalf("alarms = %v", f.hw.alarms)
		}
		if len(f.hw.signals) != 0 {
			t.Fatalf("a wrong gun must not send the right signal, got %v", f.hw.signals)
		}
		v, err := f.svc.View(ctx, f.user.ID, req.ID)
		if err != nil || !reflect.DeepEqual(v.Wrong, []int64{3}) {
			t.Fatalf("wrong = %v, %v", v.Wrong, err)
		}
	})

	t.Run("putting the wrong gun back clears the warning", func(t *testing.T) {
		setReading(wrongSlot, 1)
		v, err := f.svc.View(ctx, f.user.ID, req.ID)
		if err != nil || len(v.Wrong) != 0 {
			t.Fatalf("wrong = %v, %v", v.Wrong, err)
		}
		if got := statusOf(); got != models.RequestApproved {
			t.Fatalf("status = %q", got)
		}
		if !reflect.DeepEqual(f.hw.alarms, []string{"10.0.0.5:true", "10.0.0.5:false"}) {
			t.Fatalf("alarms = %v, want it raised then cleared", f.hw.alarms)
		}
	})

	t.Run("the right gun marks it collected and sounds the correct signal", func(t *testing.T) {
		setReading(target, 0)
		if got := statusOf(); got != models.RequestCollected {
			t.Fatalf("status = %q, want collected", got)
		}
		want := []string{"10.0.0.5:correct"}
		if !reflect.DeepEqual(f.hw.signals, want) {
			t.Fatalf("signals = %v, want %v", f.hw.signals, want)
		}
	})

	t.Run("bringing it back marks it returned", func(t *testing.T) {
		setReading(target, 1)
		if got := statusOf(); got != models.RequestReturned {
			t.Fatalf("status = %q, want returned", got)
		}
	})

	t.Run("a gun leaving with no approved request makes no sound", func(t *testing.T) {
		before := len(f.hw.signals)
		setReading(f.locker.Slots[1].ID, 0)
		if len(f.hw.signals) != before {
			t.Fatalf("signals = %v", f.hw.signals)
		}
	})
}

func TestCatalogNamesWhoHoldsAGun(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	approved, err := f.svc.Approve(ctx, f.admin.ID, req.ID)
	if err != nil {
		t.Fatal(err)
	}

	holders := func() map[int64]string {
		items, err := f.svc.Catalog(ctx)
		if err != nil {
			t.Fatal(err)
		}
		out := map[int64]string{}
		for _, sl := range items[0].Slots {
			out[sl.No] = sl.TakenBy
		}
		return out
	}
	move := func(reading byte) {
		if err := f.store.RecordSensorEvent(ctx, firstSlotID(approved), reading, "gun_moved"); err != nil {
			t.Fatal(err)
		}
		f.svc.SlotChanged(ctx, firstSlotID(approved), reading)
	}

	if got := holders(); got[firstSlotNo(approved)] != "" || got[2] != "" || got[3] != "" {
		t.Fatalf("approved: holders = %v, want nobody until it is collected", got)
	}
	move(0)
	if got := holders(); got[firstSlotNo(approved)] != "Roshan Sahani" {
		t.Fatalf("taken: holders = %v", got)
	}
	move(1)
	if got := holders(); got[firstSlotNo(approved)] != "" {
		t.Fatalf("returned: holders = %v, want nobody", got)
	}
	items, err := f.svc.Catalog(ctx)
	if err != nil || items[0].Available != 3 {
		t.Fatalf("available = %d, err = %v, want all 3 back", items[0].Available, err)
	}
}

func firstSlotID(r models.Request) int64 {
	if len(r.Slots) == 0 {
		return 0
	}
	return r.Slots[0].SlotID
}

func firstSlotNo(r models.Request) int64 {
	if len(r.Slots) == 0 {
		return 0
	}
	return r.Slots[0].SlotNo
}

func TestReopenResendsOpenWhileGunIsStillThere(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	if len(f.hw.unlocks) != 1 {
		t.Fatalf("approval unlocks = %v", f.hw.unlocks)
	}

	if _, err := f.svc.Reopen(ctx, f.user.ID); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.hw.unlocks, []string{"10.0.0.5#1", "10.0.0.5#1"}) {
		t.Fatalf("gun still there: unlocks = %v", f.hw.unlocks)
	}

	if _, err := f.svc.Reopen(ctx, f.user.ID); err != nil {
		t.Fatal(err)
	}
	if len(f.hw.unlocks) != 3 {
		t.Fatalf("every login resends: unlocks = %v", f.hw.unlocks)
	}

	if err := f.store.RecordSensorEvent(ctx, f.locker.Slots[0].ID, 0, "gun_taken"); err != nil {
		t.Fatal(err)
	}
	before := len(f.hw.unlocks)
	if _, err := f.svc.Reopen(ctx, f.user.ID); err != nil {
		t.Fatal(err)
	}
	if len(f.hw.unlocks) != before {
		t.Fatalf("gun gone, must not open again: unlocks = %v", f.hw.unlocks)
	}
}

func TestReopenDoesNothingWithoutAnApprovedRequest(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	if _, err := f.svc.Create(ctx, f.user.ID, "rifle", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Reopen(ctx, f.user.ID); err != nil {
		t.Fatal(err)
	}
	if len(f.hw.unlocks) != 0 {
		t.Fatalf("pending request must not unlock: %v", f.hw.unlocks)
	}
}

func TestNewRequesterSupersedesAnUncollectedApproval(t *testing.T) {
	ctx := context.Background()
	slot1 := func(f flow) int64 { return f.locker.Slots[0].ID }

	status := func(t *testing.T, f flow, id int64) string {
		t.Helper()
		r, err := f.store.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return r.Status
	}
	ask := func(t *testing.T, f flow, userID int64, slots ...int64) models.Request {
		t.Helper()
		r, err := f.svc.CreateForLocker(ctx, userID, f.locker.ID, slots, "")
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	takeGun := func(t *testing.T, f flow, slotID int64, reading byte) {
		t.Helper()
		if err := f.store.RecordSensorEvent(ctx, slotID, reading, "gun_moved"); err != nil {
			t.Fatal(err)
		}
		f.svc.SlotChanged(ctx, slotID, reading)
	}

	t.Run("the first requester never came, the second one gets the gun", func(t *testing.T) {
		f := newFlow(t)
		a := ask(t, f, f.user.ID, 1)
		if _, err := f.svc.Approve(ctx, f.admin.ID, a.ID); err != nil {
			t.Fatal(err)
		}
		b := ask(t, f, f.other.ID, 1)
		if _, err := f.svc.Approve(ctx, f.admin.ID, b.ID); err != nil {
			t.Fatal(err)
		}
		if got := status(t, f, a.ID); got != models.RequestExpired {
			t.Fatalf("first request = %q, want expired", got)
		}
		takeGun(t, f, slot1(f), 0)
		takeGun(t, f, slot1(f), 1)
		if got := status(t, f, a.ID); got != models.RequestExpired {
			t.Fatalf("first request = %q, it must not become returned", got)
		}
		if got := status(t, f, b.ID); got != models.RequestReturned {
			t.Fatalf("second request = %q, want returned", got)
		}
		events, err := f.store.ListEvents(ctx, 20)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range events {
			if e.Type == "request_expired" && e.Details == "did not collect, replaced by Amit" {
				found = true
			}
		}
		if !found {
			t.Fatalf("no request_expired event in %+v", events)
		}
	})

	t.Run("a different gun in the same locker also suspends the uncollected request", func(t *testing.T) {
		f := newFlow(t)
		a := ask(t, f, f.user.ID, 3)
		if _, err := f.svc.Approve(ctx, f.admin.ID, a.ID); err != nil {
			t.Fatal(err)
		}
		b := ask(t, f, f.other.ID, 1)
		if _, err := f.svc.Approve(ctx, f.admin.ID, b.ID); err != nil {
			t.Fatal(err)
		}
		if got := status(t, f, a.ID); got != models.RequestExpired {
			t.Fatalf("first request = %q, want expired", got)
		}
		takeGun(t, f, f.locker.Slots[2].ID, 0)
		if len(f.hw.signals) != 0 {
			t.Fatalf("taking the suspended request's gun must not signal correct, got %v", f.hw.signals)
		}
		if got := status(t, f, b.ID); got != models.RequestApproved {
			t.Fatalf("second request = %q, want approved", got)
		}
		if got := status(t, f, a.ID); got != models.RequestExpired {
			t.Fatalf("first request = %q, it must not become collected", got)
		}
		takeGun(t, f, slot1(f), 0)
		if got := status(t, f, b.ID); got != models.RequestCollected {
			t.Fatalf("second request = %q, want collected", got)
		}
	})

	t.Run("a gun the first requester already took cannot be given away", func(t *testing.T) {
		f := newFlow(t)
		a := ask(t, f, f.user.ID, 1, 2)
		if _, err := f.svc.Approve(ctx, f.admin.ID, a.ID); err != nil {
			t.Fatal(err)
		}
		takeGun(t, f, slot1(f), 0)
		if _, err := f.svc.CreateForLocker(ctx, f.other.ID, f.locker.ID, []int64{1}, ""); !errors.Is(err, ErrGunUnavailable) {
			t.Fatalf("taken gun: err = %v, want ErrGunUnavailable", err)
		}
		if _, err := f.svc.CreateForLocker(ctx, f.other.ID, f.locker.ID, []int64{2}, ""); !errors.Is(err, ErrGunUnavailable) {
			t.Fatalf("gun held by a partly collected request: err = %v, want ErrGunUnavailable", err)
		}
		if got := status(t, f, a.ID); got != models.RequestApproved {
			t.Fatalf("first request = %q, want approved", got)
		}
	})
}
