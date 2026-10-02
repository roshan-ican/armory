package database

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"armory/internal/models"
)

type flowFixture struct {
	st     *Store
	user   models.User
	admin  models.User
	locker models.Locker
}

func newFlowFixture(t *testing.T) flowFixture {
	t.Helper()
	ctx := context.Background()
	st := newTestStore(t)
	user, err := st.CreateUser(ctx, models.User{Name: "Roshan Sahani", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := st.CreateUser(ctx, models.User{Name: "Admin", ServiceNo: "AD-1", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	locker, err := st.CreateLocker(ctx, models.Locker{Name: "East", IPAddress: "10.0.0.5", Kind: "rifle", Capacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	for _, sl := range locker.Slots {
		if _, err := st.db.Exec("UPDATE slots SET reading = 1 WHERE id = ?", sl.ID); err != nil {
			t.Fatal(err)
		}
	}
	return flowFixture{st: st, user: user, admin: admin, locker: locker}
}

func TestRequestCarriesLockerDetails(t *testing.T) {
	ctx := context.Background()
	f := newFlowFixture(t)
	id, err := f.st.CreateRequest(ctx, f.user.ID, "rifle", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.st.ApproveRequest(ctx, id, f.admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LockerID != f.locker.ID || got.LockerIP != "10.0.0.5" || got.LockerName != "East" {
		t.Fatalf("got locker %d %q %q", got.LockerID, got.LockerIP, got.LockerName)
	}
	if got.Initials() != "RS" {
		t.Fatalf("initials = %q", got.Initials())
	}
}

func TestLatestOpenRequest(t *testing.T) {
	ctx := context.Background()
	f := newFlowFixture(t)

	if _, err := f.st.LatestOpenRequest(ctx, f.user.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	id, err := f.st.CreateRequest(ctx, f.user.ID, "rifle", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.st.LatestOpenRequest(ctx, f.user.ID)
	if err != nil || got.ID != id {
		t.Fatalf("got %+v, %v", got, err)
	}
	if err := f.st.RejectRequest(ctx, id, f.admin.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.LatestOpenRequest(ctx, f.user.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a finished request must not be open, err = %v", err)
	}
}

func TestCollectAndReturnBySlot(t *testing.T) {
	ctx := context.Background()
	f := newFlowFixture(t)
	id, err := f.st.CreateRequest(ctx, f.user.ID, "rifle", "", "")
	if err != nil {
		t.Fatal(err)
	}
	approved, err := f.st.ApproveRequest(ctx, id, f.admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	other := f.locker.Slots[3].ID
	if firstSlotID(approved) == other {
		other = f.locker.Slots[2].ID
	}

	t.Run("a pending-return check on the wrong slot changes nothing", func(t *testing.T) {
		got, _, err := f.st.MarkCollected(ctx, other)
		if err != nil || got != 0 {
			t.Fatalf("got %d, %v", got, err)
		}
	})

	t.Run("the approved slot becomes collected once", func(t *testing.T) {
		got, _, err := f.st.MarkCollected(ctx, firstSlotID(approved))
		if err != nil || got != id {
			t.Fatalf("got %d, %v, want %d", got, err, id)
		}
		again, _, err := f.st.MarkCollected(ctx, firstSlotID(approved))
		if err != nil || again != 0 {
			t.Fatalf("second collect: %d, %v", again, err)
		}
		r, err := f.st.GetRequest(ctx, id)
		if err != nil || r.Status != models.RequestCollected {
			t.Fatalf("status = %q, %v", r.Status, err)
		}
		var stamp string
		if err := f.st.db.QueryRow("SELECT collected_at FROM requests WHERE id = ?", id).Scan(&stamp); err != nil || stamp == "" {
			t.Fatalf("collected_at = %q, %v", stamp, err)
		}
	})

	t.Run("a collected request becomes returned once", func(t *testing.T) {
		got, _, err := f.st.MarkReturned(ctx, firstSlotID(approved))
		if err != nil || got != id {
			t.Fatalf("got %d, %v, want %d", got, err, id)
		}
		again, _, err := f.st.MarkReturned(ctx, firstSlotID(approved))
		if err != nil || again != 0 {
			t.Fatalf("second return: %d, %v", again, err)
		}
		r, err := f.st.GetRequest(ctx, id)
		if err != nil || r.Status != models.RequestReturned {
			t.Fatalf("status = %q, %v", r.Status, err)
		}
	})
}

func TestWrongGun(t *testing.T) {
	ctx := context.Background()
	f := newFlowFixture(t)
	id, err := f.st.CreateRequest(ctx, f.user.ID, "rifle", "", "")
	if err != nil {
		t.Fatal(err)
	}
	approved, err := f.st.ApproveRequest(ctx, id, f.admin.ID)
	if err != nil {
		t.Fatal(err)
	}

	found, err := f.st.ApprovedRequestInLocker(ctx, f.locker.ID)
	if err != nil || found.ID != id {
		t.Fatalf("got %+v, %v", found, err)
	}
	if _, err := f.st.ApprovedRequestInLocker(ctx, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	wrong := f.locker.Slots[3]
	if firstSlotID(approved) == wrong.ID {
		wrong = f.locker.Slots[2]
	}
	info, err := f.st.GetSlotInfo(ctx, wrong.ID)
	if err != nil || info.LockerID != f.locker.ID || info.SlotNo != wrong.SlotNo {
		t.Fatalf("got %+v, %v", info, err)
	}
	if _, err := f.st.GetSlotInfo(ctx, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	if err := f.st.RecordWrongGun(ctx, wrong.ID, id); err != nil {
		t.Fatal(err)
	}

	t.Run("a wrong slot that is still empty is reported", func(t *testing.T) {
		if _, err := f.st.db.Exec("UPDATE slots SET reading = 0 WHERE id = ?", wrong.ID); err != nil {
			t.Fatal(err)
		}
		got, err := f.st.WrongSlots(ctx, id)
		if err != nil || !reflect.DeepEqual(got, []int64{wrong.SlotNo}) {
			t.Fatalf("got %v, %v, want [%d]", got, err, wrong.SlotNo)
		}
	})

	t.Run("putting it back clears the warning", func(t *testing.T) {
		if _, err := f.st.db.Exec("UPDATE slots SET reading = 1 WHERE id = ?", wrong.ID); err != nil {
			t.Fatal(err)
		}
		got, err := f.st.WrongSlots(ctx, id)
		if err != nil || len(got) != 0 {
			t.Fatalf("got %v, %v, want none", got, err)
		}
	})
}

func TestCountAvailable(t *testing.T) {
	ctx := context.Background()
	f := newFlowFixture(t)

	count := func(kind string) int64 {
		n, err := f.st.CountAvailable(ctx, kind)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if got := count("rifle"); got != 4 {
		t.Fatalf("rifle = %d, want 4", got)
	}
	if got := count("pistol"); got != 0 {
		t.Fatalf("pistol = %d, want 0", got)
	}

	if _, err := f.st.db.Exec("UPDATE slots SET reading = 2 WHERE id = ?", f.locker.Slots[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := count("rifle"); got != 3 {
		t.Fatalf("a not-detecting slot must not count, got %d", got)
	}

	id, err := f.st.CreateRequest(ctx, f.user.ID, "rifle", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.ApproveRequest(ctx, id, f.admin.ID); err != nil {
		t.Fatal(err)
	}
	if got := count("rifle"); got != 2 {
		t.Fatalf("an approved slot must be reserved, got %d", got)
	}
}

func TestHasLoginAdmin(t *testing.T) {
	ctx := context.Background()
	f := newFlowFixture(t)

	has := func() bool {
		got, err := f.st.HasLoginAdmin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if has() {
		t.Fatal("an admin without a face cannot log in yet")
	}
	if err := f.st.ReplaceFaceEnrollments(ctx, f.user.ID, []string{"[]"}, "m"); err != nil {
		t.Fatal(err)
	}
	if has() {
		t.Fatal("a requester's face must not count")
	}
	if err := f.st.ReplaceFaceEnrollments(ctx, f.admin.ID, []string{"[]"}, "m"); err != nil {
		t.Fatal(err)
	}
	if !has() {
		t.Fatal("an admin with a face can log in")
	}
}
