package services

import (
	"reflect"
	"testing"

	"armory/internal/models"
)

func verifyLocker(id int64, name string, readings ...int64) models.Locker {
	l := models.Locker{ID: id, Name: name, Capacity: int64(len(readings))}
	for i, r := range readings {
		l.Slots = append(l.Slots, models.Slot{LockerID: id, SlotNo: int64(i + 1), Active: true, Reading: r})
	}
	return l
}

func TestVerifyMaintenance(t *testing.T) {
	snap := []models.LockerSnapshot{
		{LockerID: 1, Capacity: 3, GunsPresent: 3},
		{LockerID: 2, Capacity: 3, GunsPresent: 2},
	}
	removeFrom2 := []models.MaintenanceChange{{LockerID: 2, Action: models.ChangeRemoveSensor}}
	replace1Slot2 := []models.MaintenanceChange{{LockerID: 1, Action: models.ChangeReplaceSensor, SlotNo: 2}}
	add1 := []models.MaintenanceChange{{LockerID: 1, Action: models.ChangeAddSensor}}

	tests := []struct {
		name    string
		changes []models.MaintenanceChange
		lockers []models.Locker
		want    []string
	}{
		{
			name:    "everything as requested",
			changes: append(append([]models.MaintenanceChange{}, removeFrom2...), replace1Slot2...),
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 1)},
		},
		{
			name:    "gun missing from a locker",
			lockers: []models.Locker{verifyLocker(1, "L1", 0, 1, 1), verifyLocker(2, "L2", 1, 0, 1)},
			want:    []string{"L1 has 2 guns but had 3 before the maintenance."},
		},
		{
			name:    "guns shuffled but same count",
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 0, 1, 1)},
		},
		{
			name:    "gun count is not checked where the plan changes capacity",
			changes: removeFrom2,
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 0)},
		},
		{
			name:    "sensor not removed",
			changes: removeFrom2,
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 1, 0)},
			want:    []string{"L2 still has the sensor that should be removed."},
		},
		{
			name:    "sensor not added",
			changes: add1,
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 1, 0)},
			want:    []string{"L1 is missing the sensor that should be added."},
		},
		{
			name:    "added sensor is dead",
			changes: add1,
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1, 2), verifyLocker(2, "L2", 1, 1, 0)},
			want:    []string{"The sensor in slot 4 of L1 is not working."},
		},
		{
			name:    "replaced sensor still not detecting",
			changes: replace1Slot2,
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 2, 1), verifyLocker(2, "L2", 1, 1, 0)},
			want:    []string{"The sensor in slot 2 of L1 is not working."},
		},
		{
			name:    "replaced sensor with no reading yet",
			changes: replace1Slot2,
			lockers: []models.Locker{verifyLocker(1, "L1", 1, -1, 1), verifyLocker(2, "L2", 1, 1, 0)},
			want:    []string{"The sensor in slot 2 of L1 is not working."},
		},
		{
			name:    "unrequested capacity change",
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 1)},
			want:    []string{"L2 has 2 sensors but should have 3."},
		},
		{
			name:    "remove requested but capacity went up instead",
			changes: removeFrom2,
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 1, 0, 0)},
			want:    []string{"L2 still has the sensor that should be removed."},
		},
		{
			name:    "untouched broken sensor is not blamed",
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 0, 1)},
		},
		{
			name:    "two problems in different lockers",
			changes: append(append([]models.MaintenanceChange{}, removeFrom2...), replace1Slot2...),
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 2, 1), verifyLocker(2, "L2", 1, 1, 0)},
			want: []string{
				"The sensor in slot 2 of L1 is not working.",
				"L2 still has the sensor that should be removed.",
			},
		},
		{
			name:    "locker created during maintenance is ignored",
			lockers: []models.Locker{verifyLocker(1, "L1", 1, 1, 1), verifyLocker(2, "L2", 1, 1, 0), verifyLocker(9, "L9", 1)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VerifyMaintenance(snap, tc.changes, tc.lockers)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("problems = %q, want %q", got, tc.want)
			}
		})
	}
}
