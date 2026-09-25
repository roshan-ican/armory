package models

type Gun struct {
	ID           int64
	SlotID       int64
	CategoryID   int64
	CategoryName string
	LockerName   string
	SlotNo       int64
	Serial       string
	Model        string
	Notes        string
	Status       string
}

type FreeSlot struct {
	ID         int64
	LockerName string
	SlotNo     int64
}
