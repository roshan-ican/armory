package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"armory/internal/models"
)

func TestWrongPickMessage(t *testing.T) {
	cases := []struct {
		name     string
		picked   int64
		expected []int64
		want     string
	}{
		{"Roshan Sahani", 2, []int64{1}, "Roshan, you picked the gun from slot 2. That is incorrect. Please pick from slot 1, as you selected."},
		{"Roshan", 3, []int64{1, 2}, "Roshan, you picked the gun from slot 3. That is incorrect. Please pick from slots 1 and 2, as you selected."},
		{"Roshan", 4, []int64{1, 2, 3}, "Roshan, you picked the gun from slot 4. That is incorrect. Please pick from slots 1, 2 and 3, as you selected."},
		{"Roshan", 2, nil, "Roshan, you picked the gun from slot 2. That is incorrect. Please put it back."},
		{"", 2, []int64{1}, "there, you picked the gun from slot 2. That is incorrect. Please pick from slot 1, as you selected."},
	}
	for _, c := range cases {
		if got := WrongPickMessage(c.name, c.picked, c.expected); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestApprovedMessage(t *testing.T) {
	morning := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		slots []int64
		door  bool
		want  string
	}{
		{[]int64{1}, true, "Good morning Roshan. Your request is approved. The door is open. Locker 1 is open. Please take slot 1."},
		{[]int64{1, 2}, true, "Good morning Roshan. Your request is approved. The door is open. Locker 1 is open. Please take slots 1 and 2."},
		{[]int64{1}, false, "Good morning Roshan. Your request is approved. Locker 1 is open. Please take slot 1."},
		{nil, true, "Good morning Roshan. Your request is approved. The door is open. Locker 1 is open."},
	}
	for _, c := range cases {
		if got := ApprovedMessage("Roshan Sahani", "Locker 1", c.slots, c.door, morning); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestGreetingFollowsTheHour(t *testing.T) {
	cases := map[int]string{
		0: "Good morning", 5: "Good morning", 11: "Good morning",
		12: "Good afternoon", 17: "Good afternoon",
		18: "Good evening", 23: "Good evening",
	}
	for hour, want := range cases {
		if got := Greeting(time.Date(2026, 10, 7, hour, 59, 0, 0, time.UTC)); got != want {
			t.Errorf("hour %d: got %q, want %q", hour, got, want)
		}
	}
}

func TestApproveSpeaksThatTheLockerIsOpen(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case text := <-spoken:
		if !strings.HasPrefix(text, "Good ") || !strings.Contains(text, " Roshan. Your request is approved.") ||
			!strings.Contains(text, "The door is open") ||
			!strings.Contains(text, "East is open") || !strings.Contains(text, "slot 1") {
			t.Fatalf("approval said %q", text)
		}
	case <-time.After(time.Second):
		t.Fatal("approval spoke nothing")
	}
}

func TestCorrectPickMessage(t *testing.T) {
	if got, want := CorrectPickMessage("Roshan Sahani", 1), "Thank you Roshan. Slot 1 is correct."; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPendingSlotNumbersKeepsOnlyChosen(t *testing.T) {
	slots := []models.RequestSlot{
		{SlotNo: 1, Status: models.SlotCollected},
		{SlotNo: 2, Status: models.SlotChosen},
		{SlotNo: 4, Status: models.SlotChosen},
	}
	got := pendingSlotNumbers(slots)
	if len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Fatalf("got %v", got)
	}
}

func TestSlotChangedSpeaksWrongAndCorrectPicks(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	approved, err := f.svc.Approve(ctx, f.admin.ID, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	target := firstSlotID(approved)
	wrongSlot := f.locker.Slots[2].ID
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	next := func() string {
		select {
		case text := <-spoken:
			return text
		case <-time.After(time.Second):
			t.Fatal("nothing was spoken")
			return ""
		}
	}
	take := func(slotID int64) {
		if err := f.store.RecordSensorEvent(ctx, slotID, 0, "gun_removed"); err != nil {
			t.Fatal(err)
		}
		f.svc.SlotChanged(ctx, slotID, 0)
	}

	take(wrongSlot)
	wrong := next()
	if !strings.Contains(wrong, "slot 3") || !strings.Contains(wrong, "incorrect") || !strings.Contains(wrong, "slot 1") {
		t.Fatalf("wrong pick said %q", wrong)
	}

	take(target)
	right := next()
	if !strings.Contains(right, "Slot 1 is correct") {
		t.Fatalf("correct pick said %q", right)
	}
}
