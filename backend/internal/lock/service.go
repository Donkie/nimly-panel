package lock

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Publisher publishes a payload to an MQTT topic.
type Publisher interface {
	Publish(topic string, payload []byte) error
}

// ValidationError indicates the caller supplied invalid input. Handlers map it
// to HTTP 400.
type ValidationError struct{ msg string }

func (e ValidationError) Error() string { return e.msg }

func invalid(format string, a ...any) error { return ValidationError{fmt.Sprintf(format, a...)} }

// Service exposes high-level lock operations built on top of the MQTT publisher
// and the shared state cache.
type Service struct {
	pub       Publisher
	store     *Store
	baseTopic string
}

// NewService wires a Service to its publisher, cache and the lock's Z2M base
// topic (e.g. "zigbee2mqtt/Front Door Lock").
func NewService(pub Publisher, store *Store, baseTopic string) *Service {
	return &Service{pub: pub, store: store, baseTopic: baseTopic}
}

// Store returns the underlying state cache.
func (s *Service) Store() *Store { return s.store }

func (s *Service) setTopic() string { return s.baseTopic + "/set" }
func (s *Service) getTopic() string { return s.baseTopic + "/get" }

func (s *Service) publishJSON(topic string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.pub.Publish(topic, b)
}

// SetLockState locks or unlocks the door. state must be "lock" or "unlock".
func (s *Service) SetLockState(state string) error {
	var cmd string
	switch state {
	case "lock", "LOCK", "locked":
		cmd = "LOCK"
	case "unlock", "UNLOCK", "unlocked":
		cmd = "UNLOCK"
	default:
		return invalid("state must be 'lock' or 'unlock'")
	}
	return s.publishJSON(s.setTopic(), map[string]string{"state": cmd})
}

// PinRequest is the validated input for setting a PIN slot.
type PinRequest struct {
	UserType    string `json:"user_type"`
	UserEnabled bool   `json:"user_enabled"`
	PinCode     string `json:"pin_code"`
}

// SetPin programs (creates or updates) a PIN slot for the given user index.
func (s *Service) SetPin(user int, req PinRequest) error {
	st := s.store.State()
	if err := s.validateUser(user, st); err != nil {
		return err
	}

	if req.UserType == "" {
		req.UserType = UserTypeUnrestricted
	}
	if !ValidUserTypes[req.UserType] {
		return invalid("invalid user_type %q", req.UserType)
	}

	if err := validatePin(req.PinCode, st); err != nil {
		return err
	}
	code, err := strconv.Atoi(req.PinCode)
	if err != nil {
		return invalid("pin_code must be numeric")
	}

	payload := map[string]any{
		"pin_code": map[string]any{
			"user":         user,
			"user_type":    req.UserType,
			"user_enabled": req.UserEnabled,
			"pin_code":     code,
		},
	}
	if err := s.publishJSON(s.setTopic(), payload); err != nil {
		return err
	}
	// Optimistic cache update; the device will confirm via a state message.
	s.store.UpsertPin(PinCode{
		User:        user,
		UserType:    req.UserType,
		UserEnabled: req.UserEnabled,
		HasCode:     true,
	})
	return nil
}

// ClearPin removes a PIN slot.
func (s *Service) ClearPin(user int) error {
	if err := s.validateUser(user, s.store.State()); err != nil {
		return err
	}
	payload := map[string]any{
		"pin_code": map[string]any{
			"user":     user,
			"pin_code": nil,
		},
	}
	if err := s.publishJSON(s.setTopic(), payload); err != nil {
		return err
	}
	s.store.DeletePin(user)
	return nil
}

// RefreshPins asks the lock to report all PIN slots and the PIN constraints.
func (s *Service) RefreshPins() error {
	return s.publishJSON(s.getTopic(), map[string]string{"pin_code": ""})
}

// RefreshConstraints polls the read-only PIN constraints and core attributes.
func (s *Service) RefreshConstraints() error {
	return s.publishJSON(s.getTopic(), map[string]string{
		"state":            "",
		"max_pin_users":    "",
		"min_pin_length":   "",
		"max_pin_length":   "",
		"auto_relock_time": "",
		"sound_volume":     "",
		"battery":          "",
	})
}

// SetSoundVolume changes the lock's sound volume.
func (s *Service) SetSoundVolume(v string) error {
	if !ValidSoundVolumes[v] {
		return invalid("invalid sound_volume %q", v)
	}
	return s.publishJSON(s.setTopic(), map[string]string{"sound_volume": v})
}

// SetAutoRelock toggles the auto-relock feature.
func (s *Service) SetAutoRelock(on bool) error {
	val := "OFF"
	if on {
		val = "ON"
	}
	return s.publishJSON(s.setTopic(), map[string]string{"auto_relock": val})
}

func (s *Service) validateUser(user int, st State) error {
	if user < 0 {
		return invalid("user must be >= 0")
	}
	if st.MaxPinUsers != nil && user >= *st.MaxPinUsers {
		return invalid("user must be < max_pin_users (%d)", *st.MaxPinUsers)
	}
	return nil
}

func validatePin(pin string, st State) error {
	if pin == "" {
		return invalid("pin_code is required")
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return invalid("pin_code must contain digits only")
		}
	}
	minLen, maxLen := 4, 10
	if st.MinPinLength != nil {
		minLen = *st.MinPinLength
	}
	if st.MaxPinLength != nil {
		maxLen = *st.MaxPinLength
	}
	if len(pin) < minLen || len(pin) > maxLen {
		return invalid("pin_code length must be between %d and %d digits", minLen, maxLen)
	}
	return nil
}

// IsValidation reports whether err is a ValidationError.
func IsValidation(err error) bool {
	var ve ValidationError
	return errors.As(err, &ve)
}
