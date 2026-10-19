package httpserver_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"armory/internal/models"
	"armory/internal/services"
)

func TestMaintenanceFromRequestToFinishNeedsNoPlan(t *testing.T) {
	a := newApp(t, true)
	admin := a.browser(t)
	admin.adminSignIn()
	hx := map[string]string{"HX-Request": "true"}

	worker := a.browser(t)
	got := worker.signIn(a.user.ID)
	if _, has := got["maintenance_status"]; has {
		t.Fatalf("no job yet but sign-in reported one: %v", got)
	}

	status, _, body := worker.do("POST", "/api/maintenance/request", map[string]any{}, nil)
	job := asMap(t, body)
	if status != http.StatusCreated || job["status"] != "requested" || len(job["changes"].([]any)) != 0 {
		t.Fatalf("request = %d %s", status, body)
	}
	if status, _, _ := worker.do("POST", "/api/maintenance/request", map[string]any{}, nil); status != http.StatusConflict {
		t.Fatalf("second request = %d, want 409", status)
	}

	_, _, page := admin.do("GET", "/admin/requests", nil, nil)
	for _, want := range []string{"Maintenance requests", "Roshan Sahani", "Wants to do maintenance work"} {
		if !strings.Contains(page, want) {
			t.Fatalf("admin requests page is missing %q", want)
		}
	}
	if _, _, badge := admin.do("GET", "/admin/requests/badge", nil, nil); strings.TrimSpace(badge) != "1" {
		t.Fatalf("badge = %q, want 1", badge)
	}

	early := a.browser(t)
	if got := early.signIn(a.user.ID); got["maintenance_status"] != "requested" || a.hw.doorPulses() != 0 {
		t.Fatalf("before approval: %v, door pulses %d", got, a.hw.doorPulses())
	}
	if status, _, _ := early.do("POST", "/api/maintenance/begin", nil, nil); status != http.StatusConflict || a.hw.doorPulses() != 0 {
		t.Fatalf("begin before approval = %d, door pulses %d", status, a.hw.doorPulses())
	}

	jobID := int64(job["id"].(float64))
	if status, header, _ := admin.do("POST", idPath("/admin/maintenance/{id}/approve", jobID), nil, hx); status != http.StatusOK || header.Get("HX-Refresh") != "true" {
		t.Fatalf("approve = %d refresh %q", status, header.Get("HX-Refresh"))
	}

	spoken, cancel := a.hub.SubscribeSpeech()
	defer cancel()

	kiosk := a.browser(t)
	if got := kiosk.signIn(a.user.ID); got["maintenance_status"] != "assigned" || a.hw.doorPulses() != 0 {
		t.Fatalf("after approval, before choosing: %v, door pulses %d", got, a.hw.doorPulses())
	}
	if status, _, body := kiosk.do("POST", "/api/maintenance/begin", nil, nil); status != http.StatusOK || asMap(t, body)["say"] != services.MaintenanceStartMessage() {
		t.Fatalf("begin = %d %s", status, body)
	}
	if a.hw.doorPulses() != 1 {
		t.Fatalf("door pulses = %d, want 1", a.hw.doorPulses())
	}
	if got := a.hw.unlocked(); len(got) != 1 || got[0] != "10.0.0.5" {
		t.Fatalf("starting should unlock every locker once, got %v", got)
	}
	if status, _, _ := kiosk.do("POST", "/api/maintenance/unlock", map[string]any{"locker_id": a.locker.ID}, nil); status != http.StatusOK || len(a.hw.unlocked()) != 2 {
		t.Fatalf("unlock again = %d, pulses %v", status, a.hw.unlocked())
	}
	if status, _, _ := kiosk.do("POST", "/api/maintenance/unlock", map[string]any{"locker_id": 999}, nil); status != http.StatusBadRequest && status != http.StatusNotFound {
		t.Fatalf("unlock unknown locker = %d", status)
	}

	resumed := a.browser(t).signIn(a.user.ID)
	if resumed["maintenance_status"] != "active" || resumed["maintenance"] != true {
		t.Fatalf("signing in mid-job should resume it: %v", resumed)
	}

	other, err := a.store.CreateUser(t.Context(), models.User{Name: "Other Person", ServiceNo: "OP-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	a.faces[other.ID] = descriptor(2.2)
	if err := a.face.Enroll(t.Context(), other.ID, [][]float64{a.faces[other.ID]}); err != nil {
		t.Fatal(err)
	}
	stranger := a.browser(t)
	stranger.signIn(other.ID)
	gun := map[string]any{"locker_id": a.locker.ID, "slots": []int{1}, "reason": ""}
	status, _, body = stranger.do("POST", "/api/requests", gun, nil)
	if status != http.StatusConflict || !strings.Contains(body, "maintenance") {
		t.Fatalf("gun request during maintenance = %d %s", status, body)
	}
	if status, _, body = stranger.do("GET", "/api/catalog", nil, nil); status != http.StatusConflict || !strings.Contains(body, "maintenance") {
		t.Fatalf("catalog during maintenance = %d %s", status, body)
	}

	if status, _, body := kiosk.do("POST", "/api/maintenance/capacity", map[string]any{"locker_id": a.locker.ID, "capacity": 9}, nil); status != http.StatusBadRequest {
		t.Fatalf("capacity 9 = %d %s", status, body)
	}
	if status, _, body := kiosk.do("POST", "/api/maintenance/capacity", map[string]any{"locker_id": a.locker.ID, "capacity": 2}, nil); status != http.StatusOK {
		t.Fatalf("capacity = %d %s", status, body)
	}
	_, _, body = kiosk.do("GET", "/api/maintenance", nil, nil)
	if got := asMap(t, body)["lockers"].([]any)[0].(map[string]any)["capacity"]; got != float64(2) {
		t.Fatalf("capacity after edit = %v", got)
	}

	if status, _, body := kiosk.do("POST", "/api/maintenance/finish", nil, nil); status != http.StatusOK || asMap(t, body)["finished"] != true {
		t.Fatalf("finish = %d %s", status, body)
	}
	select {
	case said := <-spoken:
		if said != services.MaintenanceDoneMessage() {
			t.Fatalf("said %q", said)
		}
	case <-time.After(time.Second):
		t.Fatal("finishing spoke nothing")
	}
	if status, _, _ := kiosk.do("POST", "/api/maintenance/finish", nil, nil); status != http.StatusConflict {
		t.Fatalf("finishing twice = %d, want 409", status)
	}
	if status, _, _ := kiosk.do("POST", "/api/maintenance/capacity", map[string]any{"locker_id": a.locker.ID, "capacity": 3}, nil); status != http.StatusConflict {
		t.Fatalf("capacity after finishing = %d, want 409", status)
	}

	if status, _, body := stranger.do("POST", "/api/requests", gun, nil); status != http.StatusCreated {
		t.Fatalf("gun request after maintenance = %d %s", status, body)
	}
}

func TestSignedOutAndAdminsCannotUseMaintenance(t *testing.T) {
	a := newApp(t, true)
	anon := a.browser(t)
	if status, _, _ := anon.do("POST", "/api/maintenance/request", map[string]any{}, nil); status != http.StatusUnauthorized {
		t.Fatalf("signed out = %d, want 401", status)
	}
	b := a.browser(t)
	b.signIn(a.user.ID)
	if status, _, _ := b.do("POST", "/api/maintenance/capacity", map[string]any{"locker_id": a.locker.ID, "capacity": 2}, nil); status != http.StatusConflict {
		t.Fatalf("capacity before the job started = %d, want 409", status)
	}
	if status, _, _ := b.do("POST", "/api/maintenance/unlock", map[string]any{"locker_id": a.locker.ID}, nil); status != http.StatusConflict {
		t.Fatalf("unlock before the job started = %d, want 409", status)
	}
}
