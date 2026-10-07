package database

import (
	"context"
	"errors"
	"testing"

	"armory/internal/models"
)

func TestRenameChangesOnlyThatPerson(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	a, err := st.CreateUser(ctx, models.User{Name: "Roshn", ServiceNo: "ROSHN", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.CreateUser(ctx, models.User{Name: "Alice", ServiceNo: "ALICE", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}

	if err := st.Rename(ctx, a.ID, "Roshan Sahani"); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetUser(ctx, a.ID)
	if err != nil || got.Name != "Roshan Sahani" || got.ServiceNo != "ROSHN" {
		t.Fatalf("renamed = %+v, %v", got, err)
	}
	other, err := st.GetUser(ctx, b.ID)
	if err != nil || other.Name != "Alice" {
		t.Fatalf("other person changed: %+v, %v", other, err)
	}
}

func TestRenameUnknownOrRemovedPersonIsNotFound(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	u, err := st.CreateUser(ctx, models.User{Name: "Bob", ServiceNo: "BOB", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}

	if err := st.Rename(ctx, 9999, "Nobody"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id: %v", err)
	}
	if err := st.DeactivateRequester(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.Rename(ctx, u.ID, "Robert"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("removed person: %v", err)
	}
}
