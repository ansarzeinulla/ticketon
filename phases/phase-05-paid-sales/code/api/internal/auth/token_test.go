package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func newService() *TokenService {
	return NewTokenService("test-secret", "biletflow-test", time.Hour)
}

func TestIssueAndParse(t *testing.T) {
	svc := newService()
	userID := uuid.New()

	token, expiresAt, err := svc.Issue(userID, "dana@biletflow.kz", []string{"attendee", "organizer"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("Issue() = %q, want a three-part JWT", token)
	}
	if got := time.Until(expiresAt); got < 59*time.Minute || got > time.Hour {
		t.Errorf("expiry is %s away, want about 1h", got)
	}

	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got, _ := claims.UserID(); got != userID {
		t.Errorf("UserID() = %v, want %v", got, userID)
	}
	if claims.Email != "dana@biletflow.kz" {
		t.Errorf("Email = %q, want dana@biletflow.kz", claims.Email)
	}
	if !claims.HasRole("organizer") || !claims.HasRole("attendee") {
		t.Errorf("Roles = %v, want both attendee and organizer", claims.Roles)
	}
	if claims.HasRole("platform_admin") {
		t.Error("HasRole() reported a role that was never granted")
	}
	if claims.ID == "" {
		t.Error("token has no jti, which a future revocation list would need")
	}
}

func TestIssueGivesEachTokenAUniqueID(t *testing.T) {
	svc := newService()
	userID := uuid.New()

	first, _, err := svc.Issue(userID, "a@b.kz", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	second, _, err := svc.Issue(userID, "a@b.kz", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	firstClaims, _ := svc.Parse(first)
	secondClaims, _ := svc.Parse(second)
	if firstClaims.ID == secondClaims.ID {
		t.Error("two tokens share a jti")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	svc := NewTokenService("test-secret", "biletflow-test", time.Minute)

	// Mint a token an hour in the past, then read it with the real clock.
	svc.SetClock(func() time.Time { return time.Now().Add(-time.Hour) })
	token, _, err := svc.Issue(uuid.New(), "a@b.kz", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	svc.SetClock(time.Now)

	if _, err := svc.Parse(token); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("Parse(expired) error = %v, want ErrTokenExpired", err)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	issuer := NewTokenService("secret-one", "biletflow-test", time.Hour)
	verifier := NewTokenService("secret-two", "biletflow-test", time.Hour)

	token, _, err := issuer.Issue(uuid.New(), "a@b.kz", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if _, err := verifier.Parse(token); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("Parse(other secret) error = %v, want ErrTokenInvalid", err)
	}
}

func TestParseRejectsWrongIssuer(t *testing.T) {
	issuer := NewTokenService("test-secret", "someone-else", time.Hour)
	verifier := newService()

	token, _, err := issuer.Issue(uuid.New(), "a@b.kz", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if _, err := verifier.Parse(token); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("Parse(other issuer) error = %v, want ErrTokenInvalid", err)
	}
}

func TestParseRejectsTamperedPayload(t *testing.T) {
	svc := newService()
	token, _, err := svc.Issue(uuid.New(), "a@b.kz", []string{"attendee"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1] + "x." + parts[2]

	if _, err := svc.Parse(tampered); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("Parse(tampered) error = %v, want ErrTokenInvalid", err)
	}
}

// TestParseRejectsUnsignedToken guards against the classic "alg: none" attack:
// a token whose signature has simply been removed must never be accepted.
func TestParseRejectsUnsignedToken(t *testing.T) {
	svc := newService()

	claims := Claims{
		Email: "attacker@example.com",
		Roles: []string{"platform_admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.NewString(),
			Issuer:    "biletflow-test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build unsigned token: %v", err)
	}

	if _, err := svc.Parse(unsigned); err == nil {
		t.Fatal("Parse() accepted a token signed with alg=none")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	svc := newService()
	for _, token := range []string{"", "abc", "a.b.c", "Bearer x.y.z"} {
		if _, err := svc.Parse(token); err == nil {
			t.Errorf("Parse(%q) accepted a malformed token", token)
		}
	}
}

func TestParseRejectsNonUUIDSubject(t *testing.T) {
	svc := newService()

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "not-a-uuid",
			Issuer:    "biletflow-test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.Parse(token); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("Parse(non-uuid subject) error = %v, want ErrTokenInvalid", err)
	}
}

func TestIssueNeverEmitsNullRoles(t *testing.T) {
	svc := newService()
	token, _, err := svc.Issue(uuid.New(), "a@b.kz", nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.Roles == nil {
		t.Error("Roles decoded as nil; clients expect an array")
	}
}
