package database

import (
	"context"

	"armory/internal/models"
)

func (s *Store) ListEvents(ctx context.Context, limit int) ([]models.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
	SELECT e.id, e.occurred_at, e.type, COALESCE(l.name, ''), COALESCE(s.slot_no, 0), COALESCE(e.details, '')
	FROM events e
	LEFT JOIN slots s ON s.id = e.slot_id
	LEFT JOIN lockers l ON l.id = s.locker_id
	ORDER BY e.id DESC
	LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Type, &e.LockerName, &e.SlotNo, &e.Details); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
