package database

import (
	"context"
	"database/sql"
	"errors"

	"armory/internal/models"
)

const userSelect = `
SELECT u.id, u.name, u.service_no, u.role, COUNT(f.id)
FROM users u
LEFT JOIN face_enrollments f ON f.user_id = u.id AND f.revoked_at IS NULL
WHERE u.active = 1`

func scanUser(row scanner) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.ServiceNo, &u.Role, &u.Faces)
	return u, err
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *Store) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	ts := now()
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO users (name, service_no, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		u.Name, u.ServiceNo, u.Role, ts, ts)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, ErrDuplicate
		}
		return models.User{}, err
	}
	u.ID, err = res.LastInsertId()
	return u, err
}

func (s *Store) CreateAdmin(ctx context.Context, name, username, passwordHash string) (models.User, error) {
	ts := now()
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO users (name, service_no, role, password_hash, created_at, updated_at) VALUES (?, ?, 'admin', ?, ?, ?)",
		name, username, passwordHash, ts, ts)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, ErrDuplicate
		}
		return models.User{}, err
	}
	id, err := res.LastInsertId()
	return models.User{ID: id, Name: name, ServiceNo: username, Role: "admin"}, err
}

func (s *Store) CountPasswordAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE active = 1 AND role = 'admin' AND password_hash IS NOT NULL").Scan(&n)
	return n, err
}

func (s *Store) AdminCredentials(ctx context.Context, username string) (models.User, string, error) {
	var u models.User
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT id, name, service_no, role, password_hash FROM users
		WHERE active = 1 AND role = 'admin' AND service_no = ? AND password_hash IS NOT NULL`, username).
		Scan(&u.ID, &u.Name, &u.ServiceNo, &u.Role, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, "", ErrNotFound
	}
	return u, hash, err
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx, userSelect+" GROUP BY u.id ORDER BY u.name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) GetUser(ctx context.Context, id int64) (models.User, error) {
	u, err := scanUser(s.db.QueryRowContext(ctx, userSelect+" AND u.id = ? GROUP BY u.id", id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

// ReplaceFaceEnrollments revokes the user's current faces and stores the new ones in one step.
func (s *Store) ReplaceFaceEnrollments(ctx context.Context, userID int64, refs []string, modelVersion string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var active int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = ? AND active = 1", userID).Scan(&active)
	if err != nil {
		return err
	}
	if active == 0 {
		return ErrNotFound
	}

	ts := now()
	_, err = tx.ExecContext(ctx,
		"UPDATE face_enrollments SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL", ts, userID)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO face_enrollments (user_id, face_ref, model_version, created_at) VALUES (?, ?, ?, ?)",
			userID, ref, modelVersion, ts)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListFaceEnrollments(ctx context.Context) ([]models.FaceEnrollment, error) {
	rows, err := s.db.QueryContext(ctx, `
	SELECT u.id, u.name, u.service_no, u.role, f.face_ref
	FROM face_enrollments f
	JOIN users u ON u.id = f.user_id
	WHERE f.revoked_at IS NULL AND u.active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.FaceEnrollment
	for rows.Next() {
		var e models.FaceEnrollment
		if err := rows.Scan(&e.User.ID, &e.User.Name, &e.User.ServiceNo, &e.User.Role, &e.FaceRef); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
