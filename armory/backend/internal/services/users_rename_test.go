package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

func TestRenamedPersonIsAddressedByTheNewNameOnApproval(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	users := NewUserService(f.store)
	if err := users.Rename(ctx, f.user.ID, "Rohan Verma"); err != nil {
		t.Fatal(err)
	}
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case text := <-spoken:
		if !strings.HasPrefix(text, "Good ") || !strings.Contains(text, " Rohan. Your request is approved.") {
			t.Fatalf("approval said %q", text)
		}
	case <-time.After(time.Second):
		t.Fatal("nothing was spoken")
	}
}
