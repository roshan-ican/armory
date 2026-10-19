package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"armory/internal/models"
)

func slotWords(numbers []int64) string {
	parts := make([]string, len(numbers))
	for i, n := range numbers {
		parts[i] = strconv.FormatInt(n, 10)
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func pendingSlotNumbers(slots []models.RequestSlot) []int64 {
	var out []int64
	for _, s := range slots {
		if s.Status == models.SlotChosen {
			out = append(out, s.SlotNo)
		}
	}
	return out
}

func WrongPickMessage(picked int64, expected []int64) string {
	if len(expected) == 0 {
		return fmt.Sprintf("You picked the gun from slot %d. Please do not take more than you requested. Put it back.", picked)
	}
	noun := "slot"
	if len(expected) > 1 {
		noun = "slots"
	}
	return fmt.Sprintf("You picked the gun from slot %d. That is incorrect. Please pick from %s %s, as you selected.",
		picked, noun, slotWords(expected))
}

func Greeting(at time.Time) string {
	switch hour := at.Hour(); {
	case hour < 12:
		return "Good morning"
	case hour < 18:
		return "Good afternoon"
	}
	return "Good evening"
}

func ApprovedMessage(locker string, slots []int64, door bool) string {
	text := "Your request is approved."
	if door {
		text += " The door is open."
	}
	text += fmt.Sprintf(" %s is open.", locker)
	if len(slots) > 0 {
		noun := "slot"
		if len(slots) > 1 {
			noun = "slots"
		}
		text += fmt.Sprintf(" Please take %s %s.", noun, slotWords(slots))
	}
	return text
}

func ReturnedMessage() string {
	return "Thank you. Please close the locker door as you leave."
}

func TakenAfterReturnMessage(picked int64) string {
	return fmt.Sprintf("You picked the gun from slot %d. That is not part of your request, which is already complete. Please put it back.", picked)
}

func UnauthorizedTakeMessage(picked int64) string {
	return fmt.Sprintf("A gun was taken from slot %d without an approved request. Please put it back.", picked)
}

func CloseProblemsMessage(problems []string) string {
	return "Collection is not complete. " + strings.Join(problems, " ")
}

func ClosedMessage() string {
	return "Thank you. Everything is back in place."
}

func MaintenanceStartMessage() string {
	return "Please do the maintenance work, and close the door when you finish."
}

func MaintenanceDoneMessage() string {
	return "Maintenance finished. Please make sure the door is closed."
}

func MaintenanceProblemsMessage(problems []string) string {
	return "Maintenance is not finished. " + strings.Join(problems, " ")
}

func CorrectPickMessage(picked int64, last bool) string {
	text := fmt.Sprintf("Slot %d is correct.", picked)
	if last {
		text += " Please close the locker and close the door as you leave."
	}
	return text
}
