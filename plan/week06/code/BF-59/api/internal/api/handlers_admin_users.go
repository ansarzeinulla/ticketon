package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

const (
	// CodeCannotSuspendSelf refuses an administrator locking themselves out.
	CodeCannotSuspendSelf = "cannot_suspend_self"
	// CodeOrganizerSuspended is returned at checkout when the event itself is
	// fine but the account behind it is suspended.
	CodeOrganizerSuspended = "organizer_suspended"
)

type userResponse struct {
	User store.User `json:"user"`
}

// handleSuspendUser locks an account (SRS 4.12: "Suspend users or events").
//
// The effect is immediate rather than deferred to token expiry, because
// requireAuth re-reads the account on every authorised request instead of
// trusting the claims in the token. That was already true; what was missing
// was any way for an administrator to set the status.
//
// Suspension is reversible and destroys nothing: the account's events, orders
// and tickets are untouched, so attendees holding valid tickets to a suspended
// organizer's event are not stranded.
func (s *Server) handleSuspendUser(w http.ResponseWriter, r *http.Request) {
	s.setUserStatus(w, r, store.StatusSuspended)
}

// handleUnsuspendUser lifts a suspension. The account goes back to 'active'
// only if it had confirmed its address; otherwise it returns to
// 'pending_verification', because lifting a suspension must not double as
// granting a verification.
func (s *Server) handleUnsuspendUser(w http.ResponseWriter, r *http.Request) {
	s.setUserStatus(w, r, store.StatusActive)
}

func (s *Server) setUserStatus(w http.ResponseWriter, r *http.Request, status string) {
	targetID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The user id must be a UUID.")
		return
	}

	var req suspendRequest
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(w, r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
			return
		}
	}
	if runeLen(req.Reason) > maxReasonLength {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The reason is too long.")
		return
	}

	actorID := mustUserID(r.Context())
	if targetID == actorID && status == store.StatusSuspended {
		httpx.WriteError(w, http.StatusConflict, CodeCannotSuspendSelf,
			"You cannot suspend your own account.")
		return
	}

	var updated store.User
	if status == store.StatusActive {
		updated, err = s.users.Restore(r.Context(), targetID)
	} else {
		updated, err = s.users.SetStatus(r.Context(), targetID, status)
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No user with this id.")
		return
	case errors.Is(err, store.ErrStatusUnchanged):
		message := "This account is already suspended."
		if status == store.StatusActive {
			message = "This account is not suspended."
		}
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict, message)
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	// SRS 7: "Organizer and administrator actions shall be auditable."
	// The entry carries no event id - this action is about a person, not an
	// event - which audit_logs allows, since it holds no foreign keys.
	action, description := "user.suspended", "Account suspended by a platform administrator"
	if status == store.StatusActive {
		action, description = "user.unsuspended", "Account restored by a platform administrator"
	}
	if !blank(req.Reason) {
		description += ": " + req.Reason
	}
	if err := s.audit.Append(r.Context(), store.AuditEntry{
		ActorUserID: &actorID,
		Action:      action,
		EntityType:  "user",
		EntityID:    targetID.String(),
		Description: description,
		Metadata: map[string]any{
			"request_id": httpx.RequestIDFromContext(r.Context()),
			"email":      updated.Email,
		},
	}); err != nil {
		httpx.LogAuditFailure(r.Context(), action, err)
	}

	httpx.WriteJSON(w, http.StatusOK, userResponse{User: updated})
}
