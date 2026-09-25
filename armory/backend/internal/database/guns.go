package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"armory/internal/models"
)

const gunSelect = `
 SELECT g.id, COALESCE(g.slot_id, 0), g.category_id, c.name,
	COALESCE(l.name, ''), COALESCE(s.slot_no, 0),
	g.serial, COALESCE(g.model, ''), COALESCE(g.notes, ''), g.status
FROM guns g
JOIN categories c ON c.id = g.category_id
LEFT JOIN slots s ON s.id = g.slot_id
LEFT JOIN lockers l ON l.id = s.locker_id
`

type scanner interface {
	Scan(dest ...any) error
}

func scanGun(row scanner) (models.Gun, error) {
	var g models.Gun
	err := row.Scan(
		&g.ID,
		&g.SlotID,
		&g.CategoryID,
		&g.CategoryName,
		&g.LockerName,
		&g.SlotNo,
		&g.Serial,
		&g.Model,
		&g.Notes,
		&g.Status,
	)
	return g, err
}

func (s *Store) ListGuns(ctx context.Context) ([]models.Gun, error) {
	rows, err := s.db.QueryContext(ctx, gunSelect+" ORDER BY g.status = 'retired', c.name, g.serial")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guns []models.Gun
	for rows.Next() {
		g, err := scanGun(rows)
		if err != nil {
			return nil, err
		}
		guns = append(guns, g)
	}
	return guns, rows.Err()
}

func (s *Store) GetGun(ctx context.Context, id int64) (models.Gun, error) {
	g, err := scanGun(s.db.QueryRowContext(ctx, gunSelect+" WHERE g.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Gun{}, ErrNotFound
	}
	return g, err
}

func (s *Store) ListFreeSlots(ctx context.Context) ([]models.FreeSlot, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT s.id, l.name, s.slot_no
FROM slots s
JOIN lockers l ON l.id = s.locker_id
WHERE s.active = 1
  AND s.id NOT IN (SELECT slot_id FROM guns WHERE slot_id IS NOT NULL)
ORDER BY l.name, s.slot_no`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []models.FreeSlot
	for rows.Next() {
		var fs models.FreeSlot
		if err := rows.Scan(&fs.ID, &fs.LockerName, &fs.SlotNo); err != nil {
			return nil, err
		}
		slots = append(slots, fs)
	}
	return slots, rows.Err()
}

func (s *Store) CreateGun(ctx context.Context, g models.Gun) (models.Gun, error) {
	ts := now()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO guns (slot_id, category_id, serial, model, notes, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'in', ?, ?)`,
		nullIfZero(g.SlotID), g.CategoryID, g.Serial, nullIfEmpty(g.Model), nullIfEmpty(g.Notes), ts, ts)
	if err != nil {
		if isUniqueViolation(err) && strings.Contains(err.Error(), "guns.slot_id") {
			return models.Gun{}, ErrSlotTaken
		}
		if isUniqueViolation(err) {
			return models.Gun{}, ErrDuplicate
		}
		return models.Gun{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.Gun{}, err
	}
	return s.GetGun(ctx, id)
}

func (s *Store) RetireGun(ctx context.Context, id int64) (models.Gun, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE guns SET status = 'retired', slot_id = NULL, updated_at = ? WHERE id = ? AND status = 'in'",
		now(), id)
	if err != nil {
		return models.Gun{}, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return models.Gun{}, err
	}
	if n == 0 {
		return models.Gun{}, ErrNotFound
	}
	return s.GetGun(ctx, id)
}

func (s *Store) UpdateGun(ctx context.Context, g models.Gun) (models.Gun, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE guns SET category_id = ?, slot_id = ?, serial = ?, model = ?, notes = ?, updated_at = ?
		 WHERE id = ? AND status = 'in'`,
		g.CategoryID, g.SlotID, g.Serial, nullIfEmpty(g.Model), nullIfEmpty(g.Notes), now(), g.ID)
	if err != nil {
		if isUniqueViolation(err) && strings.Contains(err.Error(), "guns.slot_id") {
			return models.Gun{}, ErrSlotTaken
		}
		if isUniqueViolation(err) {
			return models.Gun{}, ErrDuplicate
		}
		return models.Gun{}, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return models.Gun{}, err
	}
	if n == 0 {
		return models.Gun{}, ErrNotFound
	}
	return s.GetGun(ctx, g.ID)
}
