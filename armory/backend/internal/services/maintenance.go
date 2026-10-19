package services

import (
	"context"
	"errors"
	"log"

	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/models"
)

type MaintenanceService struct {
	store *database.Store
	hw    Hardware
	hub   *live.Hub
}

func NewMaintenanceService(store *database.Store, hw Hardware, hub *live.Hub) *MaintenanceService {
	return &MaintenanceService{store: store, hw: hw, hub: hub}
}

func (s *MaintenanceService) requireAdmin(ctx context.Context, adminID int64) error {
	u, err := s.store.GetUser(ctx, adminID)
	if errors.Is(err, database.ErrNotFound) || (err == nil && u.Role != "admin") {
		return ErrNotAdmin
	}
	return err
}

func (s *MaintenanceService) Assign(ctx context.Context, adminID, userID int64) error {
	if err := s.requireAdmin(ctx, adminID); err != nil {
		return err
	}
	u, err := s.store.GetUser(ctx, userID)
	if errors.Is(err, database.ErrNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if u.Role != "requester" {
		return ErrNotRequester
	}
	if u.Faces == 0 {
		return ErrNoFaceEnrolled
	}
	_, err = s.store.AssignMaintenance(ctx, userID, adminID)
	switch {
	case errors.Is(err, database.ErrDuplicate):
		return ErrMaintenanceOpen
	case errors.Is(err, database.ErrNotFound):
		return ErrUserNotFound
	case errors.Is(err, database.ErrConflict):
		return ErrNotRequester
	case err != nil:
		return err
	}
	log.Printf("maintenance: %s assigned by admin %d", u.Name, adminID)
	s.hub.Publish()
	return nil
}

func (s *MaintenanceService) Request(ctx context.Context, userID int64, changes []models.MaintenanceChange) (models.MaintenanceJob, error) {
	lockers, err := s.store.ListLockers(ctx)
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	byID := make(map[int64]models.Locker, len(lockers))
	for _, l := range lockers {
		byID[l.ID] = l
	}

	capacity := make(map[int64]int64)
	for i := range changes {
		c := &changes[i]
		l, ok := byID[c.LockerID]
		if !ok {
			return models.MaintenanceJob{}, ErrLockerNotFound
		}
		if _, seen := capacity[l.ID]; !seen {
			capacity[l.ID] = l.Capacity
		}
		switch c.Action {
		case models.ChangeAddSensor:
			capacity[l.ID]++
			c.SlotNo = 0
		case models.ChangeRemoveSensor:
			capacity[l.ID]--
			c.SlotNo = 0
		case models.ChangeReplaceSensor:
			if c.SlotNo < 1 || c.SlotNo > l.Capacity {
				return models.MaintenanceJob{}, ErrBadChange
			}
		default:
			return models.MaintenanceJob{}, ErrBadChange
		}
		if capacity[l.ID] < 1 || capacity[l.ID] > database.SlotsPerLocker {
			return models.MaintenanceJob{}, ErrBadChange
		}
	}

	job, err := s.store.RequestMaintenance(ctx, userID, changes)
	switch {
	case errors.Is(err, database.ErrNotFound):
		return models.MaintenanceJob{}, ErrUserNotFound
	case errors.Is(err, database.ErrConflict):
		return models.MaintenanceJob{}, ErrNotRequester
	case errors.Is(err, database.ErrDuplicate):
		return models.MaintenanceJob{}, ErrMaintenanceOpen
	case err != nil:
		return models.MaintenanceJob{}, err
	}
	log.Printf("maintenance: user %d requested job %d with %d changes", userID, job.ID, len(changes))
	s.hub.Publish()
	return job, nil
}

func (s *MaintenanceService) Approve(ctx context.Context, adminID, jobID int64) error {
	if err := s.requireAdmin(ctx, adminID); err != nil {
		return err
	}
	err := s.store.ApproveMaintenance(ctx, jobID, adminID)
	if errors.Is(err, database.ErrNotFound) {
		return ErrMaintenanceNotFound
	}
	if err != nil {
		return err
	}
	log.Printf("maintenance: job %d approved by admin %d", jobID, adminID)
	s.hub.Publish()
	return nil
}

func (s *MaintenanceService) Cancel(ctx context.Context, adminID, jobID int64) error {
	if err := s.requireAdmin(ctx, adminID); err != nil {
		return err
	}
	err := s.store.CancelMaintenance(ctx, jobID)
	if errors.Is(err, database.ErrNotFound) {
		return ErrMaintenanceNotFound
	}
	if err != nil {
		return err
	}
	log.Printf("maintenance: job %d cancelled by admin %d", jobID, adminID)
	s.hub.Publish()
	return nil
}

func (s *MaintenanceService) Begin(ctx context.Context, userID int64) (bool, error) {
	job, err := s.store.OpenMaintenanceFor(ctx, userID)
	if errors.Is(err, database.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := s.store.StartMaintenance(ctx, job.ID); err != nil {
		return false, err
	}
	if err := s.hw.OpenDoor(ctx); err != nil {
		log.Printf("maintenance: open door for user %d: %v", userID, err)
	}
	s.unlockAll(ctx)
	log.Printf("maintenance: user %d started, door opened", userID)
	s.hub.Publish()
	return true, nil
}

func (s *MaintenanceService) unlockAll(ctx context.Context) {
	lockers, err := s.store.ListLockers(ctx)
	if err != nil {
		log.Printf("maintenance: list lockers to unlock: %v", err)
		return
	}
	for _, l := range lockers {
		s.pulseLocker(ctx, l)
	}
}

func (s *MaintenanceService) pulseLocker(ctx context.Context, l models.Locker) {
	if l.IPAddress == "" {
		return
	}
	slotNo := int64(1)
	if len(l.Slots) > 0 {
		slotNo = l.Slots[0].SlotNo
	}
	if err := s.hw.Unlock(ctx, l.IPAddress, slotNo); err != nil {
		log.Printf("maintenance: unlock %s: %v", l.Name, err)
	}
}

func (s *MaintenanceService) UnlockLocker(ctx context.Context, userID, lockerID int64) error {
	job, err := s.store.OpenMaintenanceFor(ctx, userID)
	if errors.Is(err, database.ErrNotFound) || (err == nil && job.Status != models.MaintenanceActive) {
		return ErrMaintenanceNotFound
	}
	if err != nil {
		return err
	}
	l, err := s.store.GetLocker(ctx, lockerID)
	if errors.Is(err, database.ErrNotFound) {
		return ErrLockerNotFound
	}
	if err != nil {
		return err
	}
	s.pulseLocker(ctx, l)
	log.Printf("maintenance: user %d unlocked %s", userID, l.Name)
	return nil
}

func (s *MaintenanceService) Finish(ctx context.Context, userID int64) error {
	err := s.store.FinishMaintenance(ctx, userID)
	if errors.Is(err, database.ErrNotFound) {
		return ErrMaintenanceNotFound
	}
	if err != nil {
		return err
	}
	log.Printf("maintenance: user %d finished", userID)
	s.hub.Say(MaintenanceDoneMessage())
	s.hub.Publish()
	return nil
}

func (s *MaintenanceService) Open(ctx context.Context, userID int64) (models.MaintenanceJob, error) {
	job, err := s.store.OpenMaintenanceFor(ctx, userID)
	if errors.Is(err, database.ErrNotFound) {
		return models.MaintenanceJob{}, ErrMaintenanceNotFound
	}
	return job, err
}

func (s *MaintenanceService) Current(ctx context.Context, userID int64) (models.MaintenanceJob, error) {
	job, err := s.store.CurrentMaintenanceFor(ctx, userID)
	if errors.Is(err, database.ErrNotFound) {
		return models.MaintenanceJob{}, ErrMaintenanceNotFound
	}
	return job, err
}

func (s *MaintenanceService) Requested(ctx context.Context) ([]models.MaintenanceJob, error) {
	return s.store.ListRequestedMaintenance(ctx)
}

func (s *MaintenanceService) Lockers(ctx context.Context) ([]models.Locker, error) {
	return s.store.ListLockers(ctx)
}

func (s *MaintenanceService) SetCapacity(ctx context.Context, userID, lockerID, capacity int64) error {
	job, err := s.store.OpenMaintenanceFor(ctx, userID)
	if errors.Is(err, database.ErrNotFound) || (err == nil && job.Status != models.MaintenanceActive) {
		return ErrMaintenanceNotFound
	}
	if err != nil {
		return err
	}
	if capacity < 1 || capacity > database.SlotsPerLocker {
		return ErrBadChange
	}

	l, err := s.store.GetLocker(ctx, lockerID)
	if errors.Is(err, database.ErrNotFound) {
		return ErrLockerNotFound
	}
	if err != nil {
		return err
	}
	if l.Capacity == capacity {
		return nil
	}
	l.Capacity = capacity
	if _, err := s.store.UpdateLocker(ctx, l); err != nil {
		return err
	}
	log.Printf("maintenance: user %d set %s to %d sensors", userID, l.Name, capacity)
	s.hub.Publish()
	return nil
}
