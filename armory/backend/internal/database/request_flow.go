package database

import (
	"context"
	"database/sql"
	"errors"

	"armory/internal/models"
)

func (s *Store) LatestOpenRequest(ctx context.Context, userID int64) (models.Request, error) {
	return s.oneRequest(ctx, " WHERE r.requester_id = ? AND (r.status IN ('pending', 'approved', 'collected') OR (r.status = 'returned' AND r.closed_at IS NULL)) ORDER BY r.id DESC LIMIT 1", userID)
}

func (s *Store) MarkCollected(ctx context.Context, slotID int64) (int64, bool, error) {
	return s.moveSlot(ctx, slotID, models.SlotChosen, models.SlotCollected, "collected_at",
		[]string{models.RequestApproved}, models.RequestCollected, []string{models.SlotChosen})
}

func (s *Store) MarkReturned(ctx context.Context, slotID int64) (int64, bool, error) {
	return s.moveSlot(ctx, slotID, models.SlotCollected, models.SlotReturned, "returned_at",
		[]string{models.RequestApproved, models.RequestCollected}, models.RequestReturned, []string{models.SlotChosen, models.SlotCollected})
}

func (s *Store) moveSlot(ctx context.Context, slotID int64, from, to, stamp string, open []string, done string, pending []string) (int64, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	var requestID int64
	err = tx.QueryRowContext(ctx, `
		SELECT rs.request_id FROM request_slots rs
		JOIN requests r ON r.id = rs.request_id
		WHERE rs.slot_id = ? AND rs.status = ? AND r.status IN (?, ?)
		ORDER BY r.id DESC LIMIT 1`, slotID, from, open[0], open[len(open)-1]).Scan(&requestID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	at := now()
	if _, err := tx.ExecContext(ctx,
		"UPDATE request_slots SET status = ?, "+stamp+" = ? WHERE request_id = ? AND slot_id = ?",
		to, at, requestID, slotID); err != nil {
		return 0, false, err
	}
	var left int64
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM request_slots WHERE request_id = ? AND status IN (?, ?)",
		requestID, pending[0], pending[len(pending)-1]).Scan(&left); err != nil {
		return 0, false, err
	}
	if left == 0 {
		if _, err := tx.ExecContext(ctx,
			"UPDATE requests SET status = ?, "+stamp+" = ? WHERE id = ?", done, at, requestID); err != nil {
			return 0, false, err
		}
	}
	return requestID, left == 0, tx.Commit()
}

func (s *Store) CloseRequest(ctx context.Context, id int64, note string) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE requests SET closed_at = ?, close_note = ? WHERE id = ? AND status = 'returned' AND closed_at IS NULL",
		now(), nullIfEmpty(note), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrConflict
	}
	return nil
}

func (s *Store) FlagRequest(ctx context.Context, id int64, note string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "UPDATE requests SET close_note = ? WHERE id = ?", note, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO events (occurred_at, type, request_id, details) VALUES (?, 'close_problem', ?, ?)",
		now(), id, note); err != nil {
		return err
	}
	return tx.Commit()
}

type SlotInfo struct {
	LockerID int64
	SlotNo   int64
}

func (s *Store) GetSlotInfo(ctx context.Context, slotID int64) (SlotInfo, error) {
	var info SlotInfo
	err := s.db.QueryRowContext(ctx,
		"SELECT locker_id, slot_no FROM slots WHERE id = ?", slotID).Scan(&info.LockerID, &info.SlotNo)
	if errors.Is(err, sql.ErrNoRows) {
		return SlotInfo{}, ErrNotFound
	}
	return info, err
}

func (s *Store) ApprovedRequestInLocker(ctx context.Context, lockerID int64) (models.Request, error) {
	return s.oneRequest(ctx, " WHERE r.status IN ('approved', 'collected') AND r.requested_locker_id = ? ORDER BY r.id LIMIT 1", lockerID)
}

