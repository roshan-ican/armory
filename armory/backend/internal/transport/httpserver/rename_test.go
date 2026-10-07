package httpserver_test

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestFaceSignInCarriesTheServersGreeting(t *testing.T) {
	a := newApp(t, true)
	got := a.browser(t).signIn(a.user.ID)
	greeting, _ := got["greeting"].(string)
	switch greeting {
	case "Good morning", "Good afternoon", "Good evening":
	default:
		t.Fatalf("greeting = %q in %v", greeting, got)
	}
}

func TestSignedInPersonSeesTheirNewNameWithoutSigningInAgain(t *testing.T) {
	a := newApp(t, true)
	kiosk := a.browser(t)
	kiosk.signIn(a.user.ID)
	meName := func() string {
		status, _, body := kiosk.do("GET", "/api/me", nil, nil)
		if status != http.StatusOK {
			t.Fatalf("/api/me = %d %s", status, body)
		}
		return asMap(t, body)["name"].(string)
	}
	if got := meName(); got != "Roshan Sahani" {
		t.Fatalf("before = %q", got)
	}

	admin := a.browser(t)
	admin.adminSignIn()
	path := "/admin/users/" + strconv.FormatInt(a.user.ID, 10) + "/rename"
	if status, _, _ := admin.do("POST", path, url.Values{"name": {"Rohan Verma"}}, nil); status != http.StatusOK {
		t.Fatalf("rename = %d", status)
	}

	if got := meName(); got != "Rohan Verma" {
		t.Fatalf("after = %q, want the new name on the same session", got)
	}
}

func TestAdminRenamesAPerson(t *testing.T) {
	a := newApp(t, true)
	path := "/admin/users/" + strconv.FormatInt(a.user.ID, 10) + "/rename"
	nameOf := func() string {
		u, err := a.store.GetUser(context.Background(), a.user.ID)
		if err != nil {
			t.Fatal(err)
		}
		return u.Name
	}

	t.Run("someone who is not signed in cannot rename", func(t *testing.T) {
		anon := a.browser(t)
		if status, _, _ := anon.do("POST", path, url.Values{"name": {"Hacker"}}, nil); status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", status)
		}
		if got := nameOf(); got != "Roshan Sahani" {
			t.Fatalf("name changed to %q", got)
		}
	})

	t.Run("a signed-in requester cannot rename", func(t *testing.T) {
		kiosk := a.browser(t)
		kiosk.signIn(a.user.ID)
		if status, _, _ := kiosk.do("POST", path, url.Values{"name": {"Hacker"}}, nil); status == http.StatusOK {
			t.Fatalf("a requester renamed a person, status %d", status)
		}
		if got := nameOf(); got != "Roshan Sahani" {
			t.Fatalf("name changed to %q", got)
		}
	})

	t.Run("an admin can rename and the page refreshes", func(t *testing.T) {
		admin := a.browser(t)
		admin.adminSignIn()
		status, header, _ := admin.do("POST", path, url.Values{"name": {"  Rohan   Verma "}}, map[string]string{"HX-Request": "true"})
		if status != http.StatusOK || header.Get("HX-Refresh") != "true" {
			t.Fatalf("rename = %d, refresh %q", status, header.Get("HX-Refresh"))
		}
		if got := nameOf(); got != "Rohan Verma" {
			t.Fatalf("name = %q", got)
		}
	})

	t.Run("a blank name is refused and keeps the old one", func(t *testing.T) {
		admin := a.browser(t)
		admin.adminSignIn()
		status, header, body := admin.do("POST", path, url.Values{"name": {"   "}}, map[string]string{"HX-Request": "true"})
		wantTarget := "#person-error-" + strconv.FormatInt(a.user.ID, 10)
		if status != http.StatusOK || header.Get("HX-Refresh") == "true" ||
			header.Get("HX-Retarget") != wantTarget || !strings.Contains(body, "name is required") {
			t.Fatalf("blank rename = %d, retarget %q, body %q", status, header.Get("HX-Retarget"), body)
		}
		if got := nameOf(); got != "Rohan Verma" {
			t.Fatalf("name = %q", got)
		}
	})

	t.Run("an unknown person is a 404", func(t *testing.T) {
		admin := a.browser(t)
		admin.adminSignIn()
		if status, _, _ := admin.do("POST", "/admin/users/9999/rename", url.Values{"name": {"X"}}, nil); status != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", status)
		}
	})
}
