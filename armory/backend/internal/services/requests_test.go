package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"armory/internal/database"
	"armory/internal/models"
)

type requestFixture struct {
	svc   *RequestService
	store *database.Store
	user  models.User
	admin models.User
}

func newRequestFixture(t *testing.T) requestFixture {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)

	user, err := store.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := store.CreateUser(ctx, models.User{Name: "Admin", ServiceNo: "AD-1", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	return requestFixture{svc: NewRequestService(store, LogHardware{}, nil), store: store, user: user, admin: admin}
}

func TestRequestServiceCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("a valid request is pending with no return time", func(t *testing.T) {
		f := newRequestFixture(t)
		got, err := f.svc.Create(ctx, f.user.ID, "rifle", "  Range practice  ")
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != models.RequestPending || got.Kind != "rifle" || got.UserName != "Roshan" {
			t.Fatalf("got %+v", got)
		}
		if got.Reason != "Range practice" {
			t.Fatalf("reason = %q, want it trimmed", got.Reason)
		}
		if got.ExpectedReturnAt != "" {
			t.Fatalf("return time = %q, want none", got.ExpectedReturnAt)
		}
	})

	t.Run("an unknown kind is refused", func(t *testing.T) {
		f := newRequestFixture(t)
		if _, err := f.svc.Create(ctx, f.user.ID, "cannon", ""); !errors.Is(err, ErrInvalidKind) {
			t.Fatalf("err = %v, want ErrInvalidKind", err)
		}
	})

	t.Run("an unknown user is refused", func(t *testing.T) {
		f := newRequestFixture(t)
		if _, err := f.svc.Create(ctx, 9999, "rifle", ""); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("err = %v, want ErrUserNotFound", err)
		}
	})

	t.Run("only one request may be open at a time", func(t *testing.T) {
		f := newRequestFixture(t)
		first, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Create(ctx, f.user.ID, "pistol", ""); !errors.Is(err, ErrRequestOpen) {
			t.Fatalf("err = %v, want ErrRequestOpen", err)
		}
		if err := f.store.CancelRequest(ctx, first.ID, f.user.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Create(ctx, f.user.ID, "pistol", ""); err != nil {
			t.Fatalf("after cancelling, a new request must work: %v", err)
		}
	})

	t.Run("another person is not blocked", func(t *testing.T) {
		f := newRequestFixture(t)
		if _, err := f.svc.Create(ctx, f.user.ID, "rifle", ""); err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Create(ctx, f.admin.ID, "rifle", ""); err != nil {
			t.Fatalf("a different user must be able to ask: %v", err)
		}
	})
}
