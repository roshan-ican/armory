package services

import (
	"context"

	"armory/internal/database"
)

type SensorService struct {
	store *database.Store
}

func NewSensorService(store *database.Store) *SensorService {
	return &SensorService{store: store}
}

func (s *SensorService) Record(ctx context.Context, lockerIP string, slotNo int64, eventType, newStatus string) error {
	slotID, gunID, err := s.store.FindSensorSlot(ctx, lockerIP, slotNo)
	if err != nil {
		return err
	}
	return s.store.RecordSensorEvent(ctx, slotID, gunID, eventType, newStatus)
}
