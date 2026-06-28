package lock

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parsed is the set of fields extracted from a single Z2M state message. Every
// field is optional (pointer / slice) so that partial updates merge cleanly.
type Parsed struct {
	LockState      *string
	Available      *bool
	Battery        *float64
	Voltage        *float64
	LinkQuality    *int
	SoundVolume    *string
	AutoRelock     *bool
	AutoRelockTime *int
	MaxPinUsers    *int
	MinPinLength   *int
	MaxPinLength   *int

	LastUnlockSource *string
	LastUnlockUser   *int
	LastLockSource   *string
	LastLockUser     *int

	Pins []PinUpdate
}

// PinUpdate is a single PIN slot observed in a state message.
type PinUpdate struct {
	User        int
	UserType    string
	UserEnabled bool
	HasCode     bool
	Delete      bool
}

// Parse decodes a Z2M lock state payload. It is intentionally permissive about
// types (the firmware/Z2M version emits booleans as ON/OFF strings, numbers as
// strings, etc.) and about the pin_code shape, which varies by firmware.
func Parse(payload []byte) (Parsed, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Parsed{}, fmt.Errorf("decode lock payload: %w", err)
	}

	var p Parsed
	if v, ok := raw["state"]; ok {
		if s := normalizeLockState(asString(v)); s != "" {
			p.LockState = &s
		}
	}
	if v, ok := raw["lock_state"]; ok && p.LockState == nil {
		if s := normalizeLockState(asString(v)); s != "" {
			p.LockState = &s
		}
	}
	if v, ok := raw["battery"]; ok {
		p.Battery = asFloatPtr(v)
	}
	if v, ok := raw["voltage"]; ok {
		p.Voltage = asFloatPtr(v)
	}
	if v, ok := raw["linkquality"]; ok {
		p.LinkQuality = asIntPtr(v)
	}
	if v, ok := raw["sound_volume"]; ok {
		s := strings.Trim(asString(v), `"`)
		p.SoundVolume = &s
	}
	if v, ok := raw["auto_relock"]; ok {
		p.AutoRelock = asBoolPtr(v)
	}
	if v, ok := raw["auto_relock_time"]; ok {
		p.AutoRelockTime = asIntPtr(v)
	}
	if v, ok := raw["max_pin_users"]; ok {
		p.MaxPinUsers = asIntPtr(v)
	}
	if v, ok := raw["min_pin_length"]; ok {
		p.MinPinLength = asIntPtr(v)
	}
	if v, ok := raw["max_pin_length"]; ok {
		p.MaxPinLength = asIntPtr(v)
	}
	if v, ok := raw["last_unlock_source"]; ok {
		s := asString(v)
		p.LastUnlockSource = &s
	}
	if v, ok := raw["last_unlock_user"]; ok {
		p.LastUnlockUser = asIntPtr(v)
	}
	if v, ok := raw["last_lock_source"]; ok {
		s := asString(v)
		p.LastLockSource = &s
	}
	if v, ok := raw["last_lock_user"]; ok {
		p.LastLockUser = asIntPtr(v)
	}

	if v, ok := raw["pin_code"]; ok {
		p.Pins = parsePins(v)
	}
	// Some firmwares publish the user table under "users".
	if v, ok := raw["users"]; ok && len(p.Pins) == 0 {
		p.Pins = parsePins(v)
	}

	return p, nil
}

// parsePins handles the several shapes a pin_code payload can take:
//   - a single object {"user":N,"user_type":"...","user_enabled":true,"pin_code":"1234"}
//   - an array of such objects
//   - an empty string / null (no data)
func parsePins(v json.RawMessage) []PinUpdate {
	trimmed := strings.TrimSpace(string(v))
	if trimmed == "" || trimmed == `""` || trimmed == "null" {
		return nil
	}

	// Try array first.
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(v, &arr); err == nil {
		out := make([]PinUpdate, 0, len(arr))
		for _, m := range arr {
			if pu, ok := pinFromMap(m); ok {
				out = append(out, pu)
			}
		}
		return out
	}

	// Try single object.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(v, &m); err == nil {
		if pu, ok := pinFromMap(m); ok {
			return []PinUpdate{pu}
		}
	}
	return nil
}

func pinFromMap(m map[string]json.RawMessage) (PinUpdate, bool) {
	userPtr := asIntPtr(m["user"])
	if userPtr == nil {
		return PinUpdate{}, false
	}
	pu := PinUpdate{User: *userPtr}
	if v, ok := m["user_type"]; ok {
		pu.UserType = asString(v)
	}
	if v, ok := m["user_enabled"]; ok {
		if b := asBoolPtr(v); b != nil {
			pu.UserEnabled = *b
		}
	}
	if v, ok := m["pin_code"]; ok {
		code := strings.TrimSpace(string(v))
		if code == "null" || code == `""` || code == "" {
			pu.Delete = true
		} else {
			pu.HasCode = true
		}
	}
	return pu, true
}

