package lock

import (
	"sort"
	"sync"
	"time"
)

const maxEvents = 100

// Store is a concurrency-safe in-memory cache of the lock's state, PIN slots and
// recent activity. It supports fan-out notifications so the websocket hub can
// push live updates to connected clients.
type Store struct {
	mu     sync.RWMutex
	state  State
	pins   map[int]PinCode
	events []Event // newest last

	// signatures of the last seen lock/unlock so we only emit an Event when
	// the source/user actually changes (Z2M republishes full state often).
	lastUnlockSig string
	lastLockSig   string
	seeded        bool

	subMu sync.Mutex
	subs  map[int]chan struct{}
	nextSub int
}

// NewStore returns an empty store with the lock state defaulted to unknown.
func NewStore() *Store {
	return &Store{
		state: State{LockState: "unknown"},
		pins:  make(map[int]PinCode),
		subs:  make(map[int]chan struct{}),
	}
}

// Snapshot returns a deep-ish copy of the current state, PIN slots (sorted by
// user) and events (newest first) safe to serialize to a client.
func (s *Store) Snapshot() (State, []PinCode, []Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := s.state

	pins := make([]PinCode, 0, len(s.pins))
	for _, p := range s.pins {
		pins = append(pins, p)
	}
	sort.Slice(pins, func(i, j int) bool { return pins[i].User < pins[j].User })

	events := make([]Event, len(s.events))
	for i, e := range s.events {
		events[len(s.events)-1-i] = e // reverse → newest first
	}

	return st, pins, events
}

// State returns just the current cached state.
func (s *Store) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// ApplyState merges a parsed state update into the cache and notifies
// subscribers. The mutate function receives a pointer to the live state copy.
func (s *Store) ApplyState(mutate func(*State)) {
	s.mu.Lock()
	mutate(&s.state)
	s.state.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notify()
}

// SeedPins replaces the live PIN cache (e.g. from the persistent store at
// startup) without emitting per-pin notifications.
func (s *Store) SeedPins(pins []PinCode) {
	s.mu.Lock()
	s.pins = make(map[int]PinCode, len(pins))
	for _, p := range pins {
		s.pins[p.User] = p
	}
	s.mu.Unlock()
	s.notify()
}

// UpsertPin replaces or inserts a single PIN slot.
func (s *Store) UpsertPin(p PinCode) {
	s.mu.Lock()
	s.pins[p.User] = p
	s.mu.Unlock()
	s.notify()
}

// DeletePin removes a PIN slot from the cache.
func (s *Store) DeletePin(user int) {
	s.mu.Lock()
	delete(s.pins, user)
	s.mu.Unlock()
	s.notify()
}

// AddEvent appends an activity event to the ring buffer.
func (s *Store) AddEvent(e Event) {
	s.mu.Lock()
	s.events = append(s.events, e)
	if len(s.events) > maxEvents {
		s.events = s.events[len(s.events)-maxEvents:]
	}
	s.mu.Unlock()
	s.notify()
}

// Subscribe registers a change listener. The returned channel receives an empty
// struct (coalesced) whenever the store changes. Call the returned cancel func
// to unsubscribe.
func (s *Store) Subscribe() (<-chan struct{}, func()) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	id := s.nextSub
	s.nextSub++
	ch := make(chan struct{}, 1)
	s.subs[id] = ch
	return ch, func() {
		s.subMu.Lock()
		delete(s.subs, id)
		close(ch)
		s.subMu.Unlock()
	}
}

func (s *Store) notify() {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for _, ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default: // listener already has a pending notification
		}
	}
}
