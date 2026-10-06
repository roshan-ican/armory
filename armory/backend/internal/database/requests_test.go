package database

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"armory/internal/models"
)

func TestCreateRequest(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("new request starts pending", func(t *testing.T) {
		id, err := st.CreateRequest(ctx, user.ID, "rifle", "Range practice", "2026-09-30T14:00:00Z")
		if err != nil {
			t.Fatal(err)
		}
		if id == 0 {
			t.Fatal("got id 0")
		}

		var status, kind, reason, expected, created string
		err = st.db.QueryRow(
			"SELECT status, kind, reason, expected_return_at, created_at FROM requests WHERE id = ?", id,
		).Scan(&status, &kind, &reason, &expected, &created)
		if err != nil {
			t.Fatal(err)
		}
		if status != models.RequestPending || kind != "rifle" || reason != "Range practice" {
			t.Fatalf("got status %q kind %q reason %q", status, kind, reason)
		}
		if expected != "2026-09-30T14:00:00Z" || created == "" {
			t.Fatalf("got expected %q created %q", expected, created)
		}
	})

	t.Run("empty reason is stored as NULL", func(t *testing.T) {
		id, err := st.CreateRequest(ctx, user.ID, "pistol", "", "")
		if err != nil {
			t.Fatal(err)
		}
		var nulls int
		err = st.db.QueryRow(
			"SELECT (reason IS NULL) + (expected_return_at IS NULL) FROM requests WHERE id = ?", id,
		).Scan(&nulls)
		if err != nil {
			t.Fatal(err)
		}
		if nulls != 2 {
			t.Fatalf("got %d NULL columns, want 2", nulls)
		}
	})

	t.Run("unknown user is refused", func(t *testing.T) {
		_, err := st.CreateRequest(ctx, 9999, "rifle", "", "")
		if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
			t.Fatalf("want a foreign key error, got %v", err)
		}
	})

	t.Run("unknown kind is refused", func(t *testing.T) {
		_, err := st.CreateRequest(ctx, user.ID, "cannon", "", "")
		if err == nil || !strings.Contains(err.Error(), "CHECK") {
			t.Fatalf("want a CHECK constraint error, got %v", err)
		}
	})
}

