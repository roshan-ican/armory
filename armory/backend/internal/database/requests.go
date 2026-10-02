package database

import (
	"context"
	"errors"

	"armory/internal/models"
	"database/sql"
)

const reservedSlots = `
SELECT rs.slot_id FROM request_slots rs
JOIN requests rr ON rr.id = rs.request_id
WHERE rr.status IN ('approved', 'collected') AND rs.status IN ('chosen', 'collected')`

func (s *Store) CreateRequest(ctx context.Context, userID int64, kind, reason, expectedReturnAt string) (int64, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO requests (requester_id, kind, reason, expected_return_at, created_at) VALUES (?,?,?,?,?)",
		userID, kind, nullIfEmpty(reason),
		nullIfEmpty(expectedReturnAt), now(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) CreateRequestForLocker(ctx context.Context, userID, lockerID int64, kind, reason, expectedReturnAt string, slotNos []int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO requests
		(requester_id, requested_locker_id, kind, reason, expected_return_at, created_at) VALUES (?,?,?,?,?,?)`,
		userID, lockerID, kind, nullIfEmpty(reason), nullIfEmpty(expectedReturnAt), now())
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, no := range slotNos {
		var slotID int64
		err := tx.QueryRowContext(ctx, `
			SELECT s.id FROM slots s
			WHERE s.locker_id = ? AND s.slot_no = ? AND s.active = 1 AND s.reading = 1
			  AND s.id NOT IN (`+reservedSlots+`)`, lockerID, no).Scan(&slotID)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNoFreeSlot
		}
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO request_slots (request_id, slot_id) VALUES (?, ?)", id, slotID); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

const requestSelect = `
SELECT r.id, r.requester_id, u.name, u.service_no,
	COALESCE(r.kind, ''), COALESCE(r.requested_locker_id, 0), COALESCE(r.reason, ''), r.status,
	COALESCE(l.id, 0), COALESCE(l.name, ''), COALESCE(l.ip_address, ''),
	COALESCE(r.expected_return_at, ''), r.created_at
FROM requests r
JOIN users u ON u.id = r.requester_id
LEFT JOIN lockers l ON l.id = r.requested_locker_id
`

func scanRequest(row scanner) (models.Request, error) {
	var r models.Request
	err := row.Scan(
		&r.ID,
		&r.UserID,
		&r.UserName,
		&r.ServiceNo,
		&r.Kind,
		&r.RequestedLockerID,
		&r.Reason,
		&r.Status,
		&r.LockerID,
		&r.LockerName,
		&r.LockerIP,
		&r.ExpectedReturnAt,
		&r.CreatedAt,
	)
	return r, err
}

func (s *Store) requestSlots(ctx context.Context, requestID int64) ([]models.RequestSlot, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT rs.slot_id, s.slot_no, rs.status
		FROM request_slots rs
		JOIN slots s ON s.id = rs.slot_id
		WHERE rs.request_id = ?
		ORDER BY s.slot_no`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.RequestSlot
	for rows.Next() {
		var sl models.RequestSlot
		if err := rows.Scan(&sl.SlotID, &sl.SlotNo, &sl.Status); err != nil {
			return nil, err
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}

func (s *Store) oneRequest(ctx context.Context, tail string, args ...any) (models.Request, error) {
	r, err := scanRequest(s.db.QueryRowContext(ctx, requestSelect+tail, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Request{}, ErrNotFound
	}
	if err != nil {
		return models.Request{}, err
	}
	r.Slots, err = s.requestSlots(ctx, r.ID)
	return r, err
}

func (s *Store) GetRequest(ctx context.Context, id int64) (models.Request, error) {
	return s.oneRequest(ctx, " WHERE r.id = ?", id)
}

func (s *Store) listRequests(ctx context.Context, tail string, args ...any) ([]models.Request, error) {
	rows, err := s.db.QueryContext(ctx, requestSelect+tail, args...)
	if err != nil {
		return nil, err
	}
	var out []models.Request
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Slots, err = s.requestSlots(ctx, out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) ListPendingRequests(ctx context.Context) ([]models.Request, error) {
	return s.listRequests(ctx, " WHERE r.status = 'pending' ORDER BY r.id")
}

func (s *Store) ListRecentRequests(ctx context.Context, limit int) ([]models.Request, error) {
	return s.listRequests(ctx, " WHERE r.status != 'pending' ORDER BY r.id DESC LIMIT ?", limit)
}

func (s *Store) RejectRequest(ctx context.Context, id, adminID int64, note string) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE requests SET status = 'rejected', decided_by = ?, admin_note = ?, decided_at = ? WHERE id = ? AND status = 'pending'",
		adminID, nullIfEmpty(note), now(), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := s.GetRequest(ctx, id); err != nil {
			return err
		}
		return ErrConflict
	}
	return nil
}

func (s *Store) ApproveRequest(ctx context.Context, id, adminID int64) (models.Request, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Request{}, err
	}
	defer tx.Rollback()

	var status, kind string
	var requestedLockerID int64
	err = tx.QueryRowContext(ctx,
		"SELECT status, COALESCE(kind, ''), COALESCE(requested_locker_id, 0) FROM requests WHERE id = ?", id).Scan(&status, &kind, &requestedLockerID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Request{}, ErrNotFound
	}
	if err != nil {
		return models.Request{}, err
	}
	if status != models.RequestPending {
		return models.Request{}, ErrConflict
	}

	var chosen, blocked int64
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*),
			COUNT(*) FILTER (WHERE s.active = 0 OR COALESCE(s.reading, -1) != 1 OR s.id IN (`+reservedSlots+`))
		FROM request_slots rs
		JOIN slots s ON s.id = rs.slot_id
		WHERE rs.request_id = ?`, id).Scan(&chosen, &blocked)
	if err != nil {
		return models.Request{}, err
	}
	if blocked > 0 {
		return models.Request{}, ErrNoFreeSlot
	}

	if chosen == 0 {
		var slotID, lockerID int64
		err = tx.QueryRowContext(ctx, `
			SELECT s.id, l.id
			FROM slots s
			JOIN lockers l ON l.id = s.locker_id
			WHERE l.kind = ? AND (? = 0 OR l.id = ?) AND s.active = 1 AND s.reading = 1
			  AND s.id NOT IN (`+reservedSlots+`)
			ORDER BY l.id, s.slot_no
			LIMIT 1`, kind, requestedLockerID, requestedLockerID).Scan(&slotID, &lockerID)
		if errors.Is(err, sql.ErrNoRows) {
			return models.Request{}, ErrNoFreeSlot
		}
		if err != nil {
			return models.Request{}, err
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO request_slots (request_id, slot_id) VALUES (?, ?)", id, slotID); err != nil {
			return models.Request{}, err
		}
		if _, err := tx.ExecContext(ctx,
			"UPDATE requests SET requested_locker_id = ? WHERE id = ?", lockerID, id); err != nil {
			return models.Request{}, err
		}
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE requests SET status = 'approved', decided_by = ?, decided_at = ?,
			slot_id = (SELECT rs.slot_id FROM request_slots rs JOIN slots s ON s.id = rs.slot_id
			           WHERE rs.request_id = requests.id ORDER BY s.slot_no LIMIT 1)
		WHERE id = ? AND status = 'pending'`,
		adminID, now(), id)
	if err != nil {
		return models.Request{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return models.Request{}, err
	}
	if n == 0 {
		return models.Request{}, ErrConflict
	}
	if err := tx.Commit(); err != nil {
		return models.Request{}, err
	}
	return s.GetRequest(ctx, id)
}

func (s *Store) CancelRequest(ctx context.Context, id, userID int64) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE requests SET status = 'cancelled' WHERE id = ? AND requester_id = ? AND status = 'pending'",
		id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := s.GetRequest(ctx, id); err != nil {
			return err
		}
		return ErrConflict
	}
	return nil
}

func (s *Store) HasOpenRequest(ctx context.Context, userID int64) (bool, error) {
	var open bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM requests WHERE requester_id = ? AND status IN ('pending', 'approved', 'collected'))",
		userID).Scan(&open)
	return open, err
}
