package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"armory/internal/models"
	"armory/internal/services"
)

func (s *Server) assignMaintenance(w http.ResponseWriter, r *http.Request) {
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
	err = s.maintain.Assign(r.Context(), sess.UserID, id)
	if errors.Is(err, services.ErrUserNotFound) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, services.ErrNotRequester) || errors.Is(err, services.ErrNoFaceEnrolled) || errors.Is(err, services.ErrMaintenanceOpen) {
		formErrorAt(w, err, "#person-error-"+r.PathValue("id"))
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
}

func (s *Server) cancelMaintenance(w http.ResponseWriter, r *http.Request) {
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
	err = s.maintain.Cancel(r.Context(), sess.UserID, id)
	if errors.Is(err, services.ErrMaintenanceNotFound) {
		w.Header().Set("HX-Refresh", "true")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
}

func (s *Server) apiFinishMaintenance(w http.ResponseWriter, r *http.Request, sess services.Session) {
	err := s.maintain.Finish(r.Context(), sess.UserID)
	if errors.Is(err, services.ErrMaintenanceNotFound) {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, map[string]any{"finished": true})
}

type changeJSON struct {
	LockerID   int64  `json:"locker_id"`
	LockerName string `json:"locker_name"`
	Action     string `json:"action"`
	SlotNo     int64  `json:"slot_no"`
	Text       string `json:"text"`
}

type maintenanceJobJSON struct {
	ID      int64        `json:"id"`
	Status  string       `json:"status"`
	Changes []changeJSON `json:"changes"`
}

type maintenanceLockerJSON struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Kind     string     `json:"kind"`
	Capacity int64      `json:"capacity"`
	Slots    []slotJSON `json:"slots"`
}

func maintenanceStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, services.ErrEmptyPlan),
		errors.Is(err, services.ErrBadChange),
		errors.Is(err, services.ErrLockerNotFound):
		return http.StatusBadRequest, true
	case errors.Is(err, services.ErrNotRequester):
		return http.StatusForbidden, true
	case errors.Is(err, services.ErrMaintenanceOpen),
		errors.Is(err, services.ErrMaintenanceNotFound):
		return http.StatusConflict, true
	}
	return 0, false
}

func (s *Server) maintenanceError(w http.ResponseWriter, err error) {
	if status, ok := maintenanceStatus(err); ok {
		writeJSONError(w, status, err.Error())
		return
	}
	serverError(w, err)
}

func jobJSON(job models.MaintenanceJob) maintenanceJobJSON {
	out := maintenanceJobJSON{ID: job.ID, Status: job.Status, Changes: []changeJSON{}}
	for _, c := range job.Changes {
		out.Changes = append(out.Changes, changeJSON{
			LockerID: c.LockerID, LockerName: c.LockerName, Action: c.Action, SlotNo: c.SlotNo, Text: c.Text(),
		})
	}
	return out
}

func (s *Server) apiMaintenance(w http.ResponseWriter, r *http.Request, sess services.Session) {
	out := map[string]any{"job": nil, "lockers": []maintenanceLockerJSON{}}
	job, err := s.maintain.Current(r.Context(), sess.UserID)
	if err != nil && !errors.Is(err, services.ErrMaintenanceNotFound) {
		serverError(w, err)
		return
	}
	if err == nil {
		out["job"] = jobJSON(job)
	}

	lockers, err := s.maintain.Lockers(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	list := []maintenanceLockerJSON{}
	for _, l := range lockers {
		item := maintenanceLockerJSON{ID: l.ID, Name: l.Name, Kind: l.Kind, Capacity: l.Capacity, Slots: []slotJSON{}}
		for _, sl := range l.Slots {
			item.Slots = append(item.Slots, slotJSON{No: sl.SlotNo, Reading: sl.Reading})
		}
		list = append(list, item)
	}
	out["lockers"] = list
	writeJSON(w, out)
}

func (s *Server) apiRequestMaintenance(w http.ResponseWriter, r *http.Request, sess services.Session) {
	var body struct {
		Changes []struct {
			LockerID int64  `json:"locker_id"`
			Action   string `json:"action"`
			SlotNo   int64  `json:"slot_no"`
		} `json:"changes"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	changes := make([]models.MaintenanceChange, 0, len(body.Changes))
	for _, c := range body.Changes {
		changes = append(changes, models.MaintenanceChange{LockerID: c.LockerID, Action: c.Action, SlotNo: c.SlotNo})
	}
	job, err := s.maintain.Request(r.Context(), sess.UserID, changes)
	if err != nil {
		s.maintenanceError(w, err)
		return
	}
	full, err := s.maintain.Current(r.Context(), sess.UserID)
	if err != nil {
		full = job
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, jobJSON(full))
}

func (s *Server) apiBeginMaintenance(w http.ResponseWriter, r *http.Request, sess services.Session) {
	began, err := s.maintain.Begin(r.Context(), sess.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if !began {
		writeJSONError(w, http.StatusConflict, services.ErrMaintenanceNotFound.Error())
		return
	}
	writeJSON(w, map[string]any{"started": true, "say": services.MaintenanceStartMessage()})
}

func (s *Server) apiUnlockLocker(w http.ResponseWriter, r *http.Request, sess services.Session) {
	var body struct {
		LockerID int64 `json:"locker_id"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := s.maintain.UnlockLocker(r.Context(), sess.UserID, body.LockerID); err != nil {
		s.maintenanceError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) apiSetCapacity(w http.ResponseWriter, r *http.Request, sess services.Session) {
	var body struct {
		LockerID int64 `json:"locker_id"`
		Capacity int64 `json:"capacity"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := s.maintain.SetCapacity(r.Context(), sess.UserID, body.LockerID, body.Capacity); err != nil {
		s.maintenanceError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) approveMaintenance(w http.ResponseWriter, r *http.Request) {
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
	err = s.maintain.Approve(r.Context(), sess.UserID, id)
	if errors.Is(err, services.ErrMaintenanceNotFound) {
		w.Header().Set("HX-Refresh", "true")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
}
