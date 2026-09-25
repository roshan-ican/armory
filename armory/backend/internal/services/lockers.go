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

func (s *LockerService) Create(ctx context.Context, name, location, ip string) (models.Locker, error) {
	l := models.Locker{
		Name:      strings.TrimSpace(name),
		Location:  strings.TrimSpace(location),
		IPAddress: strings.TrimSpace(ip),
	}
	if l.Name == "" {
		return models.Locker{}, ErrNameRequired
	}
	if l.IPAddress != "" && net.ParseIP(l.IPAddress) == nil {
		return models.Locker{}, ErrInvalidIP
	}

	created, err := s.store.CreateLocker(ctx, l)
	if errors.Is(err, database.ErrDuplicate) {
		return models.Locker{}, ErrLockerExists
	}
	return created, err
}
