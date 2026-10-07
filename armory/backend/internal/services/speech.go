package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"armory/internal/models"
)

func firstName(full string) string {
	if fields := strings.Fields(full); len(fields) > 0 {
		return fields[0]
	}
	return "there"
}

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

func WrongPickMessage(name string, picked int64, expected []int64) string {
	who := firstName(name)
	if len(expected) == 0 {
		return fmt.Sprintf("%s, you picked the gun from slot %d. That is incorrect. Please put it back.", who, picked)
	}
	noun := "slot"
	if len(expected) > 1 {
		noun = "slots"
	}
	return fmt.Sprintf("%s, you picked the gun from slot %d. That is incorrect. Please pick from %s %s, as you selected.",
		who, picked, noun, slotWords(expected))
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

func ApprovedMessage(name, locker string, slots []int64, door bool, at time.Time) string {
	text := fmt.Sprintf("%s %s. Your request is approved.", Greeting(at), firstName(name))
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

func CorrectPickMessage(name string, picked int64) string {
	return fmt.Sprintf("Thank you %s. Slot %d is correct.", firstName(name), picked)
}
