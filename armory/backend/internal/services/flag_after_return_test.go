package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"armory/internal/models"
)

func heardFrom(t *testing.T, spoken <-chan string) string {
	t.Helper()
	select {
	case text := <-spoken:
		return text
	case <-time.After(time.Second):
		t.Fatal("nothing was spoken")
		return ""
	}
}

func silent(t *testing.T, spoken <-chan string) {
	t.Helper()
	select {
	case text := <-spoken:
		t.Fatalf("nothing should be spoken, heard %q", text)
	case <-time.After(150 * time.Millisecond):
	}
}

func (f flow) alarmOn() bool {
	return len(f.hw.alarms) > 0 && strings.HasSuffix(f.hw.alarms[len(f.hw.alarms)-1], ":true")
}

func TestATakeAfterTheReturnIsFlaggedAndCaughtAtConfirmation(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req := f.returnedRequest(t, 1)

	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	f.sensor(t, 2, 0)
	said := heardFrom(t, spoken)
	if !strings.Contains(said, "slot 2") || !strings.Contains(said, "not part of your request") {
		t.Fatalf("said %q", said)
	}
	if !f.alarmOn() {
		t.Fatalf("the locker alarm should be on, alarms = %v", f.hw.alarms)
	}
	v, err := f.svc.View(ctx, f.user.ID, req.ID)
	if err != nil || len(v.Wrong) != 1 || v.Wrong[0] != 2 {
		t.Fatalf("view wrong = %v, %v", v.Wrong, err)
	}

	problems, err := f.svc.Close(ctx, f.user.ID, req.ID, false)
	if err != nil || len(problems) != 1 || !strings.Contains(problems[0], "Slot 2") {
		t.Fatalf("confirming with the wrong gun out = %v, %v", problems, err)
	}
	if said := heardFrom(t, spoken); !strings.Contains(said, "Slot 2") {
		t.Fatalf("confirm said %q", said)
	}
	if got, _ := f.store.GetRequest(ctx, req.ID); got.ClosedAt != "" {
		t.Fatal("a mismatching collection must stay open")
	}

	f.sensor(t, 2, 1)
	if f.alarmOn() {
		t.Fatalf("the alarm should clear once the gun is back, alarms = %v", f.hw.alarms)
	}
	if problems, err := f.svc.Close(ctx, f.user.ID, req.ID, false); err != nil || len(problems) != 0 {
		t.Fatalf("confirming after putting it back = %v, %v", problems, err)
	}
}

func TestTakingYourOwnGunAgainAfterTheReturnIsFlaggedToo(t *testing.T) {
	f := newFlow(t)
	f.returnedRequest(t, 1)
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	f.sensor(t, 1, 0)
	if said := heardFrom(t, spoken); !strings.Contains(said, "slot 1") {
		t.Fatalf("said %q", said)
	}
}

func TestATakeWithNoRequestAtAllIsFlagged(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	f.sensor(t, 3, 0)
	if said := heardFrom(t, spoken); said != UnauthorizedTakeMessage(3) {
		t.Fatalf("said %q", said)
	}
	if !f.alarmOn() {
		t.Fatalf("alarm should be on, alarms = %v", f.hw.alarms)
	}
	if on, err := f.store.HasUnauthorizedEmpty(ctx, f.locker.ID); err != nil || !on {
		t.Fatalf("unauthorized empty = %v, %v", on, err)
	}

	f.sensor(t, 3, 1)
	if f.alarmOn() {
		t.Fatalf("alarm should clear after the gun is back, alarms = %v", f.hw.alarms)
	}
	if on, _ := f.store.HasUnauthorizedEmpty(ctx, f.locker.ID); on {
		t.Fatal("nothing is out of place any more")
	}
}

func TestAClosedRequestNoLongerExplainsATake(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req := f.returnedRequest(t, 1)
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()
	if problems, err := f.svc.Close(ctx, f.user.ID, req.ID, false); err != nil || len(problems) != 0 {
		t.Fatalf("close = %v, %v", problems, err)
	}
	heardFrom(t, spoken)

	f.sensor(t, 2, 0)
	if said := heardFrom(t, spoken); said != UnauthorizedTakeMessage(2) {
		t.Fatalf("said %q", said)
	}
}

func TestNothingIsFlaggedWhileSomeoneIsInMaintenance(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	job, err := f.store.AssignMaintenance(ctx, f.other.ID, f.admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.StartMaintenance(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	f.sensor(t, 1, 0)
	silent(t, spoken)
	if on, _ := f.store.HasUnauthorizedEmpty(ctx, f.locker.ID); on {
		t.Fatal("a take during maintenance must not be recorded")
	}
	if f.alarmOn() {
		t.Fatalf("no alarm during maintenance, alarms = %v", f.hw.alarms)
	}
}

func TestAnApprovedRequestStillGetsTheWrongPickWarning(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{1}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	f.sensor(t, 2, 0)
	if said := heardFrom(t, spoken); said != WrongPickMessage(2, []int64{1}) {
		t.Fatalf("said %q", said)
	}
	if status, _ := f.status(t, req.ID); status != models.RequestApproved {
		t.Fatalf("status = %s", status)
	}
}
