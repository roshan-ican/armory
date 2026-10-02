package services

import (
	"testing"
	"time"

	"armory/internal/models"
)

func TestSessions(t *testing.T) {
	clock := time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)
	s := NewSessions()
	s.now = func() time.Time { return clock }

	requester := models.User{ID: 1, Name: "Roshan", ServiceNo: "SN-1", Role: "requester"}
	admin := models.User{ID: 2, Name: "Admin", ServiceNo: "AD-1", Role: "admin"}

	t.Run("a token finds its user and role", func(t *testing.T) {
		token := s.Start(requester)
		got, ok := s.Lookup(token)
		if !ok || got.UserID != 1 || got.Name != "Roshan" || got.IsAdmin() {
			t.Fatalf("got %+v, %v", got, ok)
		}
		a, ok := s.Lookup(s.Start(admin))
		if !ok || !a.IsAdmin() {
			t.Fatalf("got %+v, %v", a, ok)
		}
	})

	t.Run("tokens are long and unique", func(t *testing.T) {
		a, b := s.Start(requester), s.Start(requester)
		if a == b || len(a) != 64 {
			t.Fatalf("tokens %q %q", a, b)
		}
	})

	t.Run("an unknown token is refused", func(t *testing.T) {
		if _, ok := s.Lookup("nope"); ok {
			t.Fatal("unknown token accepted")
		}
	})

	t.Run("a requester session expires but activity extends it", func(t *testing.T) {
		token := s.Start(requester)
		clock = clock.Add(2 * time.Minute)
		if _, ok := s.Lookup(token); !ok {
			t.Fatal("expired too early")
		}
		clock = clock.Add(2 * time.Minute)
		if _, ok := s.Lookup(token); !ok {
			t.Fatal("activity should have extended the session")
		}
		clock = clock.Add(requesterSession + time.Second)
		if _, ok := s.Lookup(token); ok {
			t.Fatal("session should have expired")
		}
	})

	t.Run("an admin session lasts much longer", func(t *testing.T) {
		token := s.Start(admin)
		clock = clock.Add(time.Hour)
		if _, ok := s.Lookup(token); !ok {
			t.Fatal("admin session should still be valid")
		}
	})

	t.Run("ending a session logs out", func(t *testing.T) {
		token := s.Start(requester)
		s.End(token)
		if _, ok := s.Lookup(token); ok {
			t.Fatal("session survived End")
		}
	})
}
