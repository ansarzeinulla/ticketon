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

func TestHashAcceptsLongAndMultibytePasswords(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)

	// Pre-hashing removes bcrypt's 72-byte ceiling: a very long password, and a
	// Cyrillic one that runs well past 72 bytes in UTF-8, both hash and verify.
	long := strings.Repeat("a", 500)
	hash, err := h.Hash(long)
	if err != nil {
		t.Fatalf("Hash(500 chars) error = %v, want nil", err)
	}
	if !h.Verify(hash, long) {
		t.Error("Verify() rejected a correct 500-character password")
	}

	cyrillic := strings.Repeat("құпиясөз", 12) // ~192 bytes, 96 runes
	cyHash, err := h.Hash(cyrillic)
	if err != nil {
		t.Fatalf("Hash(cyrillic) error = %v, want nil", err)
	}
	if !h.Verify(cyHash, cyrillic) {
		t.Error("Verify() rejected a correct multi-byte password")
	}
}

func TestLongPasswordsStayDistinct(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)

	// The property the old 72-byte cap protected: two long passwords that share
	// their first 72 bytes must NOT be interchangeable. Pre-hashing guarantees
	// it because SHA-256 of the two differs.
	base := strings.Repeat("a", 72)
	hash, err := h.Hash(base + "ONE")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if h.Verify(hash, base+"TWO") {
		t.Error("two passwords sharing their first 72 bytes verified against each other")
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
