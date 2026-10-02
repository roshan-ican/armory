package models

import (
	"strconv"
	"strings"
)

const (
	RequestPending   = "pending"
	RequestApproved  = "approved"
	RequestRejected  = "rejected"
	RequestCancelled = "cancelled"
	RequestExpired   = "expired"
	RequestCollected = "collected"
	RequestReturned  = "returned"
)

const (
	SlotChosen    = "chosen"
	SlotCollected = "collected"
	SlotReturned  = "returned"
)

type RequestSlot struct {
	SlotID int64
	SlotNo int64
	Status string
}

type Request struct {
	ID                int64
	UserID            int64
	UserName          string
	ServiceNo         string
	Kind              string
	RequestedLockerID int64
	Reason            string
	Status            string
	LockerID          int64
	LockerName        string
	LockerIP          string
	Slots             []RequestSlot
	ExpectedReturnAt  string
	CreatedAt         string
}

func (r Request) Initials() string {
	out := ""
	for _, word := range strings.Fields(r.UserName) {
		out += strings.ToUpper(word[:1])
		if len(out) == 2 {
			break
		}
	}
	return out
}

func (r Request) SlotNos() []int64 {
	out := make([]int64, 0, len(r.Slots))
	for _, sl := range r.Slots {
		out = append(out, sl.SlotNo)
	}
	return out
}

func (r Request) SlotList() string {
	parts := make([]string, 0, len(r.Slots))
	for _, sl := range r.Slots {
		parts = append(parts, strconv.FormatInt(sl.SlotNo, 10))
	}
	return strings.Join(parts, ", ")
}
