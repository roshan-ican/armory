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

type EventFilter struct {
	Person string
	Locker string
	Type   string
	From   string
	To     string
}

const eventPerson = `COALESCE(
	(SELECT u.name FROM requests r JOIN users u ON u.id = r.requester_id WHERE r.id = e.request_id),
	(SELECT u.name FROM request_slots rs
		JOIN requests r ON r.id = rs.request_id
		JOIN users u ON u.id = r.requester_id
		WHERE rs.slot_id = e.slot_id
			AND r.status IN ('approved', 'collected', 'returned')
			AND r.decided_at IS NOT NULL AND r.decided_at <= e.occurred_at
			AND (rs.returned_at IS NULL OR e.occurred_at <= rs.returned_at)
		ORDER BY r.decided_at DESC LIMIT 1),
	'')`

const eventRows = `
	SELECT e.id AS id, e.occurred_at AS occurred_at, e.type AS type,
		COALESCE(l.name, '') AS locker, COALESCE(s.slot_no, 0) AS slot_no,
		COALESCE(e.details, '') AS details, ` + eventPerson + ` AS person
	FROM events e
	LEFT JOIN slots s ON s.id = e.slot_id
	LEFT JOIN lockers l ON l.id = s.locker_id`

const eventWhere = `
	WHERE (? = '' OR person = ?) AND (? = '' OR locker = ?) AND (? = '' OR type = ?)
		AND (? = '' OR occurred_at >= ?) AND (? = '' OR occurred_at <= ?)`

func (f EventFilter) args() []any {
	to := f.To
	if to != "" {
		to += "T23:59:59Z"
	}
	return []any{f.Person, f.Person, f.Locker, f.Locker, f.Type, f.Type, f.From, f.From, to, to}
}

func (s *Store) CountEvents(ctx context.Context, f EventFilter) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ("+eventRows+")"+eventWhere, f.args()...).Scan(&n)
	return n, err
}

func (s *Store) ListEventsPage(ctx context.Context, f EventFilter, limit, offset int) ([]models.Event, error) {
	args := append(f.args(), limit, offset)
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, occurred_at, type, locker, slot_no, details, person FROM ("+eventRows+")"+eventWhere+" ORDER BY id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Type, &e.LockerName, &e.SlotNo, &e.Details, &e.Person); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) EventPeople(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT DISTINCT person FROM ("+eventRows+") WHERE person <> '' ORDER BY person")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
