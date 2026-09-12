package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// registerRequest is the body of POST /api/v1/auth/register.
// Only email and password are required; full_name is derived from the address
// when it is omitted.
type registerRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
	Locale   *string `json:"locale"`
}

// authResponse is returned by both register and login.
type authResponse struct {
	User        store.User `json:"user"`
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	ExpiresAt   time.Time  `json:"expires_at"`
	ExpiresIn   int        `json:"expires_in"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	email := normalizeEmail(req.Email)
	errs := fieldErrors{}

	if msg := validateEmail(email); msg != "" {
		errs.add("email", msg)
	}
	if msg := validatePassword(req.Password); msg != "" {
		errs.add("password", msg)
	}

	fullName := nameFromEmail(email)
	if req.FullName != nil {
		if msg := validateLine("Full name", *req.FullName, minNameLength, maxNameLength); msg != "" {
			errs.add("full_name", msg)
		} else {
			fullName = strings.TrimSpace(*req.FullName)
		}
	}

	locale := "kk"
	if req.Locale != nil {
		if !validLocales[*req.Locale] {
			errs.add("locale", "Locale must be one of kk, ru or en.")
		} else {
			locale = *req.Locale
		}
	}

	if req.Phone != nil && blank(*req.Phone) {
		req.Phone = nil
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	user, err := s.users.Create(r.Context(), store.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		FullName:     fullName,
		Phone:        req.Phone,
		Locale:       locale,
		// Every account starts as an attendee; the organizer role is granted
		// the first time the user creates an event.
		Roles: []string{store.RoleAttendee},
	})
	switch {
	case errors.Is(err, store.ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"An account with this email address already exists.")
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.writeAuthResponse(w, r, http.StatusCreated, user)
}

// loginRequest is the body of POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		errs := fieldErrors{}
		if email == "" {
			errs.add("email", "Email is required.")
		}
		if req.Password == "" {
			errs.add("password", "Password is required.")
		}
		httpx.WriteValidationError(w, errs)
		return
	}

	creds, err := s.users.GetCredentialsByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		// Spend a comparable amount of time on an unknown address so response
		// timing does not reveal which addresses are registered.
		s.hasher.VerifyDummy(req.Password)
		httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials,
			"Email or password is incorrect.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	if !s.hasher.Verify(creds.PasswordHash, req.Password) {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials,
			"Email or password is incorrect.")
		return
	}

	if creds.User.Status == store.StatusSuspended || creds.User.Status == store.StatusDeactivated {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"This account is not allowed to sign in.")
		return
	}

	if err := s.users.TouchLastLogin(r.Context(), creds.User.ID); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	creds.User.LastLoginAt = ptr(s.now().UTC())

	s.writeAuthResponse(w, r, http.StatusOK, creds.User)
}

type meResponse struct {
	User store.User `json:"user"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.users.GetByID(r.Context(), mustUserID(r.Context()))
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, meResponse{User: user})
}

// writeAuthResponse issues a token for the user and writes the shared payload.
func (s *Server) writeAuthResponse(w http.ResponseWriter, r *http.Request, status int, user store.User) {
	token, expiresAt, err := s.tokens.Issue(user.ID, user.Email, user.Roles)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, status, authResponse{
		User:        user,
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.UTC(),
		ExpiresIn:   int(s.tokens.TTL().Seconds()),
	})
}

// ptr returns a pointer to v. Handy for optional response fields.
func ptr[T any](v T) *T { return &v }
