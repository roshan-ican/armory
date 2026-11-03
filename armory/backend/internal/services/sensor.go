package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"armory/internal/database"
	"armory/internal/live"
)

type SensorService struct {
	store    *database.Store
	hub      *live.Hub
	onChange func(ctx context.Context, slotID int64, reading byte)
	mu       sync.Mutex
	resync   map[string]bool
	seen     map[string]time.Time
}

func (s *SensorService) OnSlotChange(fn func(ctx context.Context, slotID int64, reading byte)) {
	s.onChange = fn
}

func NewSensorService(store *database.Store, hub *live.Hub) *SensorService {
	return &SensorService{store: store, hub: hub, resync: make(map[string]bool), seen: make(map[string]time.Time)}
}

func (s *SensorService) RequestResync(lockerIP string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resync[lockerIP] = true
}

func (s *SensorService) TakeResync(lockerIP string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.resync[lockerIP]
	delete(s.resync, lockerIP)
	return v
}

func (s *SensorService) Seen(ctx context.Context, lockerIP string) error {
	if err := s.store.MarkLockerSeen(ctx, lockerIP); err != nil {
		return err
	}
	s.mu.Lock()
	last, known := s.seen[lockerIP]
	s.seen[lockerIP] = time.Now()
	s.mu.Unlock()
	if !known || time.Since(last) > 4*time.Second {
		s.hub.Publish()
	}
	return nil
}

func (s *SensorService) Record(ctx context.Context, lockerIP string, slotNo int64, reading byte, eventType string) error {
	slotID, err := s.store.FindSensorSlot(ctx, lockerIP, slotNo)
	if errors.Is(err, database.ErrSensorDisabled) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.store.RecordSensorEvent(ctx, slotID, reading, eventType); err != nil {
		return err
	}
	if s.onChange != nil {
		s.onChange(ctx, slotID, reading)
	}
	s.hub.Publish()
	return nil
}

func (s *SensorService) Sync(ctx context.Context, lockerIP string, slotNo int64, reading byte, eventType string) error {
	slotID, err := s.store.FindSensorSlot(ctx, lockerIP, slotNo)
	if errors.Is(err, database.ErrSensorDisabled) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.store.SyncSensorSlot(ctx, slotID, reading, eventType); err != nil {
		return err
	}
	s.hub.Publish()
	return nil
}
