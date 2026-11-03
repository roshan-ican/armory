package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"armory/internal/live"
	"armory/internal/models"
)

func TestWrongPickMessage(t *testing.T) {
	cases := []struct {
		picked   int64
		expected []int64
		want     string
	}{
		{2, []int64{1}, "You picked the gun from slot 2. That is incorrect. Please pick from slot 1, as you selected."},
		{3, []int64{1, 2}, "You picked the gun from slot 3. That is incorrect. Please pick from slots 1 and 2, as you selected."},
		{4, []int64{1, 2, 3}, "You picked the gun from slot 4. That is incorrect. Please pick from slots 1, 2 and 3, as you selected."},
		{2, nil, "You picked the gun from slot 2. Please do not take more than you requested. Put it back."},
	}
	for _, c := range cases {
		if got := WrongPickMessage(c.picked, c.expected); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestAdminWrongPickMessage(t *testing.T) {
	cases := []struct {
		name     string
		picked   int64
		expected []int64
		want     string
	}{
		{"Roshan", 3, []int64{2}, "Roshan has picked a wrong gun. Gun 3 was picked instead of 2."},
		{"Roshan", 4, []int64{1, 2}, "Roshan has picked a wrong gun. Gun 4 was picked instead of 1 and 2."},
		{"Roshan", 2, nil, "Roshan has taken an extra gun. Gun 2 was not requested."},
		{"", 3, []int64{2}, "Someone has picked a wrong gun. Gun 3 was picked instead of 2."},
	}
	for _, c := range cases {
		if got := AdminWrongPickMessage(c.name, c.picked, c.expected); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestAdminTakeMessages(t *testing.T) {
	if got, want := AdminTakenAfterReturnMessage("Roshan", 2), "Roshan has taken gun 2 after the request was already complete."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := AdminUnauthorizedTakeMessage(2), "Gun 2 was taken without an approved request."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWrongPickAlertsTheAdmin(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	alerts, cancel := f.hub.SubscribeAlerts()
	defer cancel()
	_, take := spokenFlow(t, f)

	take(f.locker.Slots[2].ID)

	select {
	case alert := <-alerts:
		if want := "Roshan Sahani has picked a wrong gun. Gun 3 was picked instead of 1."; alert.Text != want || !alert.Urgent {
			t.Fatalf("admin heard %+v, want urgent %q", alert, want)
		}
	case <-time.After(time.Second):
		t.Fatal("admin was not alerted to the wrong pick")
	}
}

func TestAdminCorrectPickMessage(t *testing.T) {
	if got, want := AdminCorrectPickMessage("Roshan", 1, false), "Roshan has picked the right gun. Gun 1."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	want := "Roshan has picked the right gun. Gun 3. All requested guns are collected."
	if got := AdminCorrectPickMessage("Roshan", 3, true); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := AdminCorrectPickMessage("", 1, false), "Someone has picked the right gun. Gun 1."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAdminReturnedMessage(t *testing.T) {
	if got, want := AdminReturnedMessage("Roshan", 1, false), "Roshan has returned gun 1."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := AdminReturnedMessage("Roshan", 3, true), "Roshan has returned gun 3. All guns are back."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := AdminReturnedMessage("", 1, false), "Someone has returned gun 1."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReturningGunsNoticesTheAdminByName(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{1, 3}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	next, take := spokenFlow(t, f)
	take(f.locker.Slots[0].ID)
	next()
	take(f.locker.Slots[2].ID)
	next()
	time.Sleep(50 * time.Millisecond)

	alerts, cancel := f.hub.SubscribeAlerts()
	defer cancel()
	put := func(slotID int64) {
		if err := f.store.RecordSensorEvent(ctx, slotID, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
		f.svc.SlotChanged(ctx, slotID, 1)
	}
	told := func() live.AdminAlert {
		select {
		case alert := <-alerts:
			return alert
		case <-time.After(time.Second):
			t.Fatal("admin was told nothing")
			return live.AdminAlert{}
		}
	}

	put(f.locker.Slots[0].ID)
	if got, want := told(), (live.AdminAlert{Text: "Roshan Sahani has returned gun 1."}); got != want {
		t.Fatalf("first return told admin %+v, want %+v", got, want)
	}

	put(f.locker.Slots[2].ID)
	want := live.AdminAlert{Text: "Roshan Sahani has returned gun 3. All guns are back."}
	if got := told(); got != want {
		t.Fatalf("last return told admin %+v, want %+v", got, want)
	}
}

func TestCorrectPickNoticesTheAdmin(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{1, 3}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	alerts, cancel := f.hub.SubscribeAlerts()
	defer cancel()
	_, take := spokenFlow(t, f)

	next := func() live.AdminAlert {
		select {
		case alert := <-alerts:
			return alert
		case <-time.After(time.Second):
			t.Fatal("admin was told nothing")
			return live.AdminAlert{}
		}
	}

	take(f.locker.Slots[0].ID)
	if got, want := next(), (live.AdminAlert{Text: "Roshan Sahani has picked the right gun. Gun 1."}); got != want {
		t.Fatalf("first gun told admin %+v, want %+v", got, want)
	}

	take(f.locker.Slots[2].ID)
	want := live.AdminAlert{Text: "Roshan Sahani has picked the right gun. Gun 3. All requested guns are collected."}
	if got := next(); got != want {
		t.Fatalf("last gun told admin %+v, want %+v", got, want)
	}
}

func TestApprovedMessage(t *testing.T) {
	cases := []struct {
		slots []int64
		door  bool
		want  string
	}{
		{[]int64{1}, true, "Your request is approved. The door is open. Locker 1 is open. Please take slot 1."},
		{[]int64{1, 2}, true, "Your request is approved. The door is open. Locker 1 is open. Please take slots 1 and 2."},
		{[]int64{1}, false, "Your request is approved. Locker 1 is open. Please take slot 1."},
		{nil, true, "Your request is approved. The door is open. Locker 1 is open."},
	}
	for _, c := range cases {
		if got := ApprovedMessage("Locker 1", c.slots, c.door); got != c.want {
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
		want := "Your request is approved. The door is open. East is open. Please take slot 1."
		if text != want {
			t.Fatalf("approval said %q, want %q", text, want)
		}
	case <-time.After(time.Second):
		t.Fatal("approval spoke nothing")
	}
}

func TestCorrectPickMessage(t *testing.T) {
	if got, want := CorrectPickMessage(1, false), "Slot 1 is correct."; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	want := "Slot 1 is correct. Please close the locker and close the door as you leave."
	if got := CorrectPickMessage(1, true); got != want {
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

func spokenFlow(t *testing.T, f flow) (func() string, func(int64)) {
	t.Helper()
	ctx := context.Background()
	spoken, cancel := f.hub.SubscribeSpeech()
	t.Cleanup(cancel)
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
	return next, take
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
	next, take := spokenFlow(t, f)

	take(f.locker.Slots[2].ID)
	wantWrong := "You picked the gun from slot 3. That is incorrect. Please pick from slot 1, as you selected."
	if got := next(); got != wantWrong {
		t.Fatalf("wrong pick said %q, want %q", got, wantWrong)
	}

	take(firstSlotID(approved))
	wantRight := "Slot 1 is correct. Please close the locker and close the door as you leave."
	if got := next(); got != wantRight {
		t.Fatalf("correct pick said %q, want %q", got, wantRight)
	}
}

func TestLastGunOfSeveralTellsThemToCloseUp(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{1, 3}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	next, take := spokenFlow(t, f)

	take(f.locker.Slots[0].ID)
	if first := next(); first != "Slot 1 is correct." {
		t.Fatalf("first gun said %q", first)
	}

	take(f.locker.Slots[2].ID)
	want := "Slot 3 is correct. Please close the locker and close the door as you leave."
	if last := next(); last != want {
		t.Fatalf("last gun said %q, want %q", last, want)
	}
}

func TestReturningTheLastGunThanksAndAsksToCloseTheDoor(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{1, 3}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	next, take := spokenFlow(t, f)
	take(f.locker.Slots[0].ID)
	next()
	take(f.locker.Slots[2].ID)
	next()
	time.Sleep(50 * time.Millisecond)

	put := func(slotID int64) {
		if err := f.store.RecordSensorEvent(ctx, slotID, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
		f.svc.SlotChanged(ctx, slotID, 1)
	}

	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	put(f.locker.Slots[0].ID)
	select {
	case text := <-spoken:
		t.Fatalf("returning the first of two guns said %q", text)
	case <-time.After(100 * time.Millisecond):
	}

	put(f.locker.Slots[2].ID)
	select {
	case got := <-spoken:
		if want := ReturnedMessage(); got != want {
			t.Fatalf("last gun back said %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("nothing was spoken when the last gun came back")
	}
}

func TestNoServerMessageRepeatsThePersonsName(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.Create(ctx, f.user.ID, "rifle", "")
	if err != nil {
		t.Fatal(err)
	}
	next, take := spokenFlow(t, f)
	approved, err := f.svc.Approve(ctx, f.admin.ID, req.ID)
	if err != nil {
		t.Fatal(err)
	}

	lines := []string{next()}
	take(f.locker.Slots[2].ID)
	lines = append(lines, next())
	take(firstSlotID(approved))
	lines = append(lines, next())

	for _, line := range lines {
		if strings.Contains(line, "Roshan") || strings.Contains(line, "Sahani") {
			t.Fatalf("message repeats the name: %q", line)
		}
	}
}
