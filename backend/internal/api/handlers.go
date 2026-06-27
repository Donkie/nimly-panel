package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Donkie/nimly-panel/backend/internal/auth"
	"github.com/Donkie/nimly-panel/backend/internal/lock"
)

// meResponse is returned by GET /api/me.
type meResponse struct {
	Subject   string   `json:"subject"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Groups    []string `json:"groups"`
	CSRFToken string   `json:"csrf_token"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r)
	writeJSON(w, http.StatusOK, meResponse{
		Subject:   u.Subject,
		Name:      u.Name,
		Email:     u.Email,
		Groups:    u.Groups,
		CSRFToken: s.auth.CSRFToken(r),
	})
}

// lockResponse is returned by GET /api/lock.
type lockResponse struct {
	State  lock.State     `json:"state"`
	Pins   []lock.PinCode `json:"pins"`
	Events []lock.Event   `json:"events"`
}

func (s *Server) handleGetLock(w http.ResponseWriter, r *http.Request) {
	st, pins, events := s.svc.Store().Snapshot()
	writeJSON(w, http.StatusOK, lockResponse{State: st, Pins: pins, Events: events})
}

func (s *Server) handleSetLockState(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r)
	var body struct {
		State string `json:"state"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.SetLockState(body.State); err != nil {
		s.audit.Fail(actorOf(u), "lock.state", body.State, map[string]any{"err": err.Error()})
		s.respondErr(w, err)
		return
	}
	s.audit.OK(actorOf(u), "lock.state", body.State, nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) handleGetPins(w http.ResponseWriter, r *http.Request) {
	_, pins, _ := s.svc.Store().Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{"pins": pins})
}

func (s *Server) handleSetPin(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r)
	user, err := strconv.Atoi(r.PathValue("user"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "user must be an integer")
		return
	}
	var req lock.PinRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.SetPin(user, req); err != nil {
		s.audit.Fail(actorOf(u), "pin.set", strconv.Itoa(user), map[string]any{"err": err.Error()})
		s.respondErr(w, err)
		return
	}
	// Never log the PIN digits — only metadata.
	s.audit.OK(actorOf(u), "pin.set", strconv.Itoa(user), map[string]any{
		"user_type":    req.UserType,
		"user_enabled": req.UserEnabled,
	})
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) handleDeletePin(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r)
	user, err := strconv.Atoi(r.PathValue("user"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "user must be an integer")
		return
	}
	if err := s.svc.ClearPin(user); err != nil {
		s.audit.Fail(actorOf(u), "pin.delete", strconv.Itoa(user), map[string]any{"err": err.Error()})
		s.respondErr(w, err)
		return
	}
	s.audit.OK(actorOf(u), "pin.delete", strconv.Itoa(user), nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) handleSoundVolume(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r)
	var body struct {
		SoundVolume string `json:"sound_volume"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.SetSoundVolume(body.SoundVolume); err != nil {
		s.audit.Fail(actorOf(u), "settings.sound_volume", body.SoundVolume, map[string]any{"err": err.Error()})
		s.respondErr(w, err)
		return
	}
	s.audit.OK(actorOf(u), "settings.sound_volume", body.SoundVolume, nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) handleAutoRelock(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r)
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.SetAutoRelock(body.Enabled); err != nil {
		s.audit.Fail(actorOf(u), "settings.auto_relock", strconv.FormatBool(body.Enabled), map[string]any{"err": err.Error()})
		s.respondErr(w, err)
		return
	}
	s.audit.OK(actorOf(u), "settings.auto_relock", strconv.FormatBool(body.Enabled), nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.RefreshConstraints(); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.svc.RefreshPins(); err != nil {
		s.respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) respondErr(w http.ResponseWriter, err error) {
	if lock.IsValidation(err) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.log.Error("operation failed", "err", err)
	writeErr(w, http.StatusBadGateway, "lock command failed: "+err.Error())
}

// decodeJSON decodes a JSON request body with a sane size limit and strict
// field checking.
func decodeJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<16)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
