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
	hub      *live.Hub
	pages    map[string]*template.Template
}

func New(lockers *services.LockerService, activity *services.ActivityService, hub *live.Hub) (*Server, error) {
	pages, err := loadPages("lockers", "activity")
	if err != nil {
		return nil, err
	}
	return &Server{lockers: lockers, activity: activity, hub: hub, pages: pages}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /events", s.events)
	mux.Handle("GET /static/", http.FileServerFS(web.FS))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/lockers", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /lockers", s.listLockers)
	mux.HandleFunc("POST /lockers", s.createLocker)
	mux.HandleFunc("GET /lockers/{id}", s.lockerCard)
	mux.HandleFunc("GET /lockers/{id}/edit", s.editLocker)
	mux.HandleFunc("POST /lockers/{id}", s.updateLocker)
	mux.HandleFunc("GET /activity", s.listActivity)
	return mux
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
