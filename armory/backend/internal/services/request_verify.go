package services

import (
	"fmt"
	"slices"

	"armory/internal/models"
)

func VerifyReturn(req models.Request, slots []models.Slot, wrong []int64) []string {
	reading := make(map[int64]int64, len(slots))
	for _, sl := range slots {
		reading[sl.SlotNo] = sl.Reading
	}

	var problems []string
	for _, rs := range req.Slots {
		switch reading[rs.SlotNo] {
		case 1:
		case 2:
			problems = append(problems, fmt.Sprintf("Slot %d is not detecting.", rs.SlotNo))
		default:
			problems = append(problems, fmt.Sprintf("Slot %d is empty, the gun is not back.", rs.SlotNo))
		}
	}
	for _, no := range wrong {
		if slices.ContainsFunc(req.Slots, func(rs models.RequestSlot) bool { return rs.SlotNo == no }) {
			continue
		}
		problems = append(problems, fmt.Sprintf("Slot %d is empty, a gun was taken that was not assigned.", no))
	}
	return problems
}
