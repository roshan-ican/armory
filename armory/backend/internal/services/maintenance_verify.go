package services

import (
	"fmt"
	"slices"

	"armory/internal/models"
)

func sensorWorking(sl models.Slot) bool {
	return sl.Reading == 0 || sl.Reading == 1
}

func VerifyMaintenance(snap []models.LockerSnapshot, changes []models.MaintenanceChange, lockers []models.Locker) []string {
	before := make(map[int64]models.LockerSnapshot, len(snap))
	for _, s := range snap {
		before[s.LockerID] = s
	}

	var problems []string
	for _, l := range lockers {
		s, ok := before[l.ID]
		if !ok {
			continue
		}

		var adds, removes int64
		var replaced []int64
		for _, c := range changes {
			if c.LockerID != l.ID {
				continue
			}
			switch c.Action {
			case models.ChangeAddSensor:
				adds++
			case models.ChangeRemoveSensor:
				removes++
			case models.ChangeReplaceSensor:
				replaced = append(replaced, c.SlotNo)
			}
		}

		start := len(problems)

		expected := s.Capacity + adds - removes
		switch {
		case l.Capacity == expected:
		case l.Capacity > expected && removes > 0:
			problems = append(problems, fmt.Sprintf("%s still has the sensor that should be removed.", l.Name))
		case l.Capacity < expected && adds > 0:
			problems = append(problems, fmt.Sprintf("%s is missing the sensor that should be added.", l.Name))
		default:
			problems = append(problems, fmt.Sprintf("%s has %d sensors but should have %d.", l.Name, l.Capacity, expected))
		}

		guns := int64(0)
		for _, sl := range l.Slots {
			if sl.Reading == 1 {
				guns++
			}
			if (sl.SlotNo > s.Capacity || slices.Contains(replaced, sl.SlotNo)) && !sensorWorking(sl) {
				problems = append(problems, fmt.Sprintf("The sensor in slot %d of %s is not working.", sl.SlotNo, l.Name))
			}
		}

		if len(problems) == start && adds+removes == 0 && guns != s.GunsPresent {
			problems = append(problems, fmt.Sprintf("%s has %d guns but had %d before the maintenance.", l.Name, guns, s.GunsPresent))
		}
	}
	return problems
}
