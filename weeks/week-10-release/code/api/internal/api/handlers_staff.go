package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type assignStaffRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type staffResponse struct {
	Assignment store.StaffAssignment `json:"assignment"`
}

type staffListResponse struct {
	Staff []store.StaffAssignment `json:"staff"`
}

// validStaffRoles matches the staff_role enum.
var validStaffRoles = map[string]bool{
	"event_admin": true, "support_staff": true, "manager": true,
}

// handleAssignStaff gives someone check-in authority over an event (SRS 4.8:
// "Assign event administrators who verify tickets and check attendees in").
func (s *Server) handleAssignStaff(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	var req assignStaffRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}

	email := normalizeEmail(req.Email)
	if msg := validateEmail(email); msg != "" {
		errs.add("email", msg)
	}

	role := req.Role
	if role == "" {
		role = "event_admin"
	}
	if !validStaffRoles[role] {
		errs.add("role", "Role must be event_admin, support_staff or manager.")
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	assignment, err := s.staff.AssignByEmail(r.Context(), event.ID, email, role, mustUserID(r.Context()))
	if errors.Is(err, store.ErrUserNotFound) {
		httpx.WriteValidationError(w, fieldErrors{
			"email": "No BiletFlow account uses that address. Ask them to register first.",
		})
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, event.ID, mustUserID(r.Context()), "staff.assigned", "user",
		assignment.UserID.String(), "Assigned "+assignment.UserEmail+" as "+role)

	httpx.WriteJSON(w, http.StatusCreated, staffResponse{Assignment: assignment})
}

func (s *Server) handleListStaff(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	assignments, err := s.staff.ListForEvent(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, staffListResponse{Staff: assignments})
}

// handleRevokeStaff withdraws an assignment. The row survives so the audit
// trail keeps who was authorised and when.
func (s *Server) handleRevokeStaff(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	assignmentID, err := uuid.Parse(r.PathValue("assignmentId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The assignment id must be a UUID.")
		return
	}

	if err := s.staff.Revoke(r.Context(), assignmentID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound,
			"No active assignment with this id.")
		return
	} else if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, event.ID, mustUserID(r.Context()), "staff.revoked", "staff_assignment",
		assignmentID.String(), "Revoked a staff assignment")

	w.WriteHeader(http.StatusNoContent)
}
