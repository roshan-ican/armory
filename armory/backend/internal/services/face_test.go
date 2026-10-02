package services

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"

	"armory/internal/database"
)

func newFaceFixture(t *testing.T) (*UserService, *FaceService) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)
	return NewUserService(store), NewFaceService(store)
}

func descriptorAt(first float64) []float64 {
	d := make([]float64, descriptorLen)
	d[0] = first
	return d
}

func TestDistance(t *testing.T) {
	a := descriptorAt(0)
	b := descriptorAt(0)
	b[0], b[1] = 3, 4
	if got := distance(a, b); math.Abs(got-5) > 1e-9 {
		t.Fatalf("distance = %v, want 5", got)
	}
	if got := distance(a, a); got != 0 {
		t.Fatalf("distance to self = %v, want 0", got)
	}
}

func TestUserServiceCreate(t *testing.T) {
	ctx := context.Background()
	users, _ := newFaceFixture(t)

	u, err := users.Create(ctx, "  Roshan ", " sn-1 ", "")
	if err != nil {
		t.Fatal(err)
	}
	if u.Name != "Roshan" || u.ServiceNo != "SN-1" || u.Role != "requester" {
		t.Fatalf("got %+v", u)
	}

	tests := []struct {
		name, serviceNo, role string
		want                  error
	}{
		{"", "X", "admin", ErrNameRequired},
		{"A", " ", "admin", ErrServiceNoRequired},
		{"A", "X", "boss", ErrInvalidRole},
		{"B", "sn-1", "admin", ErrUserExists},
	}
	for _, tt := range tests {
		if _, err := users.Create(ctx, tt.name, tt.serviceNo, tt.role); !errors.Is(err, tt.want) {
			t.Errorf("Create(%q, %q, %q): got %v, want %v", tt.name, tt.serviceNo, tt.role, err, tt.want)
		}
	}
}

func TestFaceServiceEnrollAndMatch(t *testing.T) {
	ctx := context.Background()
	users, face := newFaceFixture(t)

	alice, err := users.Create(ctx, "Alice", "A1", "requester")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := users.Create(ctx, "Bob", "B1", "requester")
	if err != nil {
		t.Fatal(err)
	}
	if err := face.Enroll(ctx, alice.ID, [][]float64{descriptorAt(0), descriptorAt(0.1)}); err != nil {
		t.Fatal(err)
	}
	if err := face.Enroll(ctx, bob.ID, [][]float64{descriptorAt(2)}); err != nil {
		t.Fatal(err)
	}

	t.Run("exact match", func(t *testing.T) {
		res, err := face.Match(ctx, descriptorAt(0))
		if err != nil {
			t.Fatal(err)
		}
		if !res.Matched || res.User.ID != alice.ID {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("close enough", func(t *testing.T) {
		res, err := face.Match(ctx, descriptorAt(2.3))
		if err != nil {
			t.Fatal(err)
		}
		if !res.Matched || res.User.ID != bob.ID {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("too far from everyone", func(t *testing.T) {
		res, err := face.Match(ctx, descriptorAt(1.1))
		if err != nil {
			t.Fatal(err)
		}
		if res.Matched {
			t.Fatalf("matched %+v, want no match", res)
		}
	})

	t.Run("re-enroll replaces the old face", func(t *testing.T) {
		if err := face.Enroll(ctx, alice.ID, [][]float64{descriptorAt(-5)}); err != nil {
			t.Fatal(err)
		}
		res, err := face.Match(ctx, descriptorAt(0))
		if err != nil {
			t.Fatal(err)
		}
		if res.Matched {
			t.Fatalf("old face still matches: %+v", res)
		}
		res, err = face.Match(ctx, descriptorAt(-5))
		if err != nil {
			t.Fatal(err)
		}
		if !res.Matched || res.User.ID != alice.ID {
			t.Fatalf("new face: got %+v", res)
		}
		got, err := users.Get(ctx, alice.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Faces != 1 {
			t.Fatalf("faces = %d, want 1", got.Faces)
		}
	})

	t.Run("invalid data is refused", func(t *testing.T) {
		if err := face.Enroll(ctx, alice.ID, nil); !errors.Is(err, ErrInvalidDescriptor) {
			t.Errorf("empty: got %v", err)
		}
		if err := face.Enroll(ctx, alice.ID, [][]float64{{1, 2, 3}}); !errors.Is(err, ErrInvalidDescriptor) {
			t.Errorf("short: got %v", err)
		}
		bad := descriptorAt(math.NaN())
		if _, err := face.Match(ctx, bad); !errors.Is(err, ErrInvalidDescriptor) {
			t.Errorf("NaN: got %v", err)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		if err := face.Enroll(ctx, 999, [][]float64{descriptorAt(0)}); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestFaceMatchWithNobodyEnrolled(t *testing.T) {
	_, face := newFaceFixture(t)
	res, err := face.Match(context.Background(), descriptorAt(0))
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched {
		t.Fatalf("matched %+v with nobody enrolled", res)
	}
}
