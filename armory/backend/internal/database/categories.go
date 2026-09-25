package database

import (
	"armory/internal/models"
	"context"
)

func (s *Store) ListCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, active FROM categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Active); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (s *Store) CreateCategory(ctx context.Context, name string) (models.Category, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO categories (name, active) VALUES (?, ?)", name, true)
	if err != nil {
		if isUniqueViolation(err) {
			return models.Category{}, ErrDuplicate
		}
		return models.Category{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.Category{}, err
	}
	return models.Category{ID: id, Name: name, Active: true}, nil
}
