package services

import (
	"armory/internal/database"
	"armory/internal/models"
	"context"
	"errors"
	"strings"
)

type CategoryService struct {
	store *database.Store
}

func NewCategoryService(store *database.Store) *CategoryService {
	return &CategoryService{store: store}
}

func (s *CategoryService) List(ctx context.Context) ([]models.Category, error) {
	return s.store.ListCategories(ctx)
}

func (s *CategoryService) Create(ctx context.Context, name string) (models.Category, error) {
	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return models.Category{}, ErrNameRequired
	}
	c, err := s.store.CreateCategory(ctx, name)
	if errors.Is(err, database.ErrDuplicate) {
		return models.Category{}, ErrCategoryExists
	}
	return c, err
}
