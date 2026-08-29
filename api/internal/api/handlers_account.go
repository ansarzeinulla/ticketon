package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/biletflow/api/internal/email"
	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// CodeTokenInvalid is returned for a token that is unknown, expired or spent.
// The three cases share a code deliberately - see store.ErrTokenInvalid.
const CodeTokenInvalid = "token_invalid"

type emailRequest struct {
	Email string `json:"email"`
}

type consumeTokenRequest struct {
	Token string `json:"token"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// acceptedResponse is the deliberately uninformative answer to both "send me a
// reset link" and "send me a verification link".
type acceptedResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// handleRequestPasswordReset issues a reset token and emails it (SRS 4.1).
//
// The response is the same whether or not the address has an account. Saying
// "no account with this email" would turn this form into a way of testing which
// addresses are registered, which is a privacy leak dressed as helpfulness.
func (s *Server) handleRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req emailRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	address := normalizeEmail(req.Email)
	if msg := validateEmail(address); msg != "" {
		errs := fieldErrors{}
		errs.add("email", msg)
		httpx.WriteValidationError(w, errs)
		return
	}

	issued, err := s.accountTokens.IssueForEmail(
		r.Context(), address, store.TokenPasswordReset, store.PasswordResetTTL)

	switch {
	case errors.Is(err, store.ErrNotFound):
		// Nothing to send, and nothing to say about it.
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	default:
		s.sendPasswordReset(issued)
	}

	httpx.WriteJSON(w, http.StatusAccepted, acceptedResponse{
		Status:  "accepted",
		Message: "If that address has an account, a reset link is on its way.",
	})
}

// handleResetPassword consumes a reset token and sets the new password.
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}
	if blank(req.Token) {
		errs.add("token", "The reset code is required.")
	}
	if msg := validatePassword(req.Password); msg != "" {
		errs.add("password", msg)
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

	user, err := s.accountTokens.ConsumePasswordReset(r.Context(), req.Token, hash)
	if errors.Is(err, store.ErrTokenInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, CodeTokenInvalid,
			"This reset code is invalid, expired or has already been used.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	// No token is returned. Resetting a password proves control of an inbox,
	// not an intent to sign in on this device, and issuing a session here would
	// let a stolen reset link skip the login screen entirely.
	httpx.WriteJSON(w, http.StatusOK, acceptedResponse{
		Status:  "ok",
		Message: "Your password has been changed. Sign in with it now.",
	})
	_ = user
}

// handleRequestEmailVerification re-sends the confirmation for the signed-in
// account (SRS 4.1).
func (s *Server) handleRequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r.Context())

	user, err := s.users.GetByID(r.Context(), userID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if user.EmailVerifiedAt != nil {
		httpx.WriteJSON(w, http.StatusOK, acceptedResponse{
			Status:  "ok",
			Message: "This address is already confirmed.",
		})
		return
	}

	issued, err := s.accountTokens.Issue(
		r.Context(), userID, store.TokenEmailVerification, store.EmailVerificationTTL)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	s.sendEmailVerification(issued)

	httpx.WriteJSON(w, http.StatusAccepted, acceptedResponse{
		Status:  "accepted",
		Message: "Check your inbox for the confirmation link.",
	})
}

type verifiedResponse struct {
	Status string `json:"status"`
	Email  string `json:"email"`
	State  string `json:"account_status"`
}

// handleVerifyEmail consumes a verification token.
//
// It needs no authentication: the token is the proof, and the link is followed
// from an inbox where nobody is signed in.
func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req consumeTokenRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}
	if blank(req.Token) {
		errs := fieldErrors{}
		errs.add("token", "The confirmation code is required.")
		httpx.WriteValidationError(w, errs)
		return
	}

	user, err := s.accountTokens.ConsumeEmailVerification(r.Context(), req.Token)
	if errors.Is(err, store.ErrTokenInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, CodeTokenInvalid,
			"This confirmation code is invalid, expired or has already been used.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, verifiedResponse{
		Status: "verified", Email: user.Email, State: user.Status,
	})
}

// --- the notifications these produce ----------------------------------------

// sendPasswordReset emails a reset token (SRS 4.10).
func (s *Server) sendPasswordReset(issued store.IssuedToken) {
	msg := email.PasswordReset(email.AccountTokenDetails{
		FullName:  issued.FullName,
		Email:     issued.Email,
		Token:     issued.Token,
		Link:      s.cfg.WebBaseURL + "/reset-password?token=" + issued.Token,
		ExpiresIn: humanDuration(store.PasswordResetTTL),
	})
	userID := issued.UserID
	s.dispatch(msg, store.NotificationParams{
		UserID:         &userID,
		RecipientEmail: issued.Email,
	})
}

// sendEmailVerification emails a confirmation token (SRS 4.1).
func (s *Server) sendEmailVerification(issued store.IssuedToken) {
	msg := email.EmailVerification(email.AccountTokenDetails{
		FullName:  issued.FullName,
		Email:     issued.Email,
		Token:     issued.Token,
		Link:      s.cfg.WebBaseURL + "/verify-email?token=" + issued.Token,
		ExpiresIn: humanDuration(store.EmailVerificationTTL),
	})
	userID := issued.UserID
	s.dispatch(msg, store.NotificationParams{
		UserID:         &userID,
		RecipientEmail: issued.Email,
	})
}

// humanDuration renders a TTL the way a person would say it.
func humanDuration(d time.Duration) string {
	switch {
	case d >= 48*time.Hour:
		return itoa(int(d.Hours()/24)) + " days"
	case d >= time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return itoa(hours) + " hours"
	default:
		return itoa(int(d.Minutes())) + " minutes"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
