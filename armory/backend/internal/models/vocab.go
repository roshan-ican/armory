package models

import "slices"

const (
	RoleAdmin     = "admin"
	RoleRequester = "requester"
)

const (
	KindPistol = "pistol"
	KindRifle  = "rifle"
)

var Kinds = []string{KindPistol, KindRifle}

func ValidKind(kind string) bool {
	return slices.Contains(Kinds, kind)
}

func ValidRole(role string) bool {
	return role == RoleAdmin || role == RoleRequester
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (r Request) AwaitingClose() bool {
	return r.Status == RequestReturned && r.ClosedAt == ""
}
