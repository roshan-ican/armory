package database

import (
	"context"

	"armory/internal/models"
)

const SlotsPerLocker = 5

func (s *Store) ListLockers(ctx context.Context) ([]models.Locker, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, COALESCE(location, ''), COALESCE(ip_address, '') FROM lockers ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lockers []models.Locker

	for rows.Next() {
		var l models.Locker
		if err := rows.Scan(&l.ID, &l.Name, &l.Location, &l.IPAddress); err != nil {
			return nil, err
		}
		lockers = append(lockers, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slots, err := s.listSlots(ctx)
	if err != nil {
		return nil, err
	}
	for i := range lockers {
		lockers[i].Slots = slots[lockers[i].ID]
	}
	return lockers, nil
}

func (s *Store) listSlots(ctx context.Context) (map[int64][]models.Slot, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, locker_id, slot_no, sensor_id, active FROM slots ORDER BY locker_id, slot_no")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make(map[int64][]models.Slot)
	for rows.Next() {
		var sl models.Slot
		if err := rows.Scan(&sl.ID, &sl.LockerID, &sl.SlotNo, &sl.SensorID, &sl.Active); err != nil {
			return nil, err
		}
		slots[sl.LockerID] = append(slots[sl.LockerID], sl)
	}
	return slots, rows.Err()
}

func (s *Store) CreateLocker(ctx context.Context, l models.Locker) (models.Locker, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Locker{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		"INSERT INTO lockers (name, location, ip_address) VALUES (?, ?, ?)",
		l.Name, nullIfEmpty(l.Location), nullIfEmpty(l.IPAddress))
	if err != nil {
		if isUniqueViolation(err) {
			return models.Locker{}, ErrDuplicate
		}
		return models.Locker{}, err
	}

	l.ID, err = res.LastInsertId()
	if err != nil {
		return models.Locker{}, err
	}

	ts := now()
	l.Slots = nil
	for slotNo := int64(1); slotNo <= SlotsPerLocker; slotNo++ {
		sensorID := (l.ID-1)*SlotsPerLocker + slotNo
		res, err := tx.ExecContext(ctx,
			"INSERT INTO slots (locker_id, slot_no, sensor_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			l.ID, slotNo, sensorID, ts, ts)
		if err != nil {
			return models.Locker{}, err
		}
		slotID, err := res.LastInsertId()
		if err != nil {
			return models.Locker{}, err
		}
		l.Slots = append(l.Slots, models.Slot{
			ID: slotID, LockerID: l.ID, SlotNo: slotNo, SensorID: sensorID, Active: true,
		})
	}

	if err := tx.Commit(); err != nil {
		return models.Locker{}, err
	}
	return l, nil
}
