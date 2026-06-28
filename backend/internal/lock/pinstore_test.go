package lock

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPinStorePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pins.json")

	ps, err := NewPinStore(path, "")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := ps.Upsert(PinCode{User: 1, Name: "Alice", UserType: UserTypeUnrestricted, UserEnabled: true, HasCode: true}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := ps.Upsert(PinCode{User: 2, Name: "Bob", HasCode: true}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Reload from disk → records survive.
	ps2, err := NewPinStore(path, "")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	list := ps2.List()
	if len(list) != 2 || list[0].User != 1 || list[0].Name != "Alice" || list[1].Name != "Bob" {
		t.Fatalf("unexpected reloaded list: %+v", list)
	}
	if list[0].CreatedAt == nil || list[0].UpdatedAt == nil {
		t.Errorf("timestamps should be set")
	}

	// Rename preserves CreatedAt.
	created := *list[0].CreatedAt
	renamed, ok, err := ps2.SetName(1, "Alice Smith")
	if err != nil || !ok {
		t.Fatalf("rename: ok=%v err=%v", ok, err)
	}
	if renamed.Name != "Alice Smith" || renamed.CreatedAt == nil || !renamed.CreatedAt.Equal(created) {
		t.Errorf("rename should keep CreatedAt and update name: %+v", renamed)
	}

	// Delete persists.
	if err := ps2.Delete(2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	ps3, _ := NewPinStore(path, "")
	if len(ps3.List()) != 1 {
		t.Errorf("expected 1 record after delete, got %d", len(ps3.List()))
	}
}

func TestPinStoreEncryption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pins.json")
	const key = "super-secret-key"

	ps, err := NewPinStore(path, key)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if !ps.Encrypted() {
		t.Fatal("store should be encrypted")
	}
	if _, err := ps.Upsert(PinCode{User: 1, Name: "Sarah", HasCode: true}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// On-disk bytes must be ciphertext: magic header present, name absent.
	raw, _ := os.ReadFile(path)
	if !bytes.HasPrefix(raw, encMagic) {
		t.Error("encrypted file should start with magic header")
	}
	if bytes.Contains(raw, []byte("Sarah")) {
		t.Error("plaintext name leaked into encrypted file")
	}

	// Correct key round-trips.
	ps2, err := NewPinStore(path, key)
	if err != nil {
		t.Fatalf("reload with key: %v", err)
	}
	if l := ps2.List(); len(l) != 1 || l[0].Name != "Sarah" {
		t.Fatalf("decrypted list wrong: %+v", l)
	}

	// Wrong key fails to load.
	if _, err := NewPinStore(path, "wrong-key"); err == nil {
		t.Error("loading with wrong key should fail")
	}
	// Missing key on an encrypted file fails.
	if _, err := NewPinStore(path, ""); err == nil {
		t.Error("loading encrypted file without key should fail")
	}
}

func TestPinStorePlaintextMigratesToEncrypted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pins.json")

	// Write a plaintext store first (no key).
	plain, _ := NewPinStore(path, "")
	if _, err := plain.Upsert(PinCode{User: 3, Name: "Guest", HasCode: true}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Reopen WITH a key: legacy plaintext loads fine...
	enc, err := NewPinStore(path, "k")
	if err != nil {
		t.Fatalf("reopen with key: %v", err)
	}
	if len(enc.List()) != 1 {
		t.Fatal("should read legacy plaintext records")
	}
	// ...and the next write encrypts the file.
	if _, err := enc.Upsert(PinCode{User: 4, Name: "Plumber", HasCode: true}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	raw, _ := os.ReadFile(path)
	if !bytes.HasPrefix(raw, encMagic) {
		t.Error("file should be encrypted after a save with a key set")
	}
}

func TestPinStoreInMemory(t *testing.T) {
	ps, err := NewPinStore("", "")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if ps.Persistent() {
		t.Error("empty path should be non-persistent")
	}
	if _, err := ps.Upsert(PinCode{User: 0, Name: "Admin", HasCode: true}); err != nil {
		t.Fatalf("upsert should succeed in-memory: %v", err)
	}
	if len(ps.List()) != 1 {
		t.Error("in-memory upsert not stored")
	}
}
