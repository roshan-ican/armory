package services

import (
	"context"
	"errors"
	"net"
	"strings"

	"armory/internal/database"
	"armory/internal/models"
)

type LockerService struct {
	store *database.Store
}

func NewLockerService(store *database.Store) *LockerService {
	return &LockerService{store: store}
}

func (s *LockerService) List(ctx context.Context) ([]models.Locker, error) {
	return s.store.ListLockers(ctx)
}

func (s *LockerService) Get(ctx context.Context, id int64) (models.Locker, error) {
	l, err := s.store.GetLocker(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return models.Locker{}, ErrLockerNotFound
	}
	return l, err
}

func newLocker(name, location, ip, kind string, capacity int64) (models.Locker, error) {
	l := models.Locker{
		Name:      strings.TrimSpace(name),
		Location:  strings.TrimSpace(location),
		IPAddress: strings.TrimSpace(ip),
		Kind:      kind,
		Capacity:  capacity,
	}
	if l.Name == "" {
		return models.Locker{}, ErrNameRequired
	}
	if l.IPAddress != "" && net.ParseIP(l.IPAddress) == nil {
		return models.Locker{}, ErrInvalidIP
	}
	if kind != "rifle" && kind != "pistol" {
		return models.Locker{}, ErrInvalidKind
	}
	if capacity < 1 || capacity > database.SlotsPerLocker {
		return models.Locker{}, ErrInvalidCapacity
	}
	return l, nil
}

func (s *LockerService) Create(ctx context.Context, name, location, ip, kind string, capacity int64) (models.Locker, error) {
	l, err := newLocker(name, location, ip, kind, capacity)
	if err != nil {
		return models.Locker{}, err
	}

	created, err := s.store.CreateLocker(ctx, l)
	if errors.Is(err, database.ErrDuplicate) {
		return models.Locker{}, ErrLockerExists
	}
	return created, err
}

func (s *LockerService) Update(ctx context.Context, id int64, name, location, ip, kind string, capacity int64) (models.Locker, error) {
	l, err := newLocker(name, location, ip, kind, capacity)
	if err != nil {
		return models.Locker{}, err
	}
	l.ID = id

	updated, err := s.store.UpdateLocker(ctx, l)
	switch {
	case errors.Is(err, database.ErrDuplicate):
		return models.Locker{}, ErrLockerExists
	case errors.Is(err, database.ErrNotFound):
		return models.Locker{}, ErrLockerNotFound
	}
	return updated, err
}

func (s *LockerService) SwapSensors(ctx context.Context, id, a, b int64) (models.Locker, error) {
	l, err := s.Get(ctx, id)
	if err != nil {
		return models.Locker{}, err
	}
	if a == b || a < 1 || b < 1 || a > l.Capacity || b > l.Capacity {
		return models.Locker{}, ErrInvalidSlot
	}
	err = s.store.SwapSensors(ctx, id, a, b)
	switch {
	case errors.Is(err, database.ErrInUse):
		return models.Locker{}, ErrSlotInUse
	case errors.Is(err, database.ErrNotFound):
		return models.Locker{}, ErrInvalidSlot
	case err != nil:
		return models.Locker{}, err
	}
	return s.Get(ctx, id)
}

func (s *LockerService) SetSensorLayout(ctx context.Context, id, start int64, reverse bool) (models.Locker, error) {
	l, err := s.Get(ctx, id)
	if err != nil {
		return models.Locker{}, err
	}
	if start < 1 || start > l.Capacity {
		return models.Locker{}, ErrInvalidSensorStart
	}
	if err := s.store.SetSensorLayout(ctx, id, start, reverse); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return models.Locker{}, ErrLockerNotFound
		}
		return models.Locker{}, err
	}
	return s.Get(ctx, id)
}
