package lock

import "testing"

func TestParseCoreState(t *testing.T) {
	payload := []byte(`{
		"state":"LOCK",
		"battery":82,
		"voltage":5.9,
		"linkquality":120,
		"sound_volume":"low_volume",
		"auto_relock":"ON",
		"auto_relock_time":7,
		"max_pin_users":20,
		"min_pin_length":4,
		"max_pin_length":8
	}`)
	p, err := Parse(payload)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.LockState == nil || *p.LockState != "locked" {
		t.Errorf("lock state = %v, want locked", p.LockState)
	}
	if p.Battery == nil || *p.Battery != 82 {
		t.Errorf("battery = %v, want 82", p.Battery)
	}
	if p.AutoRelock == nil || *p.AutoRelock != true {
		t.Errorf("auto_relock = %v, want true", p.AutoRelock)
	}
	if p.MaxPinUsers == nil || *p.MaxPinUsers != 20 {
		t.Errorf("max_pin_users = %v, want 20", p.MaxPinUsers)
	}
}

func TestParsePinObject(t *testing.T) {
	payload := []byte(`{"pin_code":{"user":3,"user_type":"unrestricted","user_enabled":true,"pin_code":"1234"}}`)
	p, err := Parse(payload)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(p.Pins) != 1 {
		t.Fatalf("got %d pins, want 1", len(p.Pins))
	}
	pin := p.Pins[0]
	if pin.User != 3 || !pin.HasCode || pin.Delete || !pin.UserEnabled {
		t.Errorf("unexpected pin: %+v", pin)
	}
}

func TestParsePinArrayAndDelete(t *testing.T) {
	payload := []byte(`{"pin_code":[{"user":1,"pin_code":"5555"},{"user":2,"pin_code":null}]}`)
	p, err := Parse(payload)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(p.Pins) != 2 {
		t.Fatalf("got %d pins, want 2", len(p.Pins))
	}
	if !p.Pins[0].HasCode {
		t.Errorf("pin 1 should have a code")
	}
	if !p.Pins[1].Delete {
		t.Errorf("pin 2 should be a delete (null code)")
	}
}

func TestIngestDerivesEvents(t *testing.T) {
	s := NewStore()
	src := "keypad"
	user := 1
	// First ingest seeds the baseline (no event emitted).
	s.Ingest(Parsed{LastUnlockSource: &src, LastUnlockUser: &user})
	_, _, events := s.Snapshot()
	if len(events) != 0 {
		t.Fatalf("seed should not emit events, got %d", len(events))
	}
	// A new unlock by a different user should emit one event.
	user2 := 2
	s.Ingest(Parsed{LastUnlockSource: &src, LastUnlockUser: &user2})
	_, _, events = s.Snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventUnlock || events[0].User == nil || *events[0].User != 2 {
		t.Errorf("unexpected event: %+v", events[0])
	}
}

func TestValidatePin(t *testing.T) {
	minL, maxL := 4, 8
	st := State{MinPinLength: &minL, MaxPinLength: &maxL}
	if err := validatePin("123", st); err == nil {
		t.Error("too-short pin should fail")
	}
	if err := validatePin("12ab", st); err == nil {
		t.Error("non-numeric pin should fail")
	}
	if err := validatePin("1234", st); err != nil {
		t.Errorf("valid pin should pass: %v", err)
	}
}
