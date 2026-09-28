package database

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) FindSensorSlot(ctx context.Context, lockerIP string, slotNo int64) (slotID, gunID int64, err error) {
	err = s.db.QueryRowContext(ctx, `
	SELECT s.id, COALESCE(g.id, 0)
	FROM slots s
	JOIN lockers l ON l.id = s.locker_id
	LEFT JOIN guns g ON g.slot_id = s.id
	WHERE l.ip_address = ? AND s.slot_no = ?
	`, lockerIP, slotNo).Scan(&slotID, &gunID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	return slotID, gunID, err
}

func (s *Store) RecordSensorEvent(ctx context.Context, slotID, gunID int64, reading byte, eventType, newStatus string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ts := now()
	_, err = tx.ExecContext(ctx,
		"INSERT INTO events (occurred_at, type, gun_id, slot_id) VALUES (?, ?, ?, ?)",
		ts, eventType, nullIfZero(gunID), slotID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		"UPDATE slots SET reading = ?, updated_at = ? WHERE id = ?",
		reading, ts, slotID)
	if err != nil {
		return err
	}
	if gunID != 0 && newStatus != "" {
		_, err = tx.ExecContext(ctx,
			"UPDATE guns SET status = ?, updated_at = ? WHERE id = ? AND status != 'retired'",
			newStatus, ts, gunID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SyncSensorSlot stores the first reading after startup and logs an event only if the gun's status was wrong.
func (s *Store) SyncSensorSlot(ctx context.Context, slotID, gunID int64, reading byte, eventType, newStatus string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ts := now()
	_, err = tx.ExecContext(ctx,
		"UPDATE slots SET reading = ?, updated_at = ? WHERE id = ?",
		reading, ts, slotID)
	if err != nil {
		return err
	}
	if gunID != 0 && newStatus != "" {
		res, err := tx.ExecContext(ctx,
			"UPDATE guns SET status = ?, updated_at = ? WHERE id = ? AND status NOT IN ('retired', ?)",
			newStatus, ts, gunID, newStatus)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n > 0 {
			_, err = tx.ExecContext(ctx,
				"INSERT INTO events (occurred_at, type, gun_id, slot_id, details) VALUES (?, ?, ?, ?, ?)",
				ts, eventType, gunID, slotID, "synced from sensor on startup")
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
