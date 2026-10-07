package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"armory/internal/services"
)

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.users.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	pending, err := s.face.PendingEnrollments(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "users", "layout", map[string]any{
		"Title":   "People",
		"Users":   users,
		"Pending": pending,
	})
}

func (s *Server) requestEnrollment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string      `json:"name"`
		Descriptors [][]float64 `json:"descriptors"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	err := s.face.RequestEnrollment(r.Context(), body.Name, body.Descriptors)
	if errors.Is(err, services.ErrInvalidDescriptor) || errors.Is(err, services.ErrNameRequired) ||
		errors.Is(err, services.ErrEnrollmentPending) || errors.Is(err, services.ErrFaceAlreadyEnrolled) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.hub.Publish()
	writeJSON(w, map[string]any{"pending": true})
}

func (s *Server) peopleBadge(w http.ResponseWriter, r *http.Request) {
	pending, err := s.face.PendingEnrollments(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if len(pending) > 0 {
		w.Write([]byte(strconv.Itoa(len(pending))))
	}
}

func (s *Server) decideEnrollment(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.session(r)
		if !ok || !sess.IsAdmin() {
			http.Error(w, "sign in as an admin first", http.StatusUnauthorized)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		err = s.face.DecideEnrollment(r.Context(), id, sess.UserID, approve)
		if errors.Is(err, services.ErrEnrollmentNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, services.ErrEnrollmentHandled) || errors.Is(err, services.ErrUserExists) {
			formErrorAt(w, err, "#enrollment-error-"+r.PathValue("id"))
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		s.hub.Publish()
		w.Header().Set("HX-Refresh", "true")
	}
}

func (s *Server) removeUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	err = s.users.Remove(r.Context(), id)
	if errors.Is(err, services.ErrUserNotFound) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, services.ErrUserBusy) || errors.Is(err, services.ErrCannotRemoveAdmin) {
		formErrorAt(w, err, "#person-error-"+r.PathValue("id"))
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.hub.Publish()
	w.Header().Set("HX-Refresh", "true")
}

func (s *Server) renameUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	err = s.users.Rename(r.Context(), id, r.FormValue("name"))
	if errors.Is(err, services.ErrUserNotFound) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, services.ErrNameRequired) {
		formErrorAt(w, err, "#person-error-"+r.PathValue("id"))
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.hub.Publish()
	w.Header().Set("HX-Refresh", "true")
}
