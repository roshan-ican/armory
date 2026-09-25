package services

import (
	"context"
	"errors"
	"strings"

	"armory/internal/database"
	"armory/internal/models"
)

type GunService struct {
	store *database.Store
}

func NewGunService(store *database.Store) *GunService {
	return &GunService{store: store}
}

func (s *GunService) List(ctx context.Context) ([]models.Gun, error) {
	return s.store.ListGuns(ctx)
}

func (s *GunService) FreeSlots(ctx context.Context) ([]models.FreeSlot, error) {
	return s.store.ListFreeSlots(ctx)
}

func (s *GunService) Create(ctx context.Context, g models.Gun) (models.Gun, error) {
	g.Serial = strings.ToUpper(strings.TrimSpace(g.Serial))
	g.Model = strings.TrimSpace(g.Model)
	g.Notes = strings.TrimSpace(g.Notes)

	if g.SlotID == 0 {
		return models.Gun{}, ErrSlotRequired
	}

	if g.CategoryID == 0 {
		return models.Gun{}, ErrCategoryRequired
	}
	if g.Serial == "" {
		return models.Gun{}, ErrSerialRequired
	}

	created, err := s.store.CreateGun(ctx, g)
	switch {
	case errors.Is(err, database.ErrDuplicate):
		return models.Gun{}, ErrGunExists
	case errors.Is(err, database.ErrSlotTaken):
		return models.Gun{}, ErrSlotOccupied
	}
	return created, err
}

func (s *GunService) Retire(ctx context.Context, id int64) (models.Gun, error) {
	g, err := s.store.RetireGun(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return models.Gun{}, ErrCannotRetire
	}
	return g, err
}
