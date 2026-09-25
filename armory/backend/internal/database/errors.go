package database

import "errors"

var (
	ErrDuplicate = errors.New("already exists")
	ErrSlotTaken = errors.New("slot already has a gun")
	ErrNotFound  = errors.New("not found")
)
