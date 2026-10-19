package services

import (
	"context"
	"errors"
	"testing"
)

func TestRenameTidiesTheName(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	users := NewUserService(f.store)

	if err := users.Rename(ctx, f.user.ID, "  Roshan    Kumar  Sahani "); err != nil {
		t.Fatal(err)
	}
	got, err := users.Get(ctx, f.user.ID)
	if err != nil || got.Name != "Roshan Kumar Sahani" {
		t.Fatalf("name = %q, %v", got.Name, err)
	}
}

func TestRenameRejectsBlankAndUnknown(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	users := NewUserService(f.store)

	if err := users.Rename(ctx, f.user.ID, "   "); !errors.Is(err, ErrNameRequired) {
		t.Fatalf("blank: %v", err)
	}
	if got, _ := users.Get(ctx, f.user.ID); got.Name != "Roshan Sahani" {
		t.Fatalf("a rejected rename changed the name to %q", got.Name)
	}
	if err := users.Rename(ctx, 9999, "Nobody"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown: %v", err)
	}
}
