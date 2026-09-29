package services

import "errors"

var (
	ErrNameRequired    = errors.New("name is required")
	ErrLockerExists    = errors.New("locker already exists")
	ErrLockerNotFound  = errors.New("locker not found")
	ErrInvalidIP       = errors.New("IP address is not valid")
	ErrInvalidKind     = errors.New("choose pistol or rifle")
	ErrInvalidCapacity = errors.New("capacity must be between 1 and 5")
)
