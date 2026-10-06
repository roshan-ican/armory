package httpserver

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"

	"armory/internal/live"
	"armory/internal/services"
	"armory/web"
)

type Server struct {
	lockers  *services.LockerService
	activity *services.ActivityService
	users    *services.UserService
	face     *services.FaceService
	requests *services.RequestService
	sessions *services.Sessions
	caPath   string
	hub      *live.Hub
	pages    map[string]*template.Template
}

func New(lockers *services.LockerService, activity *services.ActivityService, users *services.UserService, face *services.FaceService, requests *services.RequestService, sessions *services.Sessions, hub *live.Hub) (*Server, error) {
	pages, err := loadPages("lockers", "activity", "users", "requests", "login")
	if err != nil {
		return nil, err
	}
	return &Server{lockers: lockers, activity: activity, users: users, face: face, requests: requests, sessions: sessions, hub: hub, pages: pages}, nil
}

func (s *Server) ServeCA(path string) {
	s.caPath = path
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	if s.caPath != "" {
		mux.HandleFunc("GET /ca.crt", s.downloadCA)
	}
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /events", s.events)
	mux.Handle("GET /static/", noCache(http.FileServerFS(web.FS)))
	mux.HandleFunc("GET /admin.webmanifest", embeddedFile("admin.webmanifest", "application/manifest+json"))
	mux.HandleFunc("GET /sw.js", embeddedFile("sw.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("GET /{$}", redirectTo("/admin"))
	mux.HandleFunc("POST /enroll", s.requestEnrollment)
	mux.HandleFunc("POST /face/match", s.matchFace)

	mux.HandleFunc("GET /api/me", s.signedIn(s.apiMe))
	mux.HandleFunc("GET /api/availability", s.signedIn(s.apiAvailability))
	mux.HandleFunc("GET /api/catalog", s.signedIn(s.apiCatalog))
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("POST /api/requests", s.signedIn(s.apiCreateRequest))
	mux.HandleFunc("GET /api/requests/current", s.signedIn(s.apiCurrentRequest))
	mux.HandleFunc("GET /api/requests/{id}", s.signedIn(s.apiGetRequest))
	mux.HandleFunc("POST /api/requests/{id}/cancel", s.signedIn(s.apiCancelRequest))

	mux.HandleFunc("GET /admin/login", s.loginPage)
	mux.HandleFunc("POST /admin/login", s.loginAdmin)
	mux.HandleFunc("POST /admin/setup", s.setupAdmin)
	mux.HandleFunc("GET /admin/logout", s.logout)
	mux.HandleFunc("GET /admin/{$}", redirectTo("/admin/lockers"))
	mux.HandleFunc("GET /admin", redirectTo("/admin/lockers"))
	mux.HandleFunc("GET /admin/lockers", s.adminOnly(s.listLockers))
	mux.HandleFunc("POST /admin/lockers", s.adminOnly(s.createLocker))
	mux.HandleFunc("GET /admin/lockers/{id}", s.adminOnly(s.lockerCard))
	mux.HandleFunc("GET /admin/lockers/{id}/edit", s.adminOnly(s.editLocker))
	mux.HandleFunc("POST /admin/lockers/{id}", s.adminOnly(s.updateLocker))
	mux.HandleFunc("POST /admin/lockers/{id}/swap", s.adminOnly(s.swapSensors))
	mux.HandleFunc("GET /admin/activity", s.adminOnly(s.listActivity))
	mux.HandleFunc("GET /admin/users", s.adminOnly(s.listUsers))
	mux.HandleFunc("GET /admin/users/badge", s.adminOnly(s.peopleBadge))
	mux.HandleFunc("POST /admin/users/{id}/remove", s.adminOnly(s.removeUser))
	mux.HandleFunc("POST /admin/enrollments/{id}/approve", s.adminOnly(s.decideEnrollment(true)))
	mux.HandleFunc("POST /admin/enrollments/{id}/reject", s.adminOnly(s.decideEnrollment(false)))
	mux.HandleFunc("GET /admin/requests", s.adminOnly(s.requestsPage))
	mux.HandleFunc("GET /admin/requests/badge", s.adminOnly(s.requestsBadge))
	mux.HandleFunc("POST /admin/requests/{id}/approve", s.adminOnly(s.decideRequest(true)))
	mux.HandleFunc("POST /admin/requests/{id}/reject", s.adminOnly(s.decideRequest(false)))

	for _, old := range []string{"lockers", "requests", "activity", "users", "login"} {
		mux.HandleFunc("GET /"+old, redirectTo("/admin/"+old))
	}
	return mux
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

func loadPages(names ...string) (map[string]*template.Template, error) {
	pages := make(map[string]*template.Template)
	for _, name := range names {
		t, err := template.ParseFS(web.FS, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, err
		}
		pages[name] = t
	}
	return pages, nil
}

func (s *Server) render(w http.ResponseWriter, page, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.pages[page].ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s/%s: %v", page, name, err)
	}
}

func formError(w http.ResponseWriter, err error) {
	formErrorAt(w, err, "#form-error")
}

func formErrorAt(w http.ResponseWriter, err error, target string) {
	w.Header().Set("HX-Retarget", target)
	w.Header().Set("HX-Reswap", "innerHTML")
	w.Write([]byte(template.HTMLEscapeString(err.Error())))
}

func serverError(w http.ResponseWriter, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	log.Printf("internal error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
