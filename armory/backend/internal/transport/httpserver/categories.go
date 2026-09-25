package httpserver

import (
	"errors"
	"html/template"
	"net/http"

	"armory/internal/services"
)

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.categories.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "categories", "layout", map[string]any{
		"Title":      "Categories",
		"Categories": categories,
	})
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	c, err := s.categories.Create(r.Context(), r.FormValue("name"))
	if errors.Is(err, services.ErrNameRequired) || errors.Is(err, services.ErrCategoryExists) {
		w.Header().Set("HX-Retarget", "#form-error")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.Write([]byte(template.HTMLEscapeString(err.Error())))
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "categories", "category_row", c)
}