func normalizeLockState(s string) string {
	switch strings.ToLower(strings.Trim(s, `"`)) {
	case "lock", "locked":
		return "locked"
	case "unlock", "unlocked":
		return "unlocked"
	case "not_fully_locked", "not fully locked":
		return "not_fully_locked"
	default:
		return ""
	}
}

func isJSONNull(v json.RawMessage) bool {
	return strings.TrimSpace(string(v)) == "null"
}

func asString(v json.RawMessage) string {
	if isJSONNull(v) {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return s
	}
	return strings.Trim(string(v), `"`)
}

func asFloatPtr(v json.RawMessage) *float64 {
	// JSON null unmarshals into a float without error but leaves it at 0;
	// treat it as "absent" so callers see nil rather than a bogus 0.
	if isJSONNull(v) {
		return nil
	}
	var f float64
	if err := json.Unmarshal(v, &f); err == nil {
		return &f
	}
	if s := asString(v); s != "" {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return &f
		}
	}
	return nil
}

func asIntPtr(v json.RawMessage) *int {
	if f := asFloatPtr(v); f != nil {
		i := int(*f)
		return &i
	}
	return nil
}

func asBoolPtr(v json.RawMessage) *bool {
	if isJSONNull(v) {
		return nil
	}
	var b bool
	if err := json.Unmarshal(v, &b); err == nil {
		return &b
	}
	switch strings.ToUpper(strings.Trim(asString(v), `"`)) {
	case "ON", "TRUE", "LOCK", "1", "YES":
		t := true
		return &t
	case "OFF", "FALSE", "UNLOCK", "0", "NO":
		f := false
		return &f
	}
	return nil
}

// Ingest merges a parsed message into the store, deriving activity events when
// the last lock/unlock source/user changes.
func (s *Store) Ingest(p Parsed) {
	now := time.Now()
	s.mu.Lock()

	st := &s.state
	st.Available = true
	if p.LockState != nil {
		st.LockState = *p.LockState
	}
	if p.Battery != nil {
		st.Battery = p.Battery
	}
	if p.Voltage != nil {
		st.Voltage = p.Voltage
	}
	if p.LinkQuality != nil {
		st.LinkQuality = p.LinkQuality
	}
	if p.SoundVolume != nil {
		st.SoundVolume = *p.SoundVolume
	}
	if p.AutoRelock != nil {
		st.AutoRelock = p.AutoRelock
	}
	if p.AutoRelockTime != nil {
		st.AutoRelockTime = p.AutoRelockTime
	}
	if p.MaxPinUsers != nil {
		st.MaxPinUsers = p.MaxPinUsers
	}
	if p.MinPinLength != nil {
		st.MinPinLength = p.MinPinLength
	}
	if p.MaxPinLength != nil {
		st.MaxPinLength = p.MaxPinLength
	}
	st.UpdatedAt = now

	// Derive events from changes in last_(un)lock_source/user.
	var newEvents []Event
	if p.LastUnlockSource != nil || p.LastUnlockUser != nil {
		sig := sigOf(p.LastUnlockSource, p.LastUnlockUser)
		if sig != "" && sig != s.lastUnlockSig {
			if s.seeded {
				newEvents = append(newEvents, Event{
					Kind:   EventUnlock,
					Source: strOr(p.LastUnlockSource, "unknown"),
					User:   p.LastUnlockUser,
					At:     now,
				})
			}
			s.lastUnlockSig = sig
		}
	}
	if p.LastLockSource != nil || p.LastLockUser != nil {
		sig := sigOf(p.LastLockSource, p.LastLockUser)
		if sig != "" && sig != s.lastLockSig {
			if s.seeded {
				newEvents = append(newEvents, Event{
					Kind:   EventLock,
					Source: strOr(p.LastLockSource, "unknown"),
					User:   p.LastLockUser,
					At:     now,
				})
			}
			s.lastLockSig = sig
		}
	}
	for _, e := range newEvents {
		s.events = append(s.events, e)
	}
	if len(s.events) > maxEvents {
		s.events = s.events[len(s.events)-maxEvents:]
	}

	// Merge PIN slots.
	for _, pu := range p.Pins {
		if pu.Delete {
			delete(s.pins, pu.User)
			continue
		}
		s.pins[pu.User] = PinCode{
			User:        pu.User,
			UserType:    pu.UserType,
			UserEnabled: pu.UserEnabled,
			HasCode:     pu.HasCode,
		}
	}

	s.seeded = true
	s.mu.Unlock()
	s.notify()
}

func sigOf(source *string, user *int) string {
	var b strings.Builder
	if source != nil {
		b.WriteString(*source)
	}
	b.WriteByte('|')
	if user != nil {
		b.WriteString(strconv.Itoa(*user))
	}
	return b.String()
}

func strOr(s *string, def string) string {
	if s != nil && *s != "" {
		return *s
	}
	return def
}
