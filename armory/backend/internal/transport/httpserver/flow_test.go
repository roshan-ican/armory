package httpserver_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/models"
	"armory/internal/services"
	"armory/internal/transport/httpserver"
)

type app struct {
	handler  http.Handler
	ts       *httptest.Server
	store    *database.Store
	requests *services.RequestService
	face     *services.FaceService
	locker   models.Locker
	admin    models.User
	user     models.User
	faces    map[int64][]float64
}

func descriptor(seed float64) []float64 {
	d := make([]float64, 128)
	for i := range d {
		d[i] = seed * float64(i%7+1) / 10
	}
	return d
}

func newApp(t *testing.T, enrollAdmin bool) *app {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)
	hub := live.NewHub()
	lockers := services.NewLockerService(store)
	users := services.NewUserService(store)
	face := services.NewFaceService(store)
	requests := services.NewRequestService(store, services.LogHardware{}, hub)
	srv, err := httpserver.New(lockers, services.NewActivityService(store), users, face, requests, services.NewSessions(), hub)
	if err != nil {
		t.Fatal(err)
	}
	handler := srv.Routes()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	var admin models.User
	if enrollAdmin {
		admin, err = users.SetupAdmin(ctx, "Admin One", "admin", "admin-password")
	} else {
		admin, err = users.Create(ctx, "Admin One", "AD-1", "admin")
	}
	if err != nil {
		t.Fatal(err)
	}
	user, err := users.Create(ctx, "Roshan Sahani", "SN-1", "requester")
	if err != nil {
		t.Fatal(err)
	}
	locker, err := store.CreateLocker(ctx, models.Locker{Name: "East", IPAddress: "10.0.0.5", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, sl := range locker.Slots {
		if err := store.RecordSensorEvent(ctx, sl.ID, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
	}
	faces := map[int64][]float64{user.ID: descriptor(-0.6)}
	if err := face.Enroll(ctx, user.ID, [][]float64{faces[user.ID]}); err != nil {
		t.Fatal(err)
	}
	return &app{handler: handler, ts: ts, store: store, requests: requests, face: face, locker: locker, admin: admin, user: user, faces: faces}
}

type browser struct {
	t *testing.T
	a *app
	c *http.Client
}

func (a *app) browser(t *testing.T) *browser {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return &browser{t: t, a: a, c: c}
}

func (b *browser) do(method, path string, body any, headers map[string]string) (int, http.Header, string) {
	b.t.Helper()
	var reader io.Reader
	contentType := ""
	switch v := body.(type) {
	case nil:
	case url.Values:
		reader = strings.NewReader(v.Encode())
		contentType = "application/x-www-form-urlencoded"
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			b.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
		contentType = "application/json"
	}
	req, err := http.NewRequest(method, b.a.ts.URL+path, reader)
	if err != nil {
		b.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := b.c.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header, string(raw)
}

func (b *browser) signIn(userID int64) map[string]any {
	b.t.Helper()
	status, _, body := b.do("POST", "/face/match", map[string]any{"descriptor": b.a.faces[userID]}, nil)
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil || status != 200 {
		b.t.Fatalf("sign in: %d %s", status, body)
	}
	return out
}

func (b *browser) adminSignIn() {
	b.t.Helper()
	status, header, body := b.do("POST", "/admin/login", url.Values{"username": {"admin"}, "password": {"admin-password"}, "next": {"/admin/lockers"}}, map[string]string{"HX-Request": "true"})
	if status != 200 || header.Get("HX-Redirect") != "/admin/lockers" {
		b.t.Fatalf("admin sign in: %d %q %s", status, header.Get("HX-Redirect"), body)
	}
}

func asMap(t *testing.T, body string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("not json: %q", body)
	}
	return out
}

func TestAdminPagesLockOnceAnAdminCanSignIn(t *testing.T) {
	t.Run("with no password admin, admin pages stay closed and login shows setup", func(t *testing.T) {
		a := newApp(t, false)
		anon := a.browser(t)
		if status, _, _ := anon.do("GET", "/admin/lockers", nil, nil); status != http.StatusSeeOther {
			t.Fatalf("lockers = %d, want 303", status)
		}
		status, _, body := anon.do("GET", "/admin/login", nil, nil)
		if status != 200 || !strings.Contains(body, "Create the first admin") {
			t.Fatalf("login = %d, setup missing", status)
		}
	})

	t.Run("once an admin can sign in, anonymous visitors are sent to the login page", func(t *testing.T) {
		a := newApp(t, true)
		anon := a.browser(t)
		for _, path := range []string{"/admin/lockers", "/admin/requests", "/admin/activity", "/admin/users"} {
			status, header, _ := anon.do("GET", path, nil, nil)
			if status != http.StatusSeeOther || !strings.HasPrefix(header.Get("Location"), "/admin/login") {
				t.Fatalf("%s = %d %q", path, status, header.Get("Location"))
			}
		}
		if status, _, _ := anon.do("POST", "/admin/lockers", url.Values{"name": {"X"}, "kind": {"rifle"}, "capacity": {"1"}}, nil); status != http.StatusUnauthorized {
			t.Fatalf("POST /lockers = %d, want 401", status)
		}
		if status, header, _ := anon.do("GET", "/admin/requests", nil, map[string]string{"HX-Request": "true"}); status != http.StatusUnauthorized || header.Get("HX-Redirect") != "/admin/login" {
			t.Fatalf("htmx request = %d %q", status, header.Get("HX-Redirect"))
		}
		for _, path := range []string{"/admin/login", "/admin.webmanifest", "/sw.js"} {
			if status, _, _ := anon.do("GET", path, nil, nil); status != 200 {
				t.Fatalf("%s = %d, want 200", path, status)
			}
		}
		for _, path := range []string{"/kiosk", "/enroll", "/kiosk.webmanifest"} {
			if status, _, _ := anon.do("GET", path, nil, nil); status != http.StatusNotFound && status != http.StatusMethodNotAllowed {
				t.Fatalf("%s = %d, want 404 or 405", path, status)
			}
		}
		if status, _, _ := anon.do("GET", "/api/me", nil, nil); status != http.StatusUnauthorized {
			t.Fatalf("/api/me = %d, want 401", status)
		}
	})

	t.Run("the bare address opens the admin area", func(t *testing.T) {
		a := newApp(t, true)
		status, header, _ := a.browser(t).do("GET", "/", nil, nil)
		if status != http.StatusSeeOther || header.Get("Location") != "/admin" {
			t.Fatalf("/ = %d %q, want a redirect to /admin", status, header.Get("Location"))
		}
	})

	t.Run("an admin signs in and reaches the pages", func(t *testing.T) {
		a := newApp(t, true)
		admin := a.browser(t)
		admin.adminSignIn()
		for _, path := range []string{"/admin/lockers", "/admin/requests", "/admin/activity", "/admin/users"} {
			if status, _, _ := admin.do("GET", path, nil, nil); status != 200 {
				t.Fatalf("%s = %d, want 200", path, status)
			}
		}
	})

	t.Run("a requester never reaches the admin pages", func(t *testing.T) {
		a := newApp(t, true)
		user := a.browser(t)
		user.signIn(a.user.ID)
		for _, path := range []string{"/admin/lockers", "/admin/requests", "/admin/users"} {
			if status, _, _ := user.do("GET", path, nil, nil); status != http.StatusSeeOther {
				t.Fatalf("%s = %d, want 303", path, status)
			}
		}
		if status, _, _ := user.do("POST", "/admin/lockers", url.Values{"name": {"X"}, "kind": {"rifle"}, "capacity": {"1"}}, nil); status != http.StatusUnauthorized {
			t.Fatalf("POST /lockers = %d, want 401", status)
		}
	})

	t.Run("a stranger gets no session", func(t *testing.T) {
		a := newApp(t, true)
		stranger := a.browser(t)
		status, _, body := stranger.do("POST", "/face/match", map[string]any{"descriptor": descriptor(9)}, nil)
		if status != 200 || asMap(t, body)["matched"] != false {
			t.Fatalf("got %d %s", status, body)
		}
		if status, _, _ := stranger.do("GET", "/api/me", nil, nil); status != http.StatusUnauthorized {
			t.Fatalf("/api/me = %d, want 401", status)
		}
	})

	t.Run("logging out ends the session", func(t *testing.T) {
		a := newApp(t, true)
		user := a.browser(t)
		user.signIn(a.user.ID)
		if status, _, _ := user.do("GET", "/api/me", nil, nil); status != 200 {
			t.Fatalf("/api/me = %d", status)
		}
		user.do("POST", "/api/logout", nil, nil)
		if status, _, _ := user.do("GET", "/api/me", nil, nil); status != http.StatusUnauthorized {
			t.Fatalf("/api/me after logout = %d, want 401", status)
		}
	})
}

func TestRequestLifecycleOverHTTP(t *testing.T) {
	ctx := context.Background()
	a := newApp(t, true)
	user := a.browser(t)
	admin := a.browser(t)
	user.signIn(a.user.ID)
	admin.adminSignIn()

	status, _, body := user.do("GET", "/api/availability", nil, nil)
	got := asMap(t, body)
	if status != 200 || got["rifle"] != float64(3) || got["pistol"] != float64(0) {
		t.Fatalf("availability = %d %s", status, body)
	}

	status, _, body = user.do("POST", "/api/requests", map[string]any{"kind": "rifle", "reason": "Range"}, nil)
	created := asMap(t, body)
	if status != http.StatusCreated || created["status"] != "pending" {
		t.Fatalf("create = %d %s", status, body)
	}
	id := int64(created["id"].(float64))
	idPath := "/api/requests/" + itoa(id)

	if status, _, _ := user.do("POST", "/api/requests", map[string]any{"kind": "rifle"}, nil); status != http.StatusConflict {
		t.Fatalf("second open request = %d, want 409", status)
	}
	if status, _, _ := user.do("POST", "/admin/requests/"+itoa(id)+"/approve", nil, map[string]string{"HX-Request": "true"}); status != http.StatusUnauthorized {
		t.Fatalf("requester approving = %d, want 401", status)
	}
	if _, _, body := admin.do("GET", "/admin/requests", nil, nil); !strings.Contains(body, "Roshan Sahani") {
		t.Fatal("the waiting request is not on the admin page")
	}
	if _, _, body := admin.do("GET", "/admin/requests/badge", nil, nil); strings.TrimSpace(body) != "1" {
		t.Fatalf("badge = %q, want 1", body)
	}

	status, _, body = admin.do("POST", "/admin/requests/"+itoa(id)+"/approve", nil, map[string]string{"HX-Request": "true"})
	if status != 200 || body != "" {
		t.Fatalf("approve = %d %q", status, body)
	}
	_, _, body = admin.do("POST", "/admin/requests/"+itoa(id)+"/approve", nil, map[string]string{"HX-Request": "true"})
	if !strings.Contains(body, "already handled") {
		t.Fatalf("second approve = %q", body)
	}
	if _, _, body := admin.do("GET", "/admin/requests/badge", nil, nil); strings.TrimSpace(body) != "" {
		t.Fatalf("badge = %q, want empty", body)
	}

	_, _, body = user.do("GET", idPath, nil, nil)
	view := asMap(t, body)
	if view["status"] != "approved" || view["slot_no"] != float64(1) || len(view["slots"].([]any)) != 3 {
		t.Fatalf("approved view = %s", body)
	}

	wrong := a.locker.Slots[2].ID
	must(t, a.store.RecordSensorEvent(ctx, wrong, 0, "gun_removed"))
	a.requests.SlotChanged(ctx, wrong, 0)
	_, _, body = user.do("GET", idPath, nil, nil)
	if w := asMap(t, body)["wrong"].([]any); len(w) != 1 || w[0] != float64(3) {
		t.Fatalf("wrong = %s", body)
	}
	must(t, a.store.RecordSensorEvent(ctx, wrong, 1, "gun_returned"))
	a.requests.SlotChanged(ctx, wrong, 1)
	_, _, body = user.do("GET", idPath, nil, nil)
	if w := asMap(t, body)["wrong"].([]any); len(w) != 0 {
		t.Fatalf("wrong after putting it back = %s", body)
	}

	right := a.locker.Slots[0].ID
	must(t, a.store.RecordSensorEvent(ctx, right, 0, "gun_removed"))
	a.requests.SlotChanged(ctx, right, 0)
	_, _, body = user.do("GET", idPath, nil, nil)
	if asMap(t, body)["status"] != "collected" {
		t.Fatalf("collected view = %s", body)
	}
	if status, _, _ := user.do("GET", "/api/requests/current", nil, nil); status != 200 {
		t.Fatalf("a collected request is still open, got %d", status)
	}

	must(t, a.store.RecordSensorEvent(ctx, right, 1, "gun_returned"))
	a.requests.SlotChanged(ctx, right, 1)
	_, _, body = user.do("GET", idPath, nil, nil)
	if asMap(t, body)["status"] != "returned" {
		t.Fatalf("returned view = %s", body)
	}
	if status, _, _ := user.do("GET", "/api/requests/current", nil, nil); status != http.StatusNotFound {
		t.Fatalf("a returned request is closed, got %d", status)
	}

	status, _, body = user.do("POST", "/api/requests", map[string]any{"kind": "rifle"}, nil)
	second := asMap(t, body)
	if status != http.StatusCreated {
		t.Fatalf("second request = %d %s", status, body)
	}
	secondID := itoa(int64(second["id"].(float64)))
	if status, _, _ := admin.do("POST", "/admin/requests/"+secondID+"/reject", nil, map[string]string{"HX-Request": "true"}); status != 200 {
		t.Fatalf("reject = %d", status)
	}
	_, _, body = user.do("GET", "/api/requests/"+secondID, nil, nil)
	if asMap(t, body)["status"] != "rejected" {
		t.Fatalf("rejected view = %s", body)
	}

	other := a.browser(t)
	other.adminSignIn()
	if status, _, _ := other.do("GET", idPath, nil, nil); status != http.StatusNotFound {
		t.Fatalf("someone else's request = %d, want 404", status)
	}
}

func TestTabletEnrollmentRequiresAdminApproval(t *testing.T) {
	a := newApp(t, true)
	tablet := a.browser(t)
	candidate := descriptor(0.9)
	status, _, body := tablet.do("POST", "/enroll", map[string]any{
		"name": "New Person", "service_no": "NP-1", "descriptors": [][]float64{candidate, candidate},
	}, nil)
	if status != 200 || asMap(t, body)["pending"] != true {
		t.Fatalf("enroll = %d %s", status, body)
	}
	if status, _, body = tablet.do("POST", "/face/match", map[string]any{"descriptor": candidate}, nil); status != 200 || asMap(t, body)["matched"] != false {
		t.Fatalf("face worked before approval: %d %s", status, body)
	}

	admin := a.browser(t)
	admin.adminSignIn()
	_, _, body = admin.do("GET", "/admin/users", nil, nil)
	if !strings.Contains(body, "New Person") || strings.Contains(body, "/admin/users/1/enroll") {
		t.Fatalf("pending enrollment missing or admin enroll link present: %s", body)
	}
	if status, _, body = admin.do("POST", "/admin/enrollments/1/approve", nil, map[string]string{"HX-Request": "true"}); status != 200 {
		t.Fatalf("approve = %d %s", status, body)
	}
	if status, _, body = tablet.do("POST", "/face/match", map[string]any{"descriptor": candidate}, nil); status != 200 || asMap(t, body)["name"] != "New Person" {
		t.Fatalf("approved face did not work: %d %s", status, body)
	}
}

func TestAdminFaceCanNeverCreateAnAdminSession(t *testing.T) {
	a := newApp(t, true)
	adminFace := descriptor(0.25)
	if err := a.face.Enroll(context.Background(), a.admin.ID, [][]float64{adminFace}); err != nil {
		t.Fatal(err)
	}
	b := a.browser(t)
	status, _, body := b.do("POST", "/face/match", map[string]any{"descriptor": adminFace}, nil)
	if status != 200 || asMap(t, body)["matched"] != false {
		t.Fatalf("admin face = %d %s", status, body)
	}
	if status, _, _ := b.do("GET", "/admin/lockers", nil, nil); status != http.StatusSeeOther {
		t.Fatalf("admin page = %d, want redirect", status)
	}
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSessionCookieIsSecureOnlyOverHTTPS(t *testing.T) {
	a := newApp(t, true)
	body, err := json.Marshal(map[string]any{"descriptor": a.faces[a.user.ID]})
	if err != nil {
		t.Fatal(err)
	}
	cookieFor := func(secure bool) *http.Cookie {
		req := httptest.NewRequest("POST", "/face/match", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if secure {
			req.TLS = &tls.ConnectionState{}
		}
		rec := httptest.NewRecorder()
		a.handler.ServeHTTP(rec, req)
		for _, c := range rec.Result().Cookies() {
			if c.Name == "armory_session" {
				return c
			}
		}
		t.Fatal("no session cookie was set")
		return nil
	}
	if c := cookieFor(true); !c.Secure || !c.HttpOnly {
		t.Fatalf("over https: secure=%v httponly=%v", c.Secure, c.HttpOnly)
	}
	if c := cookieFor(false); c.Secure || !c.HttpOnly {
		t.Fatalf("over http: secure=%v httponly=%v", c.Secure, c.HttpOnly)
	}
}

func TestPlainHTTPFromOtherDevicesGoesToHTTPS(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	handler := httpserver.RedirectHTTPS(inner, ":8443")

	get := func(host, target string, secure bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", target, nil)
		req.Host = host
		if secure {
			req.TLS = &tls.ConnectionState{}
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := get("10.252.176.161:8080", "/kiosk?x=1", false)
	if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "https://10.252.176.161:8443/kiosk?x=1" {
		t.Fatalf("got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	for _, tc := range []struct{ host, path string }{
		{"localhost:8080", "/kiosk"},
		{"127.0.0.1:8080", "/admin/lockers"},
		{"[::1]:8080", "/kiosk"},
		{"10.252.176.161:8080", "/ca.crt"},
		{"10.252.176.161:8080", "/health"},
	} {
		if rec := get(tc.host, tc.path, false); rec.Code != 200 {
			t.Fatalf("%s%s = %d, want it served directly", tc.host, tc.path, rec.Code)
		}
	}
	if rec := get("10.252.176.161:8443", "/kiosk", true); rec.Code != 200 {
		t.Fatalf("https request = %d, want 200", rec.Code)
	}
	if got := httpserver.RedirectHTTPS(inner, ""); got == nil {
		t.Fatal("no TLS address must return the plain handler")
	}
}

func TestActivityPageFiltersAndPages(t *testing.T) {
	a := newApp(t, true)
	b := a.browser(t)
	b.adminSignIn()

	status, _, body := b.do("GET", "/admin/activity", nil, nil)
	if status != 200 || !strings.Contains(body, "Person") || !strings.Contains(body, "Everyone") {
		t.Fatalf("activity page: %d %s", status, body)
	}
	status, _, body = b.do("GET", "/admin/activity?page=99&person=Nobody&type=gun_removed&from=2026-01-01&to=bad", nil, nil)
	if status != 200 || !strings.Contains(body, "Nothing matches these filters.") {
		t.Fatalf("filtered activity page: %d %s", status, body)
	}
}

func TestAdminSetsSensorLayoutFromTheEditForm(t *testing.T) {
	a := newApp(t, true)
	b := a.browser(t)
	b.adminSignIn()

	path := "/admin/lockers/" + strconv.FormatInt(a.locker.ID, 10)
	form := func(start, reverse string) url.Values {
		v := url.Values{"name": {a.locker.Name}, "kind": {a.locker.Kind}, "capacity": {strconv.FormatInt(a.locker.Capacity, 10)}, "sensor_start": {start}}
		if reverse != "" {
			v.Set("sensor_reverse", reverse)
		}
		return v
	}
	hx := map[string]string{"HX-Request": "true"}

	if status, _, body := b.do("POST", path, form("2", "1"), hx); status != 200 {
		t.Fatalf("save: %d %s", status, body)
	}
	l, err := a.store.GetLocker(context.Background(), a.locker.ID)
	if err != nil {
		t.Fatal(err)
	}
	if l.SensorStart != 2 || !l.SensorReverse {
		t.Fatalf("saved layout = %d, %v, want 2, true", l.SensorStart, l.SensorReverse)
	}

	_, header, _ := b.do("POST", path, form("99", ""), hx)
	if header.Get("HX-Retarget") == "" {
		t.Fatal("an out-of-range first sensor slot must show an error")
	}
	l, _ = a.store.GetLocker(context.Background(), a.locker.ID)
	if l.SensorStart != 2 {
		t.Fatalf("invalid value must not be saved, got %d", l.SensorStart)
	}
}
