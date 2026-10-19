package httpserver_test

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"armory/internal/services"
)

type recordingHardware struct {
	mu      sync.Mutex
	doors   int
	unlocks []string
}

func (h *recordingHardware) Unlock(ctx context.Context, lockerIP string, slotNo int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unlocks = append(h.unlocks, lockerIP)
	return nil
}

func (h *recordingHardware) unlocked() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.unlocks...)
}

func (h *recordingHardware) Signal(ctx context.Context, lockerIP string, signal services.Signal) error {
	return nil
}

func (h *recordingHardware) Alarm(ctx context.Context, lockerIP string, on bool) error {
	return nil
}

func (h *recordingHardware) OpenDoor(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.doors++
	return nil
}

func (h *recordingHardware) doorPulses() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.doors
}

func TestIdentifyNamesAKnownFaceWithoutSigningThemIn(t *testing.T) {
	a := newApp(t, true)
	b := a.browser(t)

	status, _, body := b.do("POST", "/face/identify", map[string]any{"descriptor": a.faces[a.user.ID]}, nil)
	got := asMap(t, body)
	if status != http.StatusOK || got["matched"] != true || got["name"] != "Roshan Sahani" {
		t.Fatalf("identify = %d %v", status, got)
	}
	if status, _, _ := b.do("GET", "/api/me", nil, nil); status != http.StatusUnauthorized {
		t.Fatalf("identify must not start a session, /api/me = %d", status)
	}

	_, _, body = b.do("POST", "/face/identify", map[string]any{"descriptor": descriptor(5)}, nil)
	if asMap(t, body)["matched"] != false {
		t.Fatalf("a stranger was identified: %s", body)
	}
	if status, _, _ := b.do("POST", "/face/identify", map[string]any{"descriptor": []float64{1, 2}}, nil); status != http.StatusBadRequest {
		t.Fatalf("bad descriptor = %d, want 400", status)
	}
}

func idPath(format string, id int64) string {
	return strings.Replace(format, "{id}", strconv.FormatInt(id, 10), 1)
}

func TestMaintenanceFlowFromAssignmentToFinish(t *testing.T) {
	a := newApp(t, true)
	admin := a.browser(t)
	admin.adminSignIn()
	assign := idPath("/admin/users/{id}/maintenance", a.user.ID)
	hx := map[string]string{"HX-Request": "true"}

	if status, header, body := admin.do("POST", assign, nil, hx); status != http.StatusOK || header.Get("HX-Refresh") != "true" {
		t.Fatalf("assign = %d, refresh %q, %s", status, header.Get("HX-Refresh"), body)
	}

	spoken, cancel := a.hub.SubscribeSpeech()
	defer cancel()

	kiosk := a.browser(t)
	got := kiosk.signIn(a.user.ID)
	if got["maintenance_status"] != "assigned" || got["maintenance"] != false {
		t.Fatalf("sign-in should only report the assigned job: %v", got)
	}
	if a.hw.doorPulses() != 0 {
		t.Fatalf("signing in opened the door before Maintenance was chosen: %d", a.hw.doorPulses())
	}
	status, _, body := kiosk.do("POST", "/api/maintenance/begin", nil, nil)
	if status != http.StatusOK || asMap(t, body)["say"] != services.MaintenanceStartMessage() {
		t.Fatalf("begin = %d %s", status, body)
	}
	if a.hw.doorPulses() != 1 {
		t.Fatalf("door pulses = %d, want 1", a.hw.doorPulses())
	}

	if status, _, body := kiosk.do("POST", "/api/maintenance/finish", nil, nil); status != http.StatusOK || asMap(t, body)["finished"] != true {
		t.Fatalf("finish = %d %s", status, body)
	}
	select {
	case text := <-spoken:
		if text != services.MaintenanceDoneMessage() {
			t.Fatalf("finish said %q", text)
		}
	case <-time.After(time.Second):
		t.Fatal("finishing spoke nothing")
	}
	if status, _, _ := kiosk.do("POST", "/api/maintenance/finish", nil, nil); status != http.StatusConflict {
		t.Fatalf("finishing twice = %d, want 409", status)
	}

	again := a.browser(t).signIn(a.user.ID)
	if _, has := again["maintenance_status"]; has {
		t.Fatalf("a finished job must not show again: %v", again)
	}
	if a.hw.doorPulses() != 1 {
		t.Fatalf("door opened again after the job finished: %d", a.hw.doorPulses())
	}
}

