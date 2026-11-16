package database

import "errors"

var (
	ErrDuplicate       = errors.New("already exists")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrInUse           = errors.New("in use")
	ErrSensorDisabled  = errors.New("sensor disabled")
	ErrNoFreeSlot      = errors.New("no free slot")
)
