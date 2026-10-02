package services

import (
	"context"

	"armory/internal/database"
	"armory/internal/live"
)

type SensorService struct {
	store    *database.Store
	hub      *live.Hub
	onChange func(ctx context.Context, slotID int64, reading byte)
}

func (s *SensorService) OnSlotChange(fn func(ctx context.Context, slotID int64, reading byte)) {
	s.onChange = fn
}

func NewSensorService(store *database.Store, hub *live.Hub) *SensorService {
	return &SensorService{store: store, hub: hub}
}

func (s *SensorService) Seen(ctx context.Context, lockerIP string) error {
	if err := s.store.MarkLockerSeen(ctx, lockerIP); err != nil {
		return err
	}
	s.hub.Publish()
	return nil
}

func (s *SensorService) Record(ctx context.Context, lockerIP string, slotNo int64, reading byte, eventType string) error {
	slotID, err := s.store.FindSensorSlot(ctx, lockerIP, slotNo)
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
	if err != nil {
		return err
	}
	if err := s.store.SyncSensorSlot(ctx, slotID, reading, eventType); err != nil {
		return err
	}
	s.hub.Publish()
	return nil
}