func TestMaintenanceCanBeCancelledBeforeTheyArrive(t *testing.T) {
	a := newApp(t, true)
	admin := a.browser(t)
	admin.adminSignIn()
	hx := map[string]string{"HX-Request": "true"}
	admin.do("POST", idPath("/admin/users/{id}/maintenance", a.user.ID), nil, hx)

	job, err := a.store.OpenMaintenanceFor(context.Background(), a.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status, header, _ := admin.do("POST", idPath("/admin/maintenance/{id}/cancel", job.ID), nil, hx); status != http.StatusOK || header.Get("HX-Refresh") != "true" {
		t.Fatalf("cancel = %d, refresh %q", status, header.Get("HX-Refresh"))
	}

	got := a.browser(t).signIn(a.user.ID)
	if _, has := got["maintenance_status"]; has || a.hw.doorPulses() != 0 {
		t.Fatalf("a cancelled job is still showing or opened the door: %v, pulses %d", got, a.hw.doorPulses())
	}
	b := a.browser(t)
	b.signIn(a.user.ID)
	if status, _, _ := b.do("POST", "/api/maintenance/begin", nil, nil); status != http.StatusConflict || a.hw.doorPulses() != 0 {
		t.Fatalf("begin on a cancelled job = %d, pulses %d", status, a.hw.doorPulses())
	}
}

func TestMaintenanceAssignmentRules(t *testing.T) {
	a := newApp(t, true)
	path := idPath("/admin/users/{id}/maintenance", a.user.ID)
	hx := map[string]string{"HX-Request": "true"}

	t.Run("only a signed-in admin can assign", func(t *testing.T) {
		if status, _, _ := a.browser(t).do("POST", path, nil, nil); status != http.StatusUnauthorized {
			t.Fatalf("anonymous = %d, want 401", status)
		}
		kiosk := a.browser(t)
		kiosk.signIn(a.user.ID)
		if status, _, _ := kiosk.do("POST", path, nil, nil); status == http.StatusOK {
			t.Fatalf("a requester assigned maintenance, status %d", status)
		}
		if _, err := a.store.OpenMaintenanceFor(context.Background(), a.user.ID); err == nil {
			t.Fatal("a job was created without an admin")
		}
	})

	t.Run("a second job for the same person is refused with a message", func(t *testing.T) {
		admin := a.browser(t)
		admin.adminSignIn()
		admin.do("POST", path, nil, hx)
		status, header, body := admin.do("POST", path, nil, hx)
		wantTarget := "#person-error-" + strconv.FormatInt(a.user.ID, 10)
		if status != http.StatusOK || header.Get("HX-Retarget") != wantTarget || !strings.Contains(body, "already has a maintenance job") {
			t.Fatalf("second assign = %d, retarget %q, body %q", status, header.Get("HX-Retarget"), body)
		}
	})

	t.Run("an admin cannot be assigned", func(t *testing.T) {
		admin := a.browser(t)
		admin.adminSignIn()
		status, _, body := admin.do("POST", idPath("/admin/users/{id}/maintenance", a.admin.ID), nil, hx)
		if status != http.StatusOK || !strings.Contains(body, "only a requester") {
			t.Fatalf("assign admin = %d %q", status, body)
		}
	})

	t.Run("an unknown person is a 404", func(t *testing.T) {
		admin := a.browser(t)
		admin.adminSignIn()
		if status, _, _ := admin.do("POST", "/admin/users/9999/maintenance", nil, hx); status != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", status)
		}
	})
}

func TestPeoplePageAndActivityShowMaintenance(t *testing.T) {
	a := newApp(t, true)
	admin := a.browser(t)
	admin.adminSignIn()

	_, _, before := admin.do("GET", "/admin/users", nil, nil)
	if !strings.Contains(before, "Assign maintenance") || strings.Contains(before, "Cancel maintenance") {
		t.Fatal("people page should offer to assign, not cancel, before any job")
	}

	admin.do("POST", idPath("/admin/users/{id}/maintenance", a.user.ID), url.Values{}, map[string]string{"HX-Request": "true"})
	_, _, assigned := admin.do("GET", "/admin/users", nil, nil)
	if !strings.Contains(assigned, "Maintenance assigned") || !strings.Contains(assigned, "Cancel maintenance") {
		t.Fatal("people page should show the assigned job with a cancel button")
	}

	arrived := a.browser(t)
	arrived.signIn(a.user.ID)
	arrived.do("POST", "/api/maintenance/begin", nil, nil)
	_, _, active := admin.do("GET", "/admin/users", nil, nil)
	if !strings.Contains(active, "On maintenance") {
		t.Fatal("people page should show the person is on maintenance after they arrive")
	}

	_, _, log := admin.do("GET", "/admin/activity", nil, nil)
	for _, want := range []string{"Maintenance assigned", "Maintenance started", "Roshan Sahani"} {
		if !strings.Contains(log, want) {
			t.Fatalf("activity page is missing %q", want)
		}
	}
}
