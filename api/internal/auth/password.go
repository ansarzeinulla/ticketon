// Package auth handles password hashing and access-token issuing.
package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt only reads the first 72 bytes of its input and silently ignores the
// rest, so a naive scheme both caps password length by *bytes* (which is unfair
// to Cyrillic - a Kazakh password of 40 letters is 80 bytes) and makes two
// different long passwords interchangeable.
//
// The fix is to pre-hash: every password is run through SHA-256 and base64
// first, which produces a fixed 44-byte value that always fits inside bcrypt's
// window and differs for differing inputs. The password itself can then be any
// length at all, and its user-facing limit is measured in characters, not
// bytes, like every other field.
func preHash(password string) []byte {
	sum := sha256.Sum256([]byte(password))
	encoded := base64.StdEncoding.EncodeToString(sum[:])
	return []byte(encoded)
}

// Hasher hashes and verifies passwords at a configured cost.
type Hasher struct {
	cost int
}

// NewHasher returns a Hasher; cost 0 falls back to the bcrypt default.
func NewHasher(cost int) *Hasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &Hasher{cost: cost}
}

// Hash returns the bcrypt hash of the pre-hashed password. There is no length
// limit: the pre-hash makes every password 44 bytes before bcrypt sees it.
func (h *Hasher) Hash(password string) (string, error) {
	digest, err := bcrypt.GenerateFromPassword(preHash(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(digest), nil
}

// Verify reports whether the password matches the stored hash.
//
// It tries the pre-hash scheme first, then falls back to comparing the raw
// password - so accounts whose hash predates the pre-hash scheme (the seed
// data, for instance) still sign in. Both a wrong password and a legacy account
// therefore cost two bcrypt comparisons, which VerifyDummy mirrors so response
// time cannot be used to tell a registered address from an unregistered one.
func (h *Hasher) Verify(hash, password string) bool {
	if bcrypt.CompareHashAndPassword([]byte(hash), preHash(password)) == nil {
		return true
	}
	// Legacy hash from before pre-hashing: the raw password went straight into
	// bcrypt, capped at 72 bytes.
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash is a valid bcrypt hash of a random value. Verifying against it when
// no user exists keeps login timing similar for known and unknown addresses, so
// response time cannot be used to enumerate registered accounts.
const dummyHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEe.eS4qZRZ8/8qCPXnZQ4PQdaKQZ0jH4Iu"

// VerifyDummy burns two hash comparisons - the same work Verify does on a wrong
// password or a legacy account - so an unregistered address is
// indistinguishable from a registered one by timing alone. It always reports
// false.
func (h *Hasher) VerifyDummy(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(dummyHash), preHash(password))
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
	return err == nil && !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword)
}
