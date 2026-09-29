package database

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) FindSensorSlot(ctx context.Context, lockerIP string, slotNo int64) (int64, error) {
	var slotID int64
	err := s.db.QueryRowContext(ctx, `
	SELECT s.id
	FROM slots s
	JOIN lockers l ON l.id = s.locker_id
	WHERE l.ip_address = ? AND s.slot_no = ? AND s.active = 1
	`, lockerIP, slotNo).Scan(&slotID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return slotID, err
}

func (s *Store) RecordSensorEvent(ctx context.Context, slotID int64, reading byte, eventType string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ts := now()
	_, err = tx.ExecContext(ctx,
		"INSERT INTO events (occurred_at, type, slot_id) VALUES (?, ?, ?)",
		ts, eventType, slotID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		"UPDATE slots SET reading = ?, updated_at = ? WHERE id = ?",
		reading, ts, slotID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// SyncSensorSlot stores the first reading after startup and logs an event only if it differs from the last stored one.
func (s *Store) SyncSensorSlot(ctx context.Context, slotID int64, reading byte, eventType string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var previous int64
	err = tx.QueryRowContext(ctx,
		"SELECT COALESCE(reading, -1) FROM slots WHERE id = ?", slotID).Scan(&previous)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	ts := now()
	_, err = tx.ExecContext(ctx,
		"UPDATE slots SET reading = ?, updated_at = ? WHERE id = ?",
		reading, ts, slotID)
	if err != nil {
		return err
	}
	if previous != -1 && previous != int64(reading) {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO events (occurred_at, type, slot_id, details) VALUES (?, ?, ?, ?)",
			ts, eventType, slotID, "changed while the server was offline")
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
