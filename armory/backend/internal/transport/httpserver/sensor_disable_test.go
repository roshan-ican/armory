package httpserver_test

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestAdminDisablesASensorFromTheEditCard(t *testing.T) {
	a := newApp(t, true)
	b := a.browser(t)
	b.adminSignIn()

	id := strconv.FormatInt(a.locker.ID, 10)
	hx := map[string]string{"HX-Request": "true"}

	status, _, body := b.do("GET", "/admin/lockers/"+id+"/edit", nil, hx)
	if status != 200 || !strings.Contains(body, "/slots/1/sensor") || !strings.Contains(body, "tap to disable") {
		t.Fatalf("edit card: %d %s", status, body)
	}

	status, _, body = b.do("POST", "/admin/lockers/"+id+"/slots/1/sensor", url.Values{"disabled": {"1"}}, hx)
	if status != 200 || !strings.Contains(body, "Disabled by admin, tap to enable") {
		t.Fatalf("disable: %d %s", status, body)
	}
	l, err := a.store.GetLocker(context.Background(), a.locker.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !l.Slots[0].SensorDisabled || l.Slots[1].SensorDisabled {
		t.Fatalf("slots = %+v", l.Slots)
	}

	_, body = func() (int, string) { s, _, bd := b.do("GET", "/admin/lockers/"+id, nil, hx); return s, bd }()
	if !strings.Contains(body, "slot-disabled") || !strings.Contains(body, "Disabled by admin") {
		t.Fatalf("card must show the disabled slot: %s", body)
	}

	_, header, _ := b.do("POST", "/admin/lockers/"+id+"/slots/9/sensor", url.Values{"disabled": {"1"}}, hx)
	if header.Get("HX-Retarget") == "" {
		t.Fatal("a slot outside the locker must show an error")
	}

	status, _, _ = b.do("POST", "/admin/lockers/"+id+"/slots/1/sensor", url.Values{"disabled": {"0"}}, hx)
	l, _ = a.store.GetLocker(context.Background(), a.locker.ID)
	if status != 200 || l.Slots[0].SensorDisabled {
		t.Fatalf("enable: %d, slots = %+v", status, l.Slots)
	}
}
