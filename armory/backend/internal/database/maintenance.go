package database

import (
	"context"
	"database/sql"
	"errors"

	"armory/internal/models"
)

const openMaintenance = "status IN ('assigned', 'active')"

func (s *Store) AssignMaintenance(ctx context.Context, userID, adminID int64) (models.MaintenanceJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	defer tx.Rollback()

	var role string
	err = tx.QueryRowContext(ctx, "SELECT role FROM users WHERE id = ? AND active = 1", userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return models.MaintenanceJob{}, ErrNotFound
	}
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	if role != "requester" {
		return models.MaintenanceJob{}, ErrConflict
	}

	ts := now()
	res, err := tx.ExecContext(ctx,
		"INSERT INTO maintenance_jobs (user_id, status, assigned_by, created_at) VALUES (?, 'assigned', ?, ?)",
		userID, adminID, ts)
	if err != nil {
		if isUniqueViolation(err) {
			return models.MaintenanceJob{}, ErrDuplicate
		}
		return models.MaintenanceJob{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	if err := logMaintenance(ctx, tx, "maintenance_assigned", userID, "assigned by an admin"); err != nil {
		return models.MaintenanceJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.MaintenanceJob{}, err
	}
	return models.MaintenanceJob{ID: id, UserID: userID, Status: models.MaintenanceAssigned}, nil
}

func (s *Store) StartMaintainceByAdmin(ctx context.Context, userId, adminId int64) (models.MaintenanceJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	defer tx.Rollback()

	var role string
	err = tx.QueryRowContext(ctx, "SELECT role FROM users WHERE id = ? AND active = 1", userId).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return models.MaintenanceJob{}, ErrNotFound
	}
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	if role != "requester" {
		return models.MaintenanceJob{}, ErrConflict
	}

	ts := now()
	res, err := tx.ExecContext(ctx,
		"INSERT INTO maintenance_jobs (user_id, status, assigned_by, started_at, created_at) VALUES (?, 'active', ?, ?, ?)",
		userId, adminId, ts, ts)
	if err != nil {
		if isUniqueViolation(err) {
			return models.MaintenanceJob{}, ErrDuplicate
		}
		return models.MaintenanceJob{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO maintenance_snapshots (job_id, locker_id, capacity, guns_present)
		SELECT ?, l.id, l.capacity,
			(SELECT COUNT(*) FROM slots s WHERE s.locker_id = l.id AND s.active = 1 AND s.reading = 1)
		FROM lockers l`, id); err != nil {
		return models.MaintenanceJob{}, err
	}
	if err := logMaintenance(ctx, tx, "maintenance_started", userId, "started by an admin, no tablet"); err != nil {
		return models.MaintenanceJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.MaintenanceJob{}, err
	}
	return models.MaintenanceJob{ID: id, UserID: userId, Status: models.MaintenanceActive}, nil
}

func (s *Store) FinishMaintenanceByID(ctx context.Context, id int64) error {
	return s.endMaintenance(ctx, "id = ? AND status = 'active'", id,
		models.MaintenanceFinished, "maintenance_finished", "marked finished by an admin")
}

func (s *Store) ListActiveMaintenance(ctx context.Context) ([]models.MaintenanceJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.user_id, u.name, m.status FROM maintenance_jobs m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.status = 'active' ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.MaintenanceJob
	for rows.Next() {
		var j models.MaintenanceJob
		if err := rows.Scan(&j.ID, &j.UserID, &j.UserName, &j.Status); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
func (s *Store) RequestMaintenance(ctx context.Context, userID int64, changes []models.MaintenanceChange) (models.MaintenanceJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.MaintenanceJob{}, err
	}

	defer tx.Rollback()

	var role string
	err = tx.QueryRowContext(ctx, "SELECT role FROM users WHERE id = ? AND active = 1", userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return models.MaintenanceJob{}, ErrNotFound
	}
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	if role != "requester" {
		return models.MaintenanceJob{}, ErrConflict
	}

	res, err := tx.ExecContext(ctx, "INSERT INTO maintenance_jobs (user_id, status, created_at) VALUES (?, 'requested', ?)", userID, now())
	if err != nil {
		if isUniqueViolation(err) {
			return models.MaintenanceJob{}, ErrDuplicate
		}
		return models.MaintenanceJob{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.MaintenanceJob{}, err
	}

	for i := range changes {
		c := &changes[i]
		slot := sql.NullInt64{Int64: c.SlotNo, Valid: c.SlotNo > 0}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO maintenance_changes (job_id, locker_id, action, slot_no) VALUES (?, ?, ?, ?)",
			id, c.LockerID, c.Action, slot)
		if err != nil {
			return models.MaintenanceJob{}, err
		}
		c.JobID = id
		if c.ID, err = res.LastInsertId(); err != nil {
			return models.MaintenanceJob{}, err
		}
	}

	if err := logMaintenance(ctx, tx, "maintenance_requested", userID, "requested at the kiosk"); err != nil {
		return models.MaintenanceJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.MaintenanceJob{}, err
	}
	return models.MaintenanceJob{ID: id, UserID: userID, Status: models.MaintenanceRequested, Changes: changes}, nil
}

func (s *Store) OpenMaintenanceFor(ctx context.Context, userID int64) (models.MaintenanceJob, error) {
	var j models.MaintenanceJob
	err := s.db.QueryRowContext(ctx,
		"SELECT id, user_id, status FROM maintenance_jobs WHERE user_id = ? AND "+openMaintenance, userID).
		Scan(&j.ID, &j.UserID, &j.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return models.MaintenanceJob{}, ErrNotFound
	}
	return j, err
}

func (s *Store) StartMaintenance(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int64
	var status string
	err = tx.QueryRowContext(ctx, "SELECT user_id, status FROM maintenance_jobs WHERE id = ?", id).Scan(&userID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == models.MaintenanceActive {
		return nil
	}
	if status != models.MaintenanceAssigned {
		return ErrConflict
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE maintenance_jobs SET status = 'active', started_at = ? WHERE id = ?", now(), id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO maintenance_snapshots (job_id, locker_id, capacity, guns_present)
		SELECT ?, l.id, l.capacity,
			(SELECT COUNT(*) FROM slots s WHERE s.locker_id = l.id AND s.active = 1 AND s.reading = 1)
		FROM lockers l`, id); err != nil {
		return err
	}
	if err := logMaintenance(ctx, tx, "maintenance_started", userID, "door opened for the work"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) FinishMaintenance(ctx context.Context, userID int64) error {
	return s.endMaintenance(ctx, "user_id = ? AND status = 'active'", userID,
		models.MaintenanceFinished, "maintenance_finished", "marked finished at the tablet")
}

func (s *Store) ApproveMaintenance(ctx context.Context, id, adminID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int64
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM maintenance_jobs WHERE id = ? AND status = 'requested'", id).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE maintenance_jobs SET status = 'assigned', assigned_by = ? WHERE id = ?", adminID, id); err != nil {
		return err
	}
	if err := logMaintenance(ctx, tx, "maintenance_approved", userID, "approved by an admin"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CancelMaintenance(ctx context.Context, id int64) error {
	return s.endMaintenance(ctx, "id = ? AND status IN ('requested', 'assigned', 'active')", id,
		models.MaintenanceCancelled, "maintenance_cancelled", "cancelled by an admin")
}

func (s *Store) endMaintenance(ctx context.Context, where string, arg int64, status, event, details string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var id, userID int64
	err = tx.QueryRowContext(ctx, "SELECT id, user_id FROM maintenance_jobs WHERE "+where, arg).Scan(&id, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE maintenance_jobs SET status = ?, ended_at = ? WHERE id = ?", status, now(), id); err != nil {
		return err
	}
	if err := logMaintenance(ctx, tx, event, userID, details); err != nil {
		return err
	}
	return tx.Commit()
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func logMaintenance(ctx context.Context, tx execer, event string, userID int64, details string) error {
	_, err := tx.ExecContext(ctx,
		"INSERT INTO events (occurred_at, type, user_id, details) VALUES (?, ?, ?, ?)", now(), event, userID, details)
	return err
}

func (s *Store) ListMaintenanceChanges(ctx context.Context, jobID int64) ([]models.MaintenanceChange, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.job_id, c.locker_id, l.name, c.action, COALESCE(c.slot_no, 0)
		 FROM maintenance_changes c JOIN lockers l ON l.id = c.locker_id
		 WHERE c.job_id = ? ORDER BY c.id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []models.MaintenanceChange
	for rows.Next() {
		var c models.MaintenanceChange
		if err := rows.Scan(&c.ID, &c.JobID, &c.LockerID, &c.LockerName, &c.Action, &c.SlotNo); err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, rows.Err()
}

func (s *Store) ListMaintenanceSnapshot(ctx context.Context, jobID int64) ([]models.LockerSnapshot, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT locker_id, capacity, guns_present FROM maintenance_snapshots WHERE job_id = ? ORDER BY locker_id", jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snap []models.LockerSnapshot
	for rows.Next() {
		var l models.LockerSnapshot
		if err := rows.Scan(&l.LockerID, &l.Capacity, &l.GunsPresent); err != nil {
			return nil, err
		}
		snap = append(snap, l)
	}
	return snap, rows.Err()
}

func (s *Store) FlagMaintenance(ctx context.Context, userID int64, details string) error {
	return logMaintenance(ctx, s.db, "maintenance_flagged", userID, details)
}

const liveMaintenance = "status IN ('requested', 'assigned', 'active')"

func (s *Store) CurrentMaintenanceFor(ctx context.Context, userID int64) (models.MaintenanceJob, error) {
	var j models.MaintenanceJob
	err := s.db.QueryRowContext(ctx,
		"SELECT id, user_id, status FROM maintenance_jobs WHERE user_id = ? AND "+liveMaintenance, userID).
		Scan(&j.ID, &j.UserID, &j.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return models.MaintenanceJob{}, ErrNotFound
	}
	if err != nil {
		return models.MaintenanceJob{}, err
	}
	j.Changes, err = s.ListMaintenanceChanges(ctx, j.ID)
	return j, err
}

func (s *Store) ListRequestedMaintenance(ctx context.Context) ([]models.MaintenanceJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.user_id, u.name, m.status FROM maintenance_jobs m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.status = 'requested' AND u.active = 1 ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	var jobs []models.MaintenanceJob
	for rows.Next() {
		var j models.MaintenanceJob
		if err := rows.Scan(&j.ID, &j.UserID, &j.UserName, &j.Status); err != nil {
			rows.Close()
			return nil, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].Changes, err = s.ListMaintenanceChanges(ctx, jobs[i].ID); err != nil {
			return nil, err
		}
	}
	return jobs, nil
}

func (s *Store) MaintenanceActive(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM maintenance_jobs WHERE status = 'active'").Scan(&n)
	return n > 0, err
}
