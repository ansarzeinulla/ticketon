package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// CodeEventSuspended is returned wherever a suspended event blocks an action.
const CodeEventSuspended = "event_suspended"

type suspendRequest struct {
	Reason string `json:"reason"`
}

// requirePlatformAdmin wraps a handler so only platform staff can reach it.
//
// The role is read from the account on every request, not from the token, so
// revoking it takes effect immediately rather than when the token expires.
func (s *Server) requirePlatformAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFromContext(r.Context())
		if !ok || !claims.HasRole(store.RolePlatformAdmin) {
			httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
				"This action requires a platform administrator.")
			return
		}
		next(w, r)
	})
}

// handleSuspendEvent stops an event selling without destroying it.
//
// SRS 4.12 gives platform administrators the power to suspend a reported or
// fraudulent event. Suspension is deliberately distinct from cancellation:
// cancelling is the organizer calling their own event off, suspending is the
// platform stepping in, and only the platform can undo it.
//
// Tickets already sold stay valid and their holders can still be admitted -
// stranding paying attendees is not the remedy for an organizer's misconduct.
func (s *Server) handleSuspendEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadEventForAdmin(w, r)
	if !ok {
		return
	}

	var req suspendRequest
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(w, r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
			return
		}
	}

	if event.Status == store.EventStatusSuspended {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"This event is already suspended.")
		return
	}
	if event.Status == store.EventStatusCancelled {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"This event is cancelled and is already not selling.")
		return
	}

	updated, err := s.events.SetStatus(r.Context(), event.ID, store.EventStatusSuspended)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	description := "Event suspended by a platform administrator"
	if !blank(req.Reason) {
		description += ": " + req.Reason
	}
	s.appendAudit(r, updated.ID, mustUserID(r.Context()), "event.suspended", "event",
		updated.ID.String(), description)

	httpx.WriteJSON(w, http.StatusOK, eventResponse{Event: updated})
}

// handleUnsuspendEvent lifts a suspension, returning the event to unpublished
// so the organizer must consciously publish it again.
func (s *Server) handleUnsuspendEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadEventForAdmin(w, r)
	if !ok {
		return
	}

	if event.Status != store.EventStatusSuspended {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"This event is not suspended.")
		return
	}

	updated, err := s.events.SetStatus(r.Context(), event.ID, store.EventStatusUnpublished)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, updated.ID, mustUserID(r.Context()), "event.unsuspended", "event",
		updated.ID.String(), "Suspension lifted; the event is unpublished until republished")

	httpx.WriteJSON(w, http.StatusOK, eventResponse{Event: updated})
}

func (s *Server) loadEventForAdmin(w http.ResponseWriter, r *http.Request) (store.Event, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The event id must be a UUID.")
		return store.Event{}, false
	}

	event, err := s.events.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return store.Event{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.Event{}, false
	}
	return event, true
}
