package httpserver

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"armory/internal/services"
)

const sessionCookie = "armory_session"

func (s *Server) session(r *http.Request) (services.Session, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return services.Session{}, false
	}
	return s.sessions.Lookup(c.Value)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) endSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.sessions.End(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if sess, ok := s.session(r); ok && sess.IsAdmin() {
			next(w, r)
			return
		}
		switch {
		case r.Header.Get("HX-Request") == "true":
			w.Header().Set("HX-Redirect", "/admin/login")
			w.WriteHeader(http.StatusUnauthorized)
		case r.Method == http.MethodGet:
			http.Redirect(w, r, "/admin/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
		default:
			http.Error(w, "sign in as an admin first", http.StatusUnauthorized)
		}
	}
}

func (s *Server) signedIn(next func(http.ResponseWriter, *http.Request, services.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.session(r)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "sign in first")
			return
		}
		next(w, r, sess)
	}
}

func safeNext(next string) string {
	if strings.HasPrefix(next, "/admin/") && !strings.HasPrefix(next, "//") && !strings.Contains(next, "\\") {
		return next
	}
	return "/admin/lockers"
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	hasAdmin, err := s.users.HasLoginAdmin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "login", "login", map[string]any{"Next": safeNext(r.URL.Query().Get("next")), "Setup": !hasAdmin})
}

func (s *Server) setupAdmin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	u, err := s.users.SetupAdmin(r.Context(), username, username, r.FormValue("password"))
	if err != nil {
		formError(w, err)
		return
	}
	s.startSession(w, r, s.sessions.Start(u))
	w.Header().Set("HX-Redirect", safeNext(r.FormValue("next")))
}

func (s *Server) loginAdmin(w http.ResponseWriter, r *http.Request) {
	u, err := s.users.AuthenticateAdmin(r.Context(), r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		formError(w, err)
		return
	}
	s.startSession(w, r, s.sessions.Start(u))
	w.Header().Set("HX-Redirect", safeNext(r.FormValue("next")))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.endSession(w, r)
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}
