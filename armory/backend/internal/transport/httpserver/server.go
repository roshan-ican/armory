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
	pages      map[string]*template.Template
}

func New(categories *services.CategoryService) (*Server, error) {
	pages, err := loadPages("categories")
	if err != nil {
		return nil, err
	}
	return &Server{categories: categories, pages: pages}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.Handle("GET /static/", http.FileServerFS(web.FS))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/categories", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /categories", s.listCategories)
	mux.HandleFunc("POST /categories", s.createCategory)
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

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
