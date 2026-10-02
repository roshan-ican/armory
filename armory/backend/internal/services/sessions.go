package services

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"armory/internal/models"
)

const (
	requesterSession = 3 * time.Minute
	adminSession     = 8 * time.Hour
)

type Session struct {
	UserID    int64
	Name      string
	ServiceNo string
	Role      string
	expires   time.Time
}

func (s Session) IsAdmin() bool {
	return s.Role == "admin"
}

type Sessions struct {
	mu    sync.Mutex
	items map[string]*Session
	now   func() time.Time
}

func NewSessions() *Sessions {
	return &Sessions{items: make(map[string]*Session), now: time.Now}
}

func ttlFor(role string) time.Duration {
	if role == "admin" {
		return adminSession
	}
	return requesterSession
}

func (s *Sessions) Start(u models.User) string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	token := hex.EncodeToString(raw)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweep()
	s.items[token] = &Session{
		UserID:    u.ID,
		Name:      u.Name,
		ServiceNo: u.ServiceNo,
		Role:      u.Role,
		expires:   s.now().Add(ttlFor(u.Role)),
	}
	return token
}

func (s *Sessions) Lookup(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[token]
	if !ok {
		return Session{}, false
	}
	if !s.now().Before(item.expires) {
		delete(s.items, token)
		return Session{}, false
	}
	item.expires = s.now().Add(ttlFor(item.Role))
	return *item, true
}

func (s *Sessions) End(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, token)
}

func (s *Sessions) sweep() {
	for token, item := range s.items {
		if !s.now().Before(item.expires) {
			delete(s.items, token)
		}
	}
}