func (s *Store) RecordWrongGun(ctx context.Context, slotID, requestID int64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO events (occurred_at, type, slot_id, request_id, details) VALUES (?, 'wrong_gun', ?, ?, ?)",
		now(), slotID, requestID, "took a gun that was not assigned")
	return err
}

func (s *Store) WrongSlots(ctx context.Context, requestID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
	SELECT DISTINCT s.slot_no
	FROM events e
	JOIN slots s ON s.id = e.slot_id
	WHERE e.request_id = ? AND e.type = 'wrong_gun' AND s.reading = 0
	ORDER BY s.slot_no`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) CountAvailable(ctx context.Context, kind string) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `
	SELECT COUNT(*)
	FROM slots s
	JOIN lockers l ON l.id = s.locker_id
	WHERE l.kind = ? AND s.active = 1 AND s.reading = 1
	  AND s.id NOT IN (`+reservedSlots+`)`, kind).Scan(&n)
	return n, err
}

func (s *Store) CountAvailableInLocker(ctx context.Context, lockerID int64) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM slots s
		WHERE s.locker_id = ? AND s.active = 1 AND s.reading = 1
		AND s.id NOT IN (`+reservedSlots+`)`, lockerID).Scan(&n)
	return n, err
}

func (s *Store) HasLoginAdmin(ctx context.Context) (bool, error) {
	var ok bool
	err := s.db.QueryRowContext(ctx, `
	SELECT EXISTS (
	    SELECT 1 FROM users u
	    JOIN face_enrollments f ON f.user_id = u.id AND f.revoked_at IS NULL
	    WHERE u.role = 'admin' AND u.active = 1)`).Scan(&ok)
	return ok, err
}

func (s *Store) SlotsOfLocker(ctx context.Context, lockerID int64) ([]models.Slot, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, locker_id, slot_no, sensor_id, active, CASE WHEN sensor_pos IS NULL THEN 2 ELSE COALESCE(reading, -1) END FROM slots WHERE locker_id = ? AND active = 1 ORDER BY slot_no", lockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Slot
	for rows.Next() {
		var sl models.Slot
		if err := rows.Scan(&sl.ID, &sl.LockerID, &sl.SlotNo, &sl.SensorID, &sl.Active, &sl.Reading); err != nil {
			return nil, err
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}

func (s *Store) LockerIPOfSlot(ctx context.Context, slotID int64) (string, error) {
	var ip string
	err := s.db.QueryRowContext(ctx, `
	SELECT COALESCE(l.ip_address, '')
	FROM slots s
	JOIN lockers l ON l.id = s.locker_id
	WHERE s.id = ?`, slotID).Scan(&ip)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return ip, err
}

func (s *Store) UnclosedReturnedRequestInLocker(ctx context.Context, lockerID int64) (models.Request, error) {
	return s.oneRequest(ctx, " WHERE r.status = 'returned' AND r.closed_at IS NULL AND r.requested_locker_id = ? ORDER BY r.id DESC LIMIT 1", lockerID)
}

func (s *Store) RecordUnauthorizedTake(ctx context.Context, slotID int64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO events (occurred_at, type, slot_id, details) VALUES (?, 'unauthorized_take', ?, ?)",
		now(), slotID, "took a gun with no approved request")
	return err
}

func (s *Store) HasUnauthorizedEmpty(ctx context.Context, lockerID int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
	SELECT COUNT(*) FROM slots s
	WHERE s.locker_id = ? AND s.active = 1 AND s.reading = 0
	  AND EXISTS (
	    SELECT 1 FROM events e
	    WHERE e.slot_id = s.id AND e.type = 'unauthorized_take'
	      AND e.id > COALESCE((SELECT MAX(r.id) FROM events r WHERE r.slot_id = s.id AND r.type = 'gun_returned'), 0))`,
		lockerID).Scan(&n)
	return n > 0, err
}
