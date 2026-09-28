package services

import (
	"context"

	"armory/internal/database"
	"armory/internal/live"
)

type SensorService struct {
	store *database.Store
	hub   *live.Hub
}

func NewSensorService(store *database.Store, hub *live.Hub) *SensorService {
	return &SensorService{store: store, hub: hub}
}

func (s *SensorService) Record(ctx context.Context, lockerIP string, slotNo int64, reading byte, eventType, newStatus string) error {
	slotID, gunID, err := s.store.FindSensorSlot(ctx, lockerIP, slotNo)
	if err != nil {
		return err
	}
	if err := s.store.RecordSensorEvent(ctx, slotID, gunID, reading, eventType, newStatus); err != nil {
		return err
	}
	s.hub.Publish()
	return nil
}

func (s *SensorService) Sync(ctx context.Context, lockerIP string, slotNo int64, reading byte, eventType, newStatus string) error {
	slotID, gunID, err := s.store.FindSensorSlot(ctx, lockerIP, slotNo)
	if err != nil {
		return err
	}
	if err := s.store.SyncSensorSlot(ctx, slotID, gunID, reading, eventType, newStatus); err != nil {
		return err
	}
	s.hub.Publish()
	return nil
}
