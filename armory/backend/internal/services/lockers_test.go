package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"armory/internal/database"
)

func TestLockerServiceUpdate(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	lockers := NewLockerService(database.NewStore(db))

	l, err := lockers.Create(ctx, "A", "", "10.0.0.1", "rifle", 3)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("grows and shrinks slots", func(t *testing.T) {
		got, err := lockers.Update(ctx, l.ID, "A", "east", "10.0.0.1", "rifle", 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Slots) != 5 || got.Location != "east" {
			t.Fatalf("got %d slots, location %q", len(got.Slots), got.Location)
		}
		got, err = lockers.Update(ctx, l.ID, "A", "east", "10.0.0.1", "rifle", 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Slots) != 2 {
			t.Fatalf("got %d slots, want 2", len(got.Slots))
		}
	})

	t.Run("regrowing brings the slots back", func(t *testing.T) {
		got, err := lockers.Update(ctx, l.ID, "A", "east", "10.0.0.1", "rifle", 4)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Slots) != 4 {
			t.Fatalf("got %d slots, want 4", len(got.Slots))
		}
	})

	t.Run("type can change", func(t *testing.T) {
		got, err := lockers.Update(ctx, l.ID, "A", "", "10.0.0.1", "pistol", 2)
		if err != nil {
			t.Fatal(err)
		}
		if got.Kind != "pistol" {
			t.Fatalf("kind %q", got.Kind)
		}
	})

	t.Run("invalid values are refused", func(t *testing.T) {
		if _, err := lockers.Update(ctx, l.ID, "A", "", "10.0.0.1", "cannon", 2); !errors.Is(err, ErrInvalidKind) {
			t.Fatalf("kind: got %v", err)
		}
		if _, err := lockers.Update(ctx, l.ID, "A", "", "10.0.0.1", "rifle", 9); !errors.Is(err, ErrInvalidCapacity) {
			t.Fatalf("capacity: got %v", err)
		}
		if _, err := lockers.Update(ctx, l.ID, "A", "", "not-an-ip", "rifle", 2); !errors.Is(err, ErrInvalidIP) {
			t.Fatalf("ip: got %v", err)
		}
	})

	t.Run("duplicate name is refused", func(t *testing.T) {
		other, err := lockers.Create(ctx, "B", "", "10.0.0.2", "pistol", 1)
		if err != nil {
			t.Fatal(err)
		}
		_, err = lockers.Update(ctx, other.ID, "A", "", "10.0.0.2", "pistol", 1)
		if !errors.Is(err, ErrLockerExists) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("unknown locker", func(t *testing.T) {
		_, err := lockers.Update(ctx, 999, "Z", "", "", "rifle", 1)
		if !errors.Is(err, ErrLockerNotFound) {
			t.Fatalf("got %v", err)
		}
	})
}
