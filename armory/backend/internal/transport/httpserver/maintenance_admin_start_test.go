package httpserver_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"armory/internal/services"
)

func TestAdminStartsAndFinishesMaintenanceWithoutTablet(t *testing.T) {
	a := newApp(t, true)
	admin := a.browser(t)
	admin.adminSignIn()
	hx := map[string]string{"HX-Request": "true"}

	if status, _, body := admin.do("GET", "/admin/requests", nil, nil); status != http.StatusOK ||
		!strings.Contains(body, "Start maintenance for someone") || !strings.Contains(body, a.user.Name) {
		t.Fatalf("requests page = %d, form or person missing", status)
	}

	status, header, body := admin.do("POST", "/admin/maintenance/start", url.Values{}, hx)
	if status != http.StatusOK || header.Get("HX-Retarget") != "#maintenance-start-error" {
		t.Fatalf("start without a person = %d, retarget %q, %s", status, header.Get("HX-Retarget"), body)
	}

	form := url.Values{"user_id": {itoa(a.user.ID)}}
	status, header, body = admin.do("POST", "/admin/maintenance/start", form, hx)
	if status != http.StatusOK || header.Get("HX-Refresh") != "true" {
		t.Fatalf("start = %d, refresh %q, %s", status, header.Get("HX-Refresh"), body)
	}
	if a.hw.doorPulses() != 1 || len(a.hw.unlocked()) != 0 {
		t.Fatalf("door pulses %d, unlocks %v; want 1 door pulse and no locker unlocks", a.hw.doorPulses(), a.hw.unlocked())
	}

	job, err := a.store.OpenMaintenanceFor(context.Background(), a.user.ID)
	if err != nil || job.Status != "active" {
		t.Fatalf("job = %+v, %v", job, err)
	}

	_, _, body = admin.do("GET", "/admin/requests", nil, nil)
	if !strings.Contains(body, "Maintenance in progress") || !strings.Contains(body, "/admin/maintenance/"+itoa(job.ID)+"/finish") {
		t.Fatalf("in-progress section missing:\n%s", body)
	}

	status, header, body = admin.do("POST", "/admin/maintenance/start", form, hx)
	if header.Get("HX-Retarget") != "#maintenance-start-error" || !strings.Contains(body, services.ErrMaintenanceOpen.Error()) {
		t.Fatalf("second start = %d, %s", status, body)
	}

	spoken, cancel := a.hub.SubscribeSpeech()
	defer cancel()
	status, header, _ = admin.do("POST", idPath("/admin/maintenance/{id}/finish", job.ID), nil, hx)
	if status != http.StatusOK || header.Get("HX-Refresh") != "true" {
		t.Fatalf("finish = %d, refresh %q", status, header.Get("HX-Refresh"))
	}
	if text := <-spoken; text != services.MaintenanceDoneMessage() {
		t.Fatalf("finish said %q", text)
	}
	if _, err := a.store.OpenMaintenanceFor(context.Background(), a.user.ID); err == nil {
		t.Fatal("job still open after finish")
	}
	if status, _, _ := admin.do("POST", idPath("/admin/maintenance/{id}/finish", job.ID), nil, hx); status != http.StatusOK {
		t.Fatalf("finishing twice = %d", status)
	}

	_, _, body = admin.do("GET", "/admin/activity", nil, nil)
	if !strings.Contains(body, "Maintenance started") || !strings.Contains(body, "started by an admin, no tablet") ||
		!strings.Contains(body, "Maintenance finished") {
		t.Fatalf("activity missing the maintenance events")
	}
}

func TestAnonymousCannotStartMaintenance(t *testing.T) {
	a := newApp(t, true)
	anon := a.browser(t)
	form := url.Values{"user_id": {itoa(a.user.ID)}}
	if status, _, _ := anon.do("POST", "/admin/maintenance/start", form, nil); status == http.StatusOK {
		t.Fatalf("anonymous start = %d", status)
	}
	if a.hw.doorPulses() != 0 {
		t.Fatal("door opened for an anonymous caller")
	}
}
