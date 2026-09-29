package models

type Locker struct {
	ID        int64
	Name      string
	Location  string
	IPAddress string
	Kind      string
	Capacity  int64
	Slots     []Slot
}

type Slot struct {
	ID       int64
	LockerID int64
	SlotNo   int64
	SensorID int64
	Active   bool
	Reading  int64
}
