package services

import "errors"

var (
	ErrNameRequired     = errors.New("name is required")
	ErrCategoryExists   = errors.New("category already exists")
	ErrLockerExists     = errors.New("locker already exists")
	ErrInvalidIP        = errors.New("IP address is not valid")
	ErrCategoryRequired = errors.New("category is required")
	ErrSerialRequired   = errors.New("serial number is required")
	ErrGunExists        = errors.New("a gun with this serial already exists")
	ErrSlotOccupied     = errors.New("that slot already has a gun")
	ErrCannotRetire     = errors.New("only guns that are in the locker can be retired")
	ErrSlotRequired     = errors.New("choose a slot for the gun")
	ErrCannotEdit       = errors.New("only guns that are in the locker can be edited")
)
