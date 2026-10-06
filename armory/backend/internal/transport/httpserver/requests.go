package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"armory/internal/models"
	"armory/internal/services"
)

type slotJSON struct {
	No      int64 `json:"no"`
	Reading int64 `json:"reading"`
}

type chosenJSON struct {
	No     int64  `json:"no"`
	Status string `json:"status"`
}

type requestJSON struct {
	ID               int64        `json:"id"`
	Status           string       `json:"status"`
	Kind             string       `json:"kind"`
	LockerName       string       `json:"locker_name"`
	SlotNo           int64        `json:"slot_no"`
	Chosen           []chosenJSON `json:"chosen"`
	ExpectedReturnAt string       `json:"expected_return_at"`
	Slots            []slotJSON   `json:"slots"`
	Wrong            []int64      `json:"wrong"`
}

func toJSON(v services.RequestView) requestJSON {
	out := requestJSON{
		ID:               v.Request.ID,
		Status:           v.Request.Status,
		Kind:             v.Request.Kind,
		LockerName:       v.Request.LockerName,
		Chosen:           []chosenJSON{},
		ExpectedReturnAt: v.Request.ExpectedReturnAt,
		Slots:            []slotJSON{},
		Wrong:            []int64{},
	}
	for _, sl := range v.Request.Slots {
		out.Chosen = append(out.Chosen, chosenJSON{No: sl.SlotNo, Status: sl.Status})
	}
	if len(v.Request.Slots) > 0 {
		out.SlotNo = v.Request.Slots[0].SlotNo
	}
	for _, sl := range v.Slots {
		out.Slots = append(out.Slots, slotJSON{No: sl.SlotNo, Reading: sl.Reading})
	}
	out.Wrong = append(out.Wrong, v.Wrong...)
	return out
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func requestErrorStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, services.ErrInvalidKind),
		errors.Is(err, services.ErrInvalidReturn),
		errors.Is(err, services.ErrInvalidSlot):
		return http.StatusBadRequest, true
	case errors.Is(err, services.ErrRequestNotFound):
		return http.StatusNotFound, true
	case errors.Is(err, services.ErrRequestOpen),
		errors.Is(err, services.ErrRequestClosed),
		errors.Is(err, services.ErrNoneAvailable),
		errors.Is(err, services.ErrGunUnavailable):
		return http.StatusConflict, true
	case errors.Is(err, services.ErrNotAdmin):
		return http.StatusForbidden, true
	}
	return 0, false
}

func (s *Server) apiError(w http.ResponseWriter, err error) {
	if status, ok := requestErrorStatus(err); ok {
		writeJSONError(w, status, err.Error())
		return
	}
	serverError(w, err)
}

func (s *Server) apiMe(w http.ResponseWriter, r *http.Request, sess services.Session) {
	writeJSON(w, map[string]any{"name": sess.Name, "role": sess.Role})
}

func (s *Server) apiAvailability(w http.ResponseWriter, r *http.Request, sess services.Session) {
	got, err := s.requests.Availability(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, got)
}

func (s *Server) apiCatalog(w http.ResponseWriter, r *http.Request, sess services.Session) {
	items, err := s.requests.Catalog(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, items)
}

func (s *Server) apiCreateRequest(w http.ResponseWriter, r *http.Request, sess services.Session) {
	var body struct {
		Kind     string  `json:"kind"`
		LockerID int64   `json:"locker_id"`
		Slots    []int64 `json:"slots"`
		Reason   string  `json:"reason"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	var req models.Request
	var err error
	if body.LockerID != 0 {
		req, err = s.requests.CreateForLocker(r.Context(), sess.UserID, body.LockerID, body.Slots, body.Reason)
	} else {
		req, err = s.requests.Create(r.Context(), sess.UserID, body.Kind, body.Reason)
	}
	if err != nil {
		s.apiError(w, err)
		return
	}
	v, err := s.requests.View(r.Context(), sess.UserID, req.ID)
	if err != nil {
		s.apiError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, toJSON(v))
}

func (s *Server) apiCurrentRequest(w http.ResponseWriter, r *http.Request, sess services.Session) {
	v, err := s.requests.Reopen(r.Context(), sess.UserID)
	if err != nil {
		s.apiError(w, err)
		return
	}
	writeJSON(w, toJSON(v))
}

func (s *Server) apiGetRequest(w http.ResponseWriter, r *http.Request, sess services.Session) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "request not found")
		return
	}
	v, err := s.requests.View(r.Context(), sess.UserID, id)
	if err != nil {
		s.apiError(w, err)
		return
	}
	writeJSON(w, toJSON(v))
}

func (s *Server) apiCancelRequest(w http.ResponseWriter, r *http.Request, sess services.Session) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "request not found")
		return
	}
	if err := s.requests.Cancel(r.Context(), sess.UserID, id); err != nil {
		s.apiError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) requestsPage(w http.ResponseWriter, r *http.Request) {
	pending, err := s.requests.Pending(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	recent, err := s.requests.Recent(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "requests", "layout", map[string]any{
		"Title":   "Requests",
		"Pending": pending,
		"History": recent,
	})
}

func (s *Server) requestsBadge(w http.ResponseWriter, r *http.Request) {
	pending, err := s.requests.Pending(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if len(pending) > 0 {
		w.Write([]byte(strconv.Itoa(len(pending))))
	}
}

func (s *Server) decideRequest(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := "#request-error-" + r.PathValue("id")
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		sess, ok := s.session(r)
		if !ok || !sess.IsAdmin() {
			formErrorAt(w, services.ErrNotAdmin, target)
			return
		}
		if approve {
			_, err = s.requests.Approve(r.Context(), sess.UserID, id)
		} else {
			err = s.requests.Reject(r.Context(), sess.UserID, id)
		}
		if _, known := requestErrorStatus(err); known {
			formErrorAt(w, err, target)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
