package database

import (
	"context"
	"database/sql"
	"errors"

	"armory/internal/models"
)

const SlotsPerLocker = 5

func defaultSensorPos(slotNo, capacity, start int64, reverse bool) sql.NullInt64 {
	if slotNo < start {
		return sql.NullInt64{}
	}
	pos := slotNo - start + 1
	if reverse {
		pos = capacity - slotNo + 1
	}
	if pos < 1 || pos > SlotsPerLocker {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: pos, Valid: true}
}

func (s *Store) ListLockers(ctx context.Context) ([]models.Locker, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, COALESCE(location, ''), COALESCE(ip_address, ''), kind, capacity, sensor_start, sensor_reverse,
			COALESCE(julianday(last_seen) >= julianday('now', '-5 seconds'), 0)
		 FROM lockers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lockers []models.Locker

	for rows.Next() {
		var l models.Locker
		if err := rows.Scan(&l.ID, &l.Name, &l.Location, &l.IPAddress, &l.Kind, &l.Capacity, &l.SensorStart, &l.SensorReverse, &l.Online); err != nil {
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
		`SELECT s.id, s.locker_id, s.slot_no, s.sensor_id, s.active,
			CASE WHEN s.sensor_pos IS NULL THEN 2 ELSE COALESCE(s.reading, -1) END,
			COALESCE((SELECT u.name FROM request_slots rs
				JOIN requests r ON r.id = rs.request_id
				JOIN users u ON u.id = r.requester_id
				WHERE rs.slot_id = s.id AND r.status IN ('approved', 'collected') AND rs.status IN ('chosen', 'collected')
					AND (rs.status = 'collected' OR EXISTS (
						SELECT 1 FROM request_slots x WHERE x.request_id = r.id AND x.status <> 'chosen'))
				ORDER BY r.id DESC LIMIT 1), '')
		 FROM slots s WHERE s.active = 1 ORDER BY s.locker_id, s.slot_no`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make(map[int64][]models.Slot)
	for rows.Next() {
		var sl models.Slot
		if err := rows.Scan(&sl.ID, &sl.LockerID, &sl.SlotNo, &sl.SensorID, &sl.Active, &sl.Reading, &sl.TakenBy); err != nil {
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
		"INSERT INTO lockers (name, location, ip_address, kind, capacity) VALUES (?, ?, ?, ?, ?)",
		l.Name, nullIfEmpty(l.Location), nullIfEmpty(l.IPAddress), l.Kind, l.Capacity)
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
	for slotNo := int64(1); slotNo <= l.Capacity; slotNo++ {
		sensorID := (l.ID-1)*SlotsPerLocker + slotNo
		res, err := tx.ExecContext(ctx,
			"INSERT INTO slots (locker_id, slot_no, sensor_id, sensor_pos, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			l.ID, slotNo, sensorID, defaultSensorPos(slotNo, l.Capacity, 1, false), ts, ts)
		if err != nil {
			return models.Locker{}, err
		}
		slotID, err := res.LastInsertId()
		if err != nil {
			return models.Locker{}, err
		}
		l.Slots = append(l.Slots, models.Slot{
			ID: slotID, LockerID: l.ID, SlotNo: slotNo, SensorID: sensorID, Active: true, Reading: -1,
		})
	}

	if err := tx.Commit(); err != nil {
		return models.Locker{}, err
	}
	return l, nil
}

func (s *Store) GetLocker(ctx context.Context, id int64) (models.Locker, error) {
	lockers, err := s.ListLockers(ctx)
	if err != nil {
		return models.Locker{}, err
	}
	for _, l := range lockers {
		if l.ID == id {
			return l, nil
		}
	}
	return models.Locker{}, ErrNotFound
}

func (s *Store) UpdateLocker(ctx context.Context, l models.Locker) (models.Locker, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Locker{}, err
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM lockers WHERE id = ?", l.ID).Scan(&exists)
	if err != nil {
		return models.Locker{}, err
	}
	if exists == 0 {
		return models.Locker{}, ErrNotFound
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE lockers SET name = ?, location = ?, ip_address = ?, kind = ?, capacity = ? WHERE id = ?",
		l.Name, nullIfEmpty(l.Location), nullIfEmpty(l.IPAddress), l.Kind, l.Capacity, l.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return models.Locker{}, ErrDuplicate
		}
		return models.Locker{}, err
	}

	var start int64
	var reverse bool
	err = tx.QueryRowContext(ctx, "SELECT sensor_start, sensor_reverse FROM lockers WHERE id = ?", l.ID).Scan(&start, &reverse)
	if err != nil {
		return models.Locker{}, err
	}

	ts := now()
	for slotNo := int64(1); slotNo <= SlotsPerLocker; slotNo++ {
		res, err := tx.ExecContext(ctx,
			"UPDATE slots SET active = ?, updated_at = ? WHERE locker_id = ? AND slot_no = ?",
			slotNo <= l.Capacity, ts, l.ID, slotNo)
		if err != nil {
			return models.Locker{}, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return models.Locker{}, err
		}
		if n == 0 && slotNo <= l.Capacity {
			sensorID := (l.ID-1)*SlotsPerLocker + slotNo
			_, err := tx.ExecContext(ctx,
				"INSERT INTO slots (locker_id, slot_no, sensor_id, sensor_pos, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
				l.ID, slotNo, sensorID, defaultSensorPos(slotNo, l.Capacity, start, reverse), ts, ts)
			if err != nil {
				return models.Locker{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return models.Locker{}, err
	}
	return s.GetLocker(ctx, l.ID)
}

func (s *Store) SetSensorLayout(ctx context.Context, id, start int64, reverse bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "UPDATE lockers SET sensor_start = ?, sensor_reverse = ? WHERE id = ?", start, reverse, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}

	var capacity int64
	if err := tx.QueryRowContext(ctx, "SELECT capacity FROM lockers WHERE id = ?", id).Scan(&capacity); err != nil {
		return err
	}
	for slotNo := int64(1); slotNo <= SlotsPerLocker; slotNo++ {
		_, err := tx.ExecContext(ctx, "UPDATE slots SET sensor_pos = ?, updated_at = ? WHERE locker_id = ? AND slot_no = ?",
			defaultSensorPos(slotNo, capacity, start, reverse), now(), id, slotNo)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SwapSensors(ctx context.Context, lockerID, slotA, slotB int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	type state struct {
		id      int64
		pos     sql.NullInt64
		reading sql.NullInt64
	}
	load := func(slotNo int64) (state, error) {
		var st state
		err := tx.QueryRowContext(ctx,
			"SELECT id, sensor_pos, reading FROM slots WHERE locker_id = ? AND slot_no = ? AND active = 1",
			lockerID, slotNo).Scan(&st.id, &st.pos, &st.reading)
		if errors.Is(err, sql.ErrNoRows) {
			return st, ErrNotFound
		}
		return st, err
	}
	a, err := load(slotA)
	if err != nil {
		return err
	}
	b, err := load(slotB)
	if err != nil {
		return err
	}

	var busy int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM request_slots rs
		JOIN requests r ON r.id = rs.request_id
		WHERE rs.slot_id IN (?, ?) AND r.status IN ('approved', 'collected') AND rs.status IN ('chosen', 'collected')`,
		a.id, b.id).Scan(&busy)
	if err != nil {
		return err
	}
	if busy > 0 {
		return ErrInUse
	}

	ts := now()
	for _, u := range []struct {
		id   int64
		from state
	}{{a.id, b}, {b.id, a}} {
		_, err := tx.ExecContext(ctx, "UPDATE slots SET sensor_pos = ?, reading = ?, updated_at = ? WHERE id = ?",
			u.from.pos, u.from.reading, ts, u.id)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
