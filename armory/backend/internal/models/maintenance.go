package models

import "fmt"

const (
	MaintenanceRequested = "requested"
	MaintenanceAssigned  = "assigned"
	MaintenanceActive    = "active"
	MaintenanceFinished  = "finished"
	MaintenanceCancelled = "cancelled"
)

const (
	ChangeRemoveSensor  = "remove_sensor"
	ChangeAddSensor     = "add_sensor"
	ChangeReplaceSensor = "replace_sensor"
)

type MaintenanceChange struct {
	ID         int64
	JobID      int64
	LockerID   int64
	LockerName string
	Action     string
	SlotNo     int64
}

func (c MaintenanceChange) Text() string {
	switch c.Action {
	case ChangeAddSensor:
		return "Add a sensor to " + c.LockerName
	case ChangeRemoveSensor:
		return "Remove a sensor from " + c.LockerName
	case ChangeReplaceSensor:
		return fmt.Sprintf("Replace the sensor in slot %d of %s", c.SlotNo, c.LockerName)
	}
	return c.Action
}

type MaintenanceJob struct {
	ID       int64
	UserID   int64
	UserName string
	Status   string
	Changes  []MaintenanceChange
}

type LockerSnapshot struct {
	LockerID    int64
	Capacity    int64
	GunsPresent int64
}
