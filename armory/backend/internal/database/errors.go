package database

import "errors"

var (
	ErrDuplicate       = errors.New("already exists")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrInUse           = errors.New("in use")
	ErrNoFreeSlot      = errors.New("no free slot")
	ErrRequestNotFound = errors.New("request not found")
	ErrRequestClosed   = errors.New("this request was already handled")
	ErrNoneAvailable   = errors.New("none of that type is available right now")
	ErrNotAdmin        = errors.New("only an admin can do this")
	ErrRequestOpen     = errors.New("you already have a request in progress")
	ErrInvalidReturn   = errors.New("choose how long you need it")
)
