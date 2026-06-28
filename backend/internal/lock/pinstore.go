package lock

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// encMagic prefixes encrypted PIN-store files so plaintext (legacy) and
// encrypted files can be told apart on load.
var encMagic = []byte("NPLOCKENC1")

// PinStore persists PIN slot metadata to a file. The panel is the source of
// truth for the PIN inventory because the lock firmware never reports its
// stored PINs. Only metadata is kept here — never the PIN digits.
//
// If a key is configured the file is encrypted at rest with AES-256-GCM. If
// path is empty the store is in-memory only (no persistence), so the app still
// runs in development without a configured volume.
type PinStore struct {
	path string
	gcm  cipher.AEAD // nil = plaintext
	mu   sync.Mutex
	pins map[int]PinCode
}

// NewPinStore creates the store and loads any existing records from disk. If
// keyMaterial is non-empty, the store is encrypted at rest (a 32-byte AES key
// is derived from it via SHA-256).
func NewPinStore(path, keyMaterial string) (*PinStore, error) {
	ps := &PinStore{path: path, pins: make(map[int]PinCode)}
	if keyMaterial != "" {
		gcm, err := newGCM(keyMaterial)
		if err != nil {
			return nil, err
		}
		ps.gcm = gcm
	}
	if path == "" {
		return ps, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ps, nil // first run
		}
		return nil, fmt.Errorf("read pin store: %w", err)
	}
	data, err := ps.decode(raw)
	if err != nil {
		return nil, err
	}
	var list []PinCode
	if len(data) > 0 {
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("parse pin store %s: %w", path, err)
		}
	}
	for _, p := range list {
		ps.pins[p.User] = p
	}
	return ps, nil
}

// Persistent reports whether records are written to disk.
func (ps *PinStore) Persistent() bool { return ps.path != "" }

// Encrypted reports whether records are encrypted at rest.
func (ps *PinStore) Encrypted() bool { return ps.gcm != nil }

func newGCM(keyMaterial string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(keyMaterial)) // 32 bytes → AES-256
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("init cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// decode returns the plaintext JSON from a stored file, decrypting if needed.
func (ps *PinStore) decode(raw []byte) ([]byte, error) {
	if !bytes.HasPrefix(raw, encMagic) {
		// Legacy plaintext file; it will be re-encrypted on the next save.
		return raw, nil
	}
	if ps.gcm == nil {
		return nil, errors.New("pin store is encrypted but PIN_STORE_KEY is not set")
	}
	body := raw[len(encMagic):]
	ns := ps.gcm.NonceSize()
	if len(body) < ns {
		return nil, errors.New("pin store file is corrupt (too short)")
	}
	nonce, ct := body[:ns], body[ns:]
	pt, err := ps.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, errors.New("pin store decryption failed (wrong PIN_STORE_KEY?)")
	}
	return pt, nil
}

// encode serializes the on-disk representation, encrypting if a key is set.
func (ps *PinStore) encode(plaintext []byte) ([]byte, error) {
	if ps.gcm == nil {
		return plaintext, nil
	}
	nonce := make([]byte, ps.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := ps.gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(encMagic)+len(nonce)+len(ct))
	out = append(out, encMagic...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// List returns all slots sorted by user index.
func (ps *PinStore) List() []PinCode {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.snapshotLocked()
}

func (ps *PinStore) snapshotLocked() []PinCode {
	out := make([]PinCode, 0, len(ps.pins))
	for _, p := range ps.pins {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].User < out[j].User })
	return out
}

// Upsert inserts or updates a slot, preserving CreatedAt and stamping
// UpdatedAt, then persists.
func (ps *PinStore) Upsert(p PinCode) (PinCode, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	now := time.Now().UTC()
	if existing, ok := ps.pins[p.User]; ok && existing.CreatedAt != nil {
		p.CreatedAt = existing.CreatedAt
	} else {
		p.CreatedAt = &now
	}
	p.UpdatedAt = &now
	ps.pins[p.User] = p
	if err := ps.saveLocked(); err != nil {
		return PinCode{}, err
	}
	return p, nil
}

// SetName updates only the friendly name of an existing slot (panel metadata,
// no lock interaction). Returns false if the slot is unknown.
func (ps *PinStore) SetName(user int, name string) (PinCode, bool, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	p, ok := ps.pins[user]
	if !ok {
		return PinCode{}, false, nil
	}
	now := time.Now().UTC()
	p.Name = name
	p.UpdatedAt = &now
	ps.pins[user] = p
	if err := ps.saveLocked(); err != nil {
		return PinCode{}, false, err
	}
	return p, true, nil
}

// Delete removes a slot and persists.
func (ps *PinStore) Delete(user int) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.pins, user)
	return ps.saveLocked()
}

// saveLocked atomically writes the store to disk (write-temp + rename).
func (ps *PinStore) saveLocked() error {
	if ps.path == "" {
		return nil
	}
	plaintext, err := json.MarshalIndent(ps.snapshotLocked(), "", "  ")
	if err != nil {
		return err
	}
	data, err := ps.encode(plaintext)
	if err != nil {
		return err
	}
	dir := filepath.Dir(ps.path)
	tmp, err := os.CreateTemp(dir, ".pins-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp pin store: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, ps.path)
}
