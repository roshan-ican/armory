package services

import (
	"context"

	"armory/internal/database"
	"armory/internal/models"
)

const activityLimit = 200

type ActivityService struct {
	store *database.Store
}

func NewActivityService(store *database.Store) *ActivityService {
	return &ActivityService{store: store}
}

func (s *ActivityService) Recent(ctx context.Context) ([]models.Event, error) {
	return s.store.ListEvents(ctx, activityLimit)
}
