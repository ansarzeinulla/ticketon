// Package auth handles password hashing and access-token issuing.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptMaxPasswordBytes is a hard limit of the algorithm: bcrypt silently
// truncates anything longer, so a longer password is rejected instead.
const bcryptMaxPasswordBytes = 72

// ErrPasswordTooLong is returned when a password exceeds bcrypt's input limit.
var ErrPasswordTooLong = fmt.Errorf("password must not exceed %d bytes", bcryptMaxPasswordBytes)

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

// Hash returns the bcrypt hash of the password.
func (h *Hasher) Hash(password string) (string, error) {
	if len(password) > bcryptMaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	digest, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(digest), nil
}

// Verify reports whether the password matches the stored hash.
func (h *Hasher) Verify(hash, password string) bool {
	if len(password) > bcryptMaxPasswordBytes {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash is a valid bcrypt hash of a random value. Verifying against it when
// no user exists keeps login timing similar for known and unknown addresses, so
// response time cannot be used to enumerate registered accounts.
const dummyHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEe.eS4qZRZ8/8qCPXnZQ4PQdaKQZ0jH4Iu"

// VerifyDummy burns roughly one hash comparison. It always reports false.
func (h *Hasher) VerifyDummy(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
	return err == nil && !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword)
}
