package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleStream pushes live lock state to the client via Server-Sent Events.
// One-way server→client updates fit the panel's needs and work cleanly behind
// Traefik without a protocol upgrade.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	changes, cancel := s.svc.Store().Subscribe()
	defer cancel()

	send := func() bool {
		st, pins, events := s.svc.Store().Snapshot()
		payload, err := json.Marshal(lockResponse{State: st, Pins: pins, Events: events})
		if err != nil {
			return false
		}
		if _, err := w.Write([]byte("event: state\ndata: ")); err != nil {
			return false
		}
		if _, err := w.Write(payload); err != nil {
			return false
		}
		if _, err := w.Write([]byte("\n\n")); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// Initial snapshot.
	if !send() {
		return
	}

	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-changes:
			if !send() {
				return
			}
		case <-keepalive.C:
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