func TestGetRequest(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, models.User{Name: "Roshan Sahani", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	locker, err := st.CreateLocker(ctx, models.Locker{Name: "East", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("pending request has no slot yet", func(t *testing.T) {
		id, err := st.CreateRequest(ctx, user.ID, "rifle", "Range practice", "2026-09-30T14:00:00Z")
		if err != nil {
			t.Fatal(err)
		}
		got, err := st.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		want := models.Request{
			ID: id, UserID: user.ID, UserName: "Roshan Sahani", ServiceNo: "SN-1",
			Kind: "rifle", Reason: "Range practice", Status: models.RequestPending,
			ExpectedReturnAt: "2026-09-30T14:00:00Z", CreatedAt: got.CreatedAt,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
		if got.CreatedAt == "" {
			t.Fatal("created_at is empty")
		}
	})

	t.Run("empty optional fields come back as empty strings", func(t *testing.T) {
		id, err := st.CreateRequest(ctx, user.ID, "pistol", "", "")
		if err != nil {
			t.Fatal(err)
		}
		got, err := st.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Reason != "" || got.ExpectedReturnAt != "" || firstSlotID(got) != 0 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("a chosen slot brings its locker and number", func(t *testing.T) {
		id, err := st.CreateRequest(ctx, user.ID, "rifle", "", "")
		if err != nil {
			t.Fatal(err)
		}
		slot := locker.Slots[2]
		if _, err := st.db.Exec("UPDATE requests SET requested_locker_id = ? WHERE id = ?", locker.ID, id); err != nil {
			t.Fatal(err)
		}
		if _, err := st.db.Exec("INSERT INTO request_slots (request_id, slot_id) VALUES (?, ?)", id, slot.ID); err != nil {
			t.Fatal(err)
		}
		got, err := st.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if firstSlotID(got) != slot.ID || got.LockerName != "East" || firstSlotNo(got) != 3 {
			t.Fatalf("got slot %d locker %q number %d", firstSlotID(got), got.LockerName, firstSlotNo(got))
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		if _, err := st.GetRequest(ctx, 9999); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestListRequests(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, kind := range []string{"rifle", "pistol", "rifle", "pistol", "rifle"} {
		id, err := st.CreateRequest(ctx, user.ID, kind, "", "")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	setStatus := func(id int64, status string) {
		if _, err := st.db.Exec("UPDATE requests SET status = ? WHERE id = ?", status, id); err != nil {
			t.Fatal(err)
		}
	}
	setStatus(ids[1], models.RequestApproved)
	setStatus(ids[2], models.RequestRejected)
	setStatus(ids[3], models.RequestReturned)

	idsOf := func(list []models.Request) []int64 {
		var out []int64
		for _, r := range list {
			out = append(out, r.ID)
		}
		return out
	}

	t.Run("pending are oldest first", func(t *testing.T) {
		got, err := st.ListPendingRequests(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if g := idsOf(got); len(g) != 2 || g[0] != ids[0] || g[1] != ids[4] {
			t.Fatalf("got %v, want [%d %d]", g, ids[0], ids[4])
		}
		if got[0].UserName != "Roshan" {
			t.Fatalf("user name = %q", got[0].UserName)
		}
	})

	t.Run("recent are newest first and skip pending", func(t *testing.T) {
		got, err := st.ListRecentRequests(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if g := idsOf(got); len(g) != 3 || g[0] != ids[3] || g[1] != ids[2] || g[2] != ids[1] {
			t.Fatalf("got %v, want [%d %d %d]", g, ids[3], ids[2], ids[1])
		}
	})

	t.Run("limit is respected", func(t *testing.T) {
		got, err := st.ListRecentRequests(ctx, 2)
		if err != nil {
			t.Fatal(err)
		}
		if g := idsOf(got); len(g) != 2 || g[0] != ids[3] || g[1] != ids[2] {
			t.Fatalf("got %v, want [%d %d]", g, ids[3], ids[2])
		}
	})

	t.Run("nothing pending gives an empty list", func(t *testing.T) {
		for _, id := range ids {
			setStatus(id, models.RequestCancelled)
		}
		got, err := st.ListPendingRequests(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d pending, want 0", len(got))
		}
	})
}

func TestRejectRequest(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := st.CreateUser(ctx, models.User{Name: "Admin", ServiceNo: "AD-1", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	newPending := func() int64 {
		id, err := st.CreateRequest(ctx, user.ID, "rifle", "", "")
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	t.Run("pending request is rejected and recorded", func(t *testing.T) {
		id := newPending()
		if err := st.RejectRequest(ctx, id, admin.ID, "Not today"); err != nil {
			t.Fatal(err)
		}
		var status, note, decidedAt string
		var decidedBy int64
		err := st.db.QueryRow(
			"SELECT status, admin_note, decided_by, decided_at FROM requests WHERE id = ?", id,
		).Scan(&status, &note, &decidedBy, &decidedAt)
		if err != nil {
			t.Fatal(err)
		}
		if status != models.RequestRejected || note != "Not today" || decidedBy != admin.ID || decidedAt == "" {
			t.Fatalf("got status %q note %q by %d at %q", status, note, decidedBy, decidedAt)
		}
	})

	t.Run("rejecting twice is a conflict", func(t *testing.T) {
		id := newPending()
		if err := st.RejectRequest(ctx, id, admin.ID, ""); err != nil {
			t.Fatal(err)
		}
		if err := st.RejectRequest(ctx, id, admin.ID, ""); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("an approved request cannot be rejected", func(t *testing.T) {
		id := newPending()
		if _, err := st.db.Exec("UPDATE requests SET status = 'approved' WHERE id = ?", id); err != nil {
			t.Fatal(err)
		}
		if err := st.RejectRequest(ctx, id, admin.ID, ""); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
		got, err := st.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != models.RequestApproved {
			t.Fatalf("status changed to %q", got.Status)
		}
	})

	t.Run("unknown request", func(t *testing.T) {
		if err := st.RejectRequest(ctx, 9999, admin.ID, ""); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("unknown admin is refused and nothing changes", func(t *testing.T) {
		id := newPending()
		err := st.RejectRequest(ctx, id, 9999, "")
		if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
			t.Fatalf("want a foreign key error, got %v", err)
		}
		got, err := st.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != models.RequestPending {
			t.Fatalf("status = %q, want pending", got.Status)
		}
	})
}

func TestApproveRequest(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := st.CreateUser(ctx, models.User{Name: "Admin", ServiceNo: "AD-1", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	rifles, err := st.CreateLocker(ctx, models.Locker{Name: "Rifles", Kind: "rifle", Capacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateLocker(ctx, models.Locker{Name: "Pistols", Kind: "pistol", Capacity: 2}); err != nil {
		t.Fatal(err)
	}
	setReading := func(slotID int64, reading int) {
		if _, err := st.db.Exec("UPDATE slots SET reading = ? WHERE id = ?", reading, slotID); err != nil {
			t.Fatal(err)
		}
	}
	setStatus := func(id int64, status string) {
		if _, err := st.db.Exec("UPDATE requests SET status = ? WHERE id = ?", status, id); err != nil {
			t.Fatal(err)
		}
	}
	newRequest := func(kind string) int64 {
		id, err := st.CreateRequest(ctx, user.ID, kind, "", "")
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	setReading(rifles.Slots[0].ID, 0)
	setReading(rifles.Slots[1].ID, 1)
	setReading(rifles.Slots[2].ID, 1)
	setReading(rifles.Slots[3].ID, 2)

	first := newRequest("rifle")
	t.Run("picks the first slot that has a gun", func(t *testing.T) {
		got, err := st.ApproveRequest(ctx, first, admin.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != models.RequestApproved || got.LockerName != "Rifles" || firstSlotNo(got) != 2 {
			t.Fatalf("got status %q locker %q slot %d", got.Status, got.LockerName, firstSlotNo(got))
		}
		var decidedBy int64
		var decidedAt string
		err = st.db.QueryRow("SELECT decided_by, decided_at FROM requests WHERE id = ?", first).Scan(&decidedBy, &decidedAt)
		if err != nil {
			t.Fatal(err)
		}
		if decidedBy != admin.ID || decidedAt == "" {
			t.Fatalf("decided by %d at %q", decidedBy, decidedAt)
		}
	})

	second := newRequest("rifle")
	t.Run("a collected slot is reserved, so the next one is chosen", func(t *testing.T) {
		if _, err := st.db.Exec("UPDATE request_slots SET status = 'collected' WHERE request_id = ?", first); err != nil {
			t.Fatal(err)
		}
		got, err := st.ApproveRequest(ctx, second, admin.ID)
		if err != nil {
			t.Fatal(err)
		}
		if firstSlotNo(got) != 3 {
			t.Fatalf("slot = %d, want 3", firstSlotNo(got))
		}
	})

	third := newRequest("rifle")
	t.Run("no slot left leaves the request pending", func(t *testing.T) {
		if _, err := st.ApproveRequest(ctx, third, admin.ID); !errors.Is(err, ErrNoFreeSlot) {
			t.Fatalf("err = %v, want ErrNoFreeSlot", err)
		}
		got, err := st.GetRequest(ctx, third)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != models.RequestPending || firstSlotID(got) != 0 {
			t.Fatalf("got status %q slot %d", got.Status, firstSlotID(got))
		}
	})

	t.Run("a collected request still holds its slot", func(t *testing.T) {
		setStatus(first, models.RequestCollected)
		if _, err := st.ApproveRequest(ctx, third, admin.ID); !errors.Is(err, ErrNoFreeSlot) {
			t.Fatalf("err = %v, want ErrNoFreeSlot", err)
		}
	})

	t.Run("a returned request frees its slot", func(t *testing.T) {
		setStatus(first, models.RequestReturned)
		got, err := st.ApproveRequest(ctx, third, admin.ID)
		if err != nil {
			t.Fatal(err)
		}
		if firstSlotNo(got) != 2 {
			t.Fatalf("slot = %d, want 2", firstSlotNo(got))
		}
	})

	t.Run("the kind must match the locker", func(t *testing.T) {
		pistol := newRequest("pistol")
		if _, err := st.ApproveRequest(ctx, pistol, admin.ID); !errors.Is(err, ErrNoFreeSlot) {
			t.Fatalf("err = %v, want ErrNoFreeSlot", err)
		}
	})

	t.Run("an inactive slot is never chosen", func(t *testing.T) {
		rifle := newRequest("rifle")
		setStatus(second, models.RequestReturned)
		setStatus(third, models.RequestReturned)
		if _, err := st.db.Exec("UPDATE slots SET active = 0 WHERE id IN (?, ?)", rifles.Slots[1].ID, rifles.Slots[2].ID); err != nil {
			t.Fatal(err)
		}
		if _, err := st.ApproveRequest(ctx, rifle, admin.ID); !errors.Is(err, ErrNoFreeSlot) {
			t.Fatalf("err = %v, want ErrNoFreeSlot", err)
		}
	})

	t.Run("a request that is not pending is a conflict", func(t *testing.T) {
		if _, err := st.ApproveRequest(ctx, first, admin.ID); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("unknown request", func(t *testing.T) {
		if _, err := st.ApproveRequest(ctx, 9999, admin.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("unknown admin changes nothing", func(t *testing.T) {
		setReading(rifles.Slots[1].ID, 1)
		if _, err := st.db.Exec("UPDATE slots SET active = 1 WHERE id = ?", rifles.Slots[1].ID); err != nil {
			t.Fatal(err)
		}
		id := newRequest("rifle")
		_, err := st.ApproveRequest(ctx, id, 9999)
		if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
			t.Fatalf("want a foreign key error, got %v", err)
		}
		got, err := st.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != models.RequestPending || firstSlotID(got) != 0 {
			t.Fatalf("got status %q slot %d", got.Status, firstSlotID(got))
		}
	})
}

func TestCancelRequest(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	owner, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateUser(ctx, models.User{Name: "Amit", ServiceNo: "SN-2", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	newPending := func() int64 {
		id, err := st.CreateRequest(ctx, owner.ID, "rifle", "", "")
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	statusOf := func(id int64) string {
		got, err := st.GetRequest(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return got.Status
	}

	t.Run("the owner cancels a pending request", func(t *testing.T) {
		id := newPending()
		if err := st.CancelRequest(ctx, id, owner.ID); err != nil {
			t.Fatal(err)
		}
		if got := statusOf(id); got != models.RequestCancelled {
			t.Fatalf("status = %q, want cancelled", got)
		}
		var decidedBy sql.NullInt64
		var decidedAt sql.NullString
		err := st.db.QueryRow("SELECT decided_by, decided_at FROM requests WHERE id = ?", id).Scan(&decidedBy, &decidedAt)
		if err != nil {
			t.Fatal(err)
		}
		if decidedBy.Valid || decidedAt.Valid {
			t.Fatalf("cancel must not record a decision: %v %v", decidedBy, decidedAt)
		}
	})

	t.Run("someone else cannot cancel it", func(t *testing.T) {
		id := newPending()
		if err := st.CancelRequest(ctx, id, other.ID); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
		if got := statusOf(id); got != models.RequestPending {
			t.Fatalf("status = %q, want pending", got)
		}
	})

	t.Run("an approved request cannot be cancelled", func(t *testing.T) {
		id := newPending()
		if _, err := st.db.Exec("UPDATE requests SET status = 'approved' WHERE id = ?", id); err != nil {
			t.Fatal(err)
		}
		if err := st.CancelRequest(ctx, id, owner.ID); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
		if got := statusOf(id); got != models.RequestApproved {
			t.Fatalf("status = %q, want approved", got)
		}
	})

	t.Run("cancelling twice is a conflict", func(t *testing.T) {
		id := newPending()
		if err := st.CancelRequest(ctx, id, owner.ID); err != nil {
			t.Fatal(err)
		}
		if err := st.CancelRequest(ctx, id, owner.ID); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("unknown request", func(t *testing.T) {
		if err := st.CancelRequest(ctx, 9999, owner.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestHasOpenRequest(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateUser(ctx, models.User{Name: "Amit", ServiceNo: "SN-2", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}

	open := func(userID int64) bool {
		got, err := st.HasOpenRequest(ctx, userID)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	if open(user.ID) {
		t.Fatal("a user with no requests must not be open")
	}

	id, err := st.CreateRequest(ctx, user.ID, "rifle", "", "")
	if err != nil {
		t.Fatal(err)
	}

	statuses := []struct {
		status string
		want   bool
	}{
		{models.RequestPending, true},
		{models.RequestApproved, true},
		{models.RequestCollected, true},
		{models.RequestRejected, false},
		{models.RequestCancelled, false},
		{models.RequestExpired, false},
		{models.RequestReturned, false},
	}
	for _, tt := range statuses {
		t.Run(tt.status, func(t *testing.T) {
			if _, err := st.db.Exec("UPDATE requests SET status = ? WHERE id = ?", tt.status, id); err != nil {
				t.Fatal(err)
			}
			if got := open(user.ID); got != tt.want {
				t.Fatalf("status %q: open = %v, want %v", tt.status, got, tt.want)
			}
		})
	}

	t.Run("someone else's open request does not count", func(t *testing.T) {
		if _, err := st.db.Exec("UPDATE requests SET status = 'pending' WHERE id = ?", id); err != nil {
			t.Fatal(err)
		}
		if open(other.ID) {
			t.Fatal("other user must not be open")
		}
	})
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
