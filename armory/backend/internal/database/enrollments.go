package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"armory/internal/models"
)

func (s *Store) CreateFaceEnrollmentRequest(ctx context.Context, name, serviceNo, faceRefs, modelVersion string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO face_enrollment_requests
		(name, service_no, face_refs, model_version, created_at) VALUES (?, ?, ?, ?, ?)`,
		name, serviceNo, faceRefs, modelVersion, now())
	if err != nil && isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (s *Store) ListPendingFaceEnrollmentRequests(ctx context.Context) ([]models.FaceEnrollmentRequest, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, service_no, face_refs, status, created_at
		FROM face_enrollment_requests WHERE status = 'pending' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.FaceEnrollmentRequest
	for rows.Next() {
		var item models.FaceEnrollmentRequest
		if err := rows.Scan(&item.ID, &item.Name, &item.ServiceNo, &item.FaceRefs, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) DecideFaceEnrollmentRequest(ctx context.Context, id, adminID int64, approve bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var item models.FaceEnrollmentRequest
	var modelVersion string
	err = tx.QueryRowContext(ctx, `SELECT id, name, service_no, face_refs, status, created_at, model_version
		FROM face_enrollment_requests WHERE id = ?`, id).
		Scan(&item.ID, &item.Name, &item.ServiceNo, &item.FaceRefs, &item.Status, &item.CreatedAt, &modelVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if item.Status != "pending" {
		return ErrConflict
	}
	status := "rejected"
	if approve {
		status = "approved"
		ts := now()
		var userID int64
		var role string
		err := tx.QueryRowContext(ctx, "SELECT id, role FROM users WHERE service_no = ? OR upper(name) = ?", item.ServiceNo, item.ServiceNo).Scan(&userID, &role)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := tx.ExecContext(ctx, `INSERT INTO users (name, service_no, role, created_at, updated_at)
				VALUES (?, ?, 'requester', ?, ?)`, item.Name, item.ServiceNo, ts, ts)
			if err != nil {
				if isUniqueViolation(err) {
					return ErrDuplicate
				}
				return err
			}
			userID, err = res.LastInsertId()
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if role != "requester" {
			return ErrDuplicate
		} else {
			if _, err := tx.ExecContext(ctx, "UPDATE face_enrollments SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL", ts, userID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE users SET active = 1, updated_at = ? WHERE id = ?", ts, userID); err != nil {
				return err
			}
		}
		var refs []string
		if err := json.Unmarshal([]byte(item.FaceRefs), &refs); err != nil {
			return err
		}
		for _, ref := range refs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO face_enrollments
				(user_id, face_ref, model_version, enrolled_by, created_at) VALUES (?, ?, ?, ?, ?)`,
				userID, ref, modelVersion, adminID, ts); err != nil {
				return err
			}
		}
	}
	res, err := tx.ExecContext(ctx, `UPDATE face_enrollment_requests
		SET status = ?, decided_by = ?, decided_at = ? WHERE id = ? AND status = 'pending'`, status, adminID, now(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrConflict
	}
	return tx.Commit()
}
