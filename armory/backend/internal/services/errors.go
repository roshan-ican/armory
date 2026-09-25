package services

import "errors"

var (
	ErrNameRequired   = errors.New("name is required")
	ErrCategoryExists = errors.New("category already exists")
	ErrLockerExists   = errors.New("locker already exists")
	ErrInvalidIP      = errors.New("IP address is not valid")
)
