package services

import "errors"

var (
	ErrNameRequired   = errors.New("name is required")
	ErrCategoryExists = errors.New("category already exists")
)
