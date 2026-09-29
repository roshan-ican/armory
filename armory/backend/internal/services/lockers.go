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
