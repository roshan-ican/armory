package httpserver

import (
	"html/template"
	"log"
	"net/http"

	"armory/internal/services"
	"armory/web"
)

type Server struct {
	categories *services.CategoryService
	lockers    *services.LockerService
	guns       *services.GunService
	pages      map[string]*template.Template
}

func New(categories *services.CategoryService, lockers *services.LockerService, guns *services.GunService) (*Server, error) {
	pages, err := loadPages("categories", "lockers", "guns")
	if err != nil {
		return nil, err
	}
	return &Server{categories: categories, lockers: lockers, guns: guns, pages: pages}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.Handle("GET /static/", http.FileServerFS(web.FS))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/guns", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /categories", s.listCategories)
	mux.HandleFunc("POST /categories", s.createCategory)
	mux.HandleFunc("GET /lockers", s.listLockers)
	mux.HandleFunc("POST /lockers", s.createLocker)
	mux.HandleFunc("GET /guns", s.listGuns)
	mux.HandleFunc("POST /guns", s.createGun)
	mux.HandleFunc("POST /guns/{id}/retire", s.retireGun)
	mux.HandleFunc("GET /guns/{id}", s.gunRow)
	mux.HandleFunc("GET /guns/{id}/edit", s.editGun)
	mux.HandleFunc("POST /guns/{id}", s.updateGun)
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
	w.Header().Set("HX-Retarget", "#form-error")
	w.Header().Set("HX-Reswap", "innerHTML")
	w.Write([]byte(template.HTMLEscapeString(err.Error())))
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
