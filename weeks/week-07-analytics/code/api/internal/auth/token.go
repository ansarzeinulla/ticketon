package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Errors returned when an access token cannot be trusted.
var (
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token has expired")
)

// Claims is the payload carried by a BiletFlow access token.
type Claims struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// UserID returns the authenticated user's id.
func (c Claims) UserID() (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}

// HasRole reports whether the token carries the given role.
func (c Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// TokenService issues and validates HS256 access tokens.
//
// Tokens are stateless: there is no server-side session table, so a token is
// valid until it expires. Revocation and refresh tokens are deliberately out of
// scope for Phase 2 and need a storage table when they are added.
type TokenService struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time // injectable so tests can produce expired tokens
}

// NewTokenService builds a TokenService.
func NewTokenService(secret, issuer string, ttl time.Duration) *TokenService {
	return &TokenService{secret: []byte(secret), issuer: issuer, ttl: ttl, now: time.Now}
}

// SetClock replaces the time source. Intended for tests only.
func (s *TokenService) SetClock(now func() time.Time) { s.now = now }

// TTL is the lifetime of a freshly issued token.
func (s *TokenService) TTL() time.Duration { return s.ttl }

// Issue signs an access token for the user and returns it with its expiry.
func (s *TokenService) Issue(userID uuid.UUID, email string, roles []string) (string, time.Time, error) {
	now := s.now()
	expiresAt := now.Add(s.ttl)

	if roles == nil {
		roles = []string{}
	}

	claims := Claims{
		Email: email,
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    s.issuer,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse validates a token's signature, issuer and expiry, and returns its claims.
func (s *TokenService) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		// Pin the algorithm: without this an attacker could present a token
		// signed with "none" or with an asymmetric algorithm.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return s.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	if _, err := claims.UserID(); err != nil {
		return nil, fmt.Errorf("%w: subject is not a uuid", ErrTokenInvalid)
	}
	return claims, nil
}
