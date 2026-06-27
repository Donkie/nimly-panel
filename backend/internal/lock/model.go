// Package lock models the Nimly / Onesti (easyCodeTouch) door lock as exposed
// over Zigbee2MQTT, maintains an in-memory cache of its state, and builds the
// MQTT payloads used to control it.
package lock

import "time"

// UserType values accepted by the lock for a PIN slot.
const (
	UserTypeUnrestricted     = "unrestricted"
	UserTypeYearDaySchedule  = "year_day_schedule"
	UserTypeWeekDaySchedule  = "week_day_schedule"
	UserTypeMaster           = "master"
	UserTypeNonAccess        = "non_access"
)

// ValidUserTypes is the set of user_type values the lock accepts.
var ValidUserTypes = map[string]bool{
	UserTypeUnrestricted:    true,
	UserTypeYearDaySchedule: true,
	UserTypeWeekDaySchedule: true,
	UserTypeMaster:          true,
	UserTypeNonAccess:       true,
}

// SoundVolume values accepted by the lock.
const (
	SoundSilent = "silent_mode"
	SoundLow    = "low_volume"
	SoundHigh   = "high_volume"
)

// ValidSoundVolumes is the set of sound_volume values the lock accepts.
var ValidSoundVolumes = map[string]bool{
	SoundSilent: true,
	SoundLow:    true,
	SoundHigh:   true,
}

// State is the cached, public-facing view of the lock. Secrets (actual PIN
// digits) are deliberately excluded — only whether a slot has a code is kept.
type State struct {
	LockState       string    `json:"lock_state"` // locked | unlocked | not_fully_locked | unknown
	BrokerConnected bool      `json:"broker_connected"`
	Available       bool      `json:"available"`
	Battery        *float64   `json:"battery,omitempty"`
	Voltage        *float64   `json:"voltage,omitempty"`
	LinkQuality    *int       `json:"link_quality,omitempty"`
	SoundVolume    string     `json:"sound_volume,omitempty"`
	AutoRelock     *bool      `json:"auto_relock,omitempty"`
	AutoRelockTime *int       `json:"auto_relock_time,omitempty"`
	MaxPinUsers    *int       `json:"max_pin_users,omitempty"`
	MinPinLength   *int       `json:"min_pin_length,omitempty"`
	MaxPinLength   *int       `json:"max_pin_length,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PinCode is the public-facing view of a single PIN slot. The actual digits are
// never exposed to clients.
type PinCode struct {
	User        int    `json:"user"`
	UserType    string `json:"user_type,omitempty"`
	UserEnabled bool   `json:"user_enabled"`
	HasCode     bool   `json:"has_code"`
}

// EventKind distinguishes lock activity.
type EventKind string

const (
	EventUnlock EventKind = "unlock"
	EventLock   EventKind = "lock"
)

// Event is a single lock/unlock activity record derived from the lock's
// last_(un)lock_source / last_(un)lock_user fields.
type Event struct {
	Kind   EventKind `json:"kind"`
	Source string    `json:"source"` // keypad | fingerprint | rfid | zigbee | self | unknown
	User   *int      `json:"user,omitempty"`
	At     time.Time `json:"at"`
}
