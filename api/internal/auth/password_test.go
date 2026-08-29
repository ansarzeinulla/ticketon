package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerify(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)

	hash, err := h.Hash("correct horse battery")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash == "correct horse battery" {
		t.Fatal("Hash() returned the plaintext password")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("Hash() = %q, want a bcrypt digest", hash)
	}
	if !h.Verify(hash, "correct horse battery") {
		t.Error("Verify() rejected the correct password")
	}
	if h.Verify(hash, "wrong password") {
		t.Error("Verify() accepted an incorrect password")
	}
	if h.Verify(hash, "") {
		t.Error("Verify() accepted an empty password")
	}
}

func TestHashIsSaltedPerCall(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)

	first, err := h.Hash("same password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	second, err := h.Hash("same password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if first == second {
		t.Error("two hashes of the same password are identical, so the salt is not random")
	}
	if !h.Verify(first, "same password") || !h.Verify(second, "same password") {
		t.Error("both hashes should verify against the original password")
	}
}

func TestHashRejectsOverlongPassword(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)

	// bcrypt silently truncates past 72 bytes, which would make two different
	// long passwords interchangeable. The hasher must refuse instead.
	if _, err := h.Hash(strings.Repeat("a", 73)); err != ErrPasswordTooLong {
		t.Errorf("Hash(73 bytes) error = %v, want ErrPasswordTooLong", err)
	}
	if _, err := h.Hash(strings.Repeat("a", 72)); err != nil {
		t.Errorf("Hash(72 bytes) error = %v, want nil", err)
	}
}

func TestVerifyRejectsGarbageHash(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)
	if h.Verify("not-a-hash", "anything") {
		t.Error("Verify() accepted a malformed hash")
	}
}

func TestVerifyDummyAlwaysFails(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)
	for _, password := range []string{"", "password", "correct horse battery"} {
		if h.VerifyDummy(password) {
			t.Errorf("VerifyDummy(%q) = true, want false", password)
		}
	}
}
