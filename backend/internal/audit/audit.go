// Package audit records security-relevant actions (lock/unlock, PIN changes,
// auth events) with the authenticated actor and a timestamp. Entries always go
// to the structured application log; if a path is configured they are also
// appended as JSON lines to a file (e.g. on a mounted volume).
package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Entry is a single audit record.
type Entry struct {
	Time    time.Time      `json:"time"`
	Actor   string         `json:"actor"`   // authenticated subject/email
	Action  string         `json:"action"`  // e.g. "pin.set", "lock.unlock"
	Target  string         `json:"target,omitempty"`
	Result  string         `json:"result"`  // "ok" | "error"
	Details map[string]any `json:"details,omitempty"`
}

// Logger writes audit entries.
type Logger struct {
	log  *slog.Logger
	mu   sync.Mutex
	file *os.File
}

// New creates an audit logger. If path is non-empty, entries are also appended
// to that file as JSON lines.
func New(log *slog.Logger, path string) (*Logger, error) {
	l := &Logger{log: log}
	if path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}
		l.file = f
	}
	return l, nil
}

// Close releases the audit file handle if open.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Record writes an audit entry.
func (l *Logger) Record(e Entry) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	l.log.Info("audit",
		"actor", e.Actor,
		"action", e.Action,
		"target", e.Target,
		"result", e.Result,
		"details", e.Details,
	)
	if l.file == nil {
		return
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	b = append(b, '\n')
	_, _ = l.file.Write(b)
}

// OK records a successful action.
func (l *Logger) OK(actor, action, target string, details map[string]any) {
	l.Record(Entry{Actor: actor, Action: action, Target: target, Result: "ok", Details: details})
}

// Fail records a failed action.
func (l *Logger) Fail(actor, action, target string, details map[string]any) {
	l.Record(Entry{Actor: actor, Action: action, Target: target, Result: "error", Details: details})
}
