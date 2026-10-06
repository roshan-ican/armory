package models

type Event struct {
	ID         int64
	OccurredAt string
	Type       string
	LockerName string
	SlotNo     int64
	Details    string
	Person     string
}
