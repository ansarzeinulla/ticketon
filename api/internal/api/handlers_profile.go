package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// CodeWrongPassword refuses a password change whose current password is wrong.
const CodeWrongPassword = "wrong_password"

type profileResponse struct {
	Profile store.OrganizerProfile `json:"profile"`
}

// handleGetProfile returns the caller's organizer profile (SRS 4.1).
//
// An organizer who has never filled one in gets an empty profile rather than a
// 404: the dashboard wants a form to render, and "you have no profile yet" and
// "your profile is blank" are the same thing to the person filling it in.
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r.Context())

	profile, err := s.profiles.Get(r.Context(), userID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteJSON(w, http.StatusOK, profileResponse{Profile: store.OrganizerProfile{
			UserID:         userID,
			PayoutAccounts: []store.PayoutAccount{},
		}})
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, profileResponse{Profile: profile})
}

// profilePatch mirrors the PATCH body. store.Optional records presence and
// nullness, which is what makes "absent" distinguishable from "explicitly
// null" - the same tri-state the event PATCH uses.
type profilePatch struct {
	DisplayName  store.Optional[string] `json:"display_name"`
	LegalName    store.Optional[string] `json:"legal_name"`
	ContactEmail store.Optional[string] `json:"contact_email"`
	ContactPhone store.Optional[string] `json:"contact_phone"`
	Description  store.Optional[string] `json:"description"`
	WebsiteURL   store.Optional[string] `json:"website_url"`
}

// handleUpdateProfile creates or updates the caller's organizer profile.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var patch profilePatch
	if err := httpx.DecodeJSON(w, r, &patch); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}
	update := store.ProfileUpdate{
		DisplayName:  patch.DisplayName,
		LegalName:    patch.LegalName,
		ContactEmail: patch.ContactEmail,
		ContactPhone: patch.ContactPhone,
		Description:  patch.Description,
		WebsiteURL:   patch.WebsiteURL,
	}

	if update.DisplayName.Set && update.DisplayName.Valid {
		if blank(update.DisplayName.Value) {
			// organizer_display_name_not_blank_chk would reject it anyway; a
			// named field error is more use than a constraint violation.
			errs.add("display_name", "The display name cannot be blank.")
		}
		if len(update.DisplayName.Value) > 200 {
			errs.add("display_name", "Keep the display name under 200 characters.")
		}
	}

	// contact_email is a citext column with no shape constraint, so an obvious
	// non-address is caught here rather than stored and later bounced.
	if update.ContactEmail.Set && update.ContactEmail.Valid &&
		!blank(update.ContactEmail.Value) && !looksLikeEmail(update.ContactEmail.Value) {
		errs.add("contact_email", "Enter a valid email address.")
	}
	if update.WebsiteURL.Set && update.WebsiteURL.Valid && !blank(update.WebsiteURL.Value) &&
		!strings.HasPrefix(update.WebsiteURL.Value, "http://") &&
		!strings.HasPrefix(update.WebsiteURL.Value, "https://") {
		errs.add("website_url", "The website must start with http:// or https://.")
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	profile, err := s.profiles.Upsert(r.Context(), mustUserID(r.Context()), update)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, profileResponse{Profile: profile})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// handleChangePassword changes the signed-in user's password (SRS 4.1).
//
// The current password is required even though the caller is already
// authenticated: a token left behind on a shared machine should not be enough
// to lock its owner out of their own account.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}
	if req.CurrentPassword == "" {
		errs.add("current_password", "Enter your current password.")
	}
	if msg := validatePassword(req.NewPassword); msg != "" {
		errs.add("new_password", msg)
	}
	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	userID := mustUserID(r.Context())
	user, err := s.users.GetByID(r.Context(), userID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	creds, err := s.users.GetCredentialsByEmail(r.Context(), user.Email)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if !s.hasher.Verify(creds.PasswordHash, req.CurrentPassword) {
		httpx.WriteError(w, http.StatusForbidden, CodeWrongPassword,
			"That is not your current password.")
		return
	}

	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if err := s.profiles.ChangePassword(r.Context(), userID, hash); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	// The existing token stays valid on purpose: SRS does not ask for session
	// revocation, and the schema carries no revocation list to do it honestly.
	// What does happen is that any outstanding password-reset link is closed,
	// which the store does in the same transaction as the write.
	w.WriteHeader(http.StatusNoContent)
}

// looksLikeEmail is the same shape check the users table applies, so a contact
// address is held to the account address's standard.
func looksLikeEmail(v string) bool {
	at := strings.IndexByte(v, '@')
	if at <= 0 || at == len(v)-1 || strings.ContainsAny(v, " \t") {
		return false
	}
	domain := v[at+1:]
	dot := strings.IndexByte(domain, '.')
	return dot > 0 && dot < len(domain)-1 && !strings.Contains(domain, "@")
}
