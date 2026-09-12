package api

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

const (
	maxSubjectLength = 200
	maxMessageLength = 5000
)

var validSupportCategories = func() map[string]bool {
	set := map[string]bool{}
	for _, category := range store.SupportCategories {
		set[category] = true
	}
	return set
}()

var validSupportStatuses = map[string]bool{
	store.SupportOpen: true, store.SupportInProgress: true,
	store.SupportWaitingForCustomer: true, store.SupportResolved: true,
}

type openCaseRequest struct {
	Category string  `json:"category"`
	Subject  string  `json:"subject"`
	Message  string  `json:"message"`
	OrderID  *string `json:"order_id"`
	TicketID *string `json:"ticket_id"`
	EventID  *string `json:"event_id"`
}

type caseResponse struct {
	Case     store.SupportCase      `json:"case"`
	Messages []store.SupportMessage `json:"messages"`
	// CanReply and CanModerate tell the UI which controls to render, so it does
	// not have to reimplement the authorization rules.
	CanReply    bool `json:"can_reply"`
	CanModerate bool `json:"can_moderate"`
}

type caseListResponse struct {
	Cases []store.SupportCase `json:"cases"`
}

// handleOpenCase creates a support request from an order, ticket or event.
//
// Signing in is required. support_cases.requester_user_id is NOT NULL, and a
// conversation needs an identity to come back to - so a guest buyer registers
// with the address they bought under, and OrderBelongsTo matches them by email.
func (s *Server) handleOpenCase(w http.ResponseWriter, r *http.Request) {
	var req openCaseRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	userID := mustUserID(r.Context())
	errs := fieldErrors{}

	if !validSupportCategories[req.Category] {
		errs.add("category", "Choose one of the listed issue categories.")
	}
	if blank(req.Subject) {
		errs.add("subject", "A short subject is required.")
	} else if msg := validateLine("Subject", req.Subject, 1, maxSubjectLength); msg != "" {
		errs.add("subject", msg)
	}
	if blank(req.Message) {
		errs.add("message", "Describe the problem so the organizer can help.")
	} else if msg := validateMultiline("Message", req.Message, 1, maxMessageLength); msg != "" {
		errs.add("message", msg)
	}

	params := store.OpenCaseParams{
		RequesterID: userID,
		Kind:        store.SupportKindAttendee,
		Category:    req.Category,
		Subject:     req.Subject,
		Body:        req.Message,
	}

	// The context is captured automatically wherever it is available (SRS 4.13),
	// and an order carries its event and the ticket's event with it.
	if req.OrderID != nil {
		orderID, err := uuid.Parse(*req.OrderID)
		if err != nil {
			errs.add("order_id", "The order id must be a UUID.")
		} else {
			owns, ownErr := s.support.OrderBelongsTo(r.Context(), orderID, userID)
			if ownErr != nil {
				httpx.WriteInternalError(w, r, ownErr)
				return
			}
			if !owns {
				httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
					"You can only open a support case about your own order.")
				return
			}
			params.OrderID = &orderID

			result, getErr := s.checkout.GetOrder(r.Context(), orderID)
			if getErr == nil {
				eventID := result.Order.EventID
				params.EventID = &eventID
			}
		}
	}

	if req.TicketID != nil {
		ticketID, err := uuid.Parse(*req.TicketID)
		if err != nil {
			errs.add("ticket_id", "The ticket id must be a UUID.")
		} else {
			detail, detailErr := s.tickets.GetDetail(r.Context(), ticketID)
			if detailErr != nil {
				errs.add("ticket_id", "No ticket with this id.")
			} else {
				params.TicketID = &ticketID
				if params.EventID == nil {
					eventID := detail.EventID
					params.EventID = &eventID
				}
			}
		}
	}

	if params.EventID == nil && req.EventID != nil {
		eventID, err := uuid.Parse(*req.EventID)
		if err != nil {
			errs.add("event_id", "The event id must be a UUID.")
		} else {
			params.EventID = &eventID
		}
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	opened, err := s.support.Open(r.Context(), params)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	messages, err := s.support.Messages(r.Context(), opened.ID, opened.RequesterID, false)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/support/cases/"+opened.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, caseResponse{
		Case: opened, Messages: messages, CanReply: true,
	})
}

// handleListMyCases returns the cases the caller opened.
func (s *Server) handleListMyCases(w http.ResponseWriter, r *http.Request) {
	cases, err := s.support.ListForRequester(r.Context(), mustUserID(r.Context()))
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, caseListResponse{Cases: cases})
}

// handleListEventCases is the organizer's inbox for one event.
func (s *Server) handleListEventCases(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	cases, err := s.support.ListForEvent(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, caseListResponse{Cases: cases})
}

func (s *Server) handleGetCase(w http.ResponseWriter, r *http.Request) {
	supportCase, access, ok := s.loadCase(w, r)
	if !ok {
		return
	}

	messages, err := s.support.Messages(r.Context(), supportCase.ID,
		supportCase.RequesterID, access.staff)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, caseResponse{
		Case: supportCase, Messages: messages,
		CanReply: true, CanModerate: access.staff,
	})
}

type postMessageRequest struct {
	Message  string `json:"message"`
	Internal bool   `json:"internal_note"`
	// Attachment is a file already uploaded through POST /uploads (bonus,
	// SRS 4.13). The client sends back what that endpoint returned.
	Attachment *attachmentRequest `json:"attachment"`
}

// attachmentRequest is a file the client uploaded before posting.
//
// Two steps rather than one multipart message: the upload endpoint already
// exists, already checks the type and size, and already returns a URL. Making
// the chat endpoint a second uploader would duplicate all of that, and would
// mean a large file is re-sent whenever the message text fails validation.
type attachmentRequest struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Bytes    int64  `json:"bytes"`
}

func (s *Server) handlePostCaseMessage(w http.ResponseWriter, r *http.Request) {
	supportCase, access, ok := s.loadCase(w, r)
	if !ok {
		return
	}

	var req postMessageRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	if blank(req.Message) {
		httpx.WriteValidationError(w, fieldErrors{"message": "Write a message first."})
		return
	}
	if msg := validateMultiline("Message", req.Message, 1, maxMessageLength); msg != "" {
		httpx.WriteValidationError(w, fieldErrors{"message": msg})
		return
	}

	// Only staff can leave a note the requester will never see.
	internal := req.Internal && access.staff

	attachment, ok := s.validateAttachment(w, req.Attachment)
	if !ok {
		return
	}

	if _, err := s.support.PostMessage(r.Context(), supportCase.ID,
		mustUserID(r.Context()), req.Message, internal, access.staff, attachment); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	updated, err := s.support.GetByID(r.Context(), supportCase.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	messages, err := s.support.Messages(r.Context(), updated.ID, updated.RequesterID, access.staff)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, caseResponse{
		Case: updated, Messages: messages,
		CanReply: true, CanModerate: access.staff,
	})

	// SRS 4.10: the other side of the conversation is told a reply arrived.
	// Internal notes notify nobody - see sendSupportMessageNotice.
	s.sendSupportMessageNotice(updated, mustUserID(r.Context()),
		s.displayName(r), req.Message, internal)
}

// displayName is who a notification says wrote the message. It falls back to a
// role rather than an empty string, because "replied about your order" with no
// subject reads like a bug.
func (s *Server) displayName(r *http.Request) string {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		return "BiletFlow"
	}
	if userID, err := claims.UserID(); err == nil {
		if user, err := s.users.GetByID(r.Context(), userID); err == nil && user.FullName != "" {
			return user.FullName
		}
	}
	return "BiletFlow"
}

type setCaseStatusRequest struct {
	Status string `json:"status"`
}

// handleSetCaseStatus moves a case between Open, In Progress, Waiting and
// Resolved. Only the organizer or a platform admin may change it - the
// requester cannot close their own case as answered.
func (s *Server) handleSetCaseStatus(w http.ResponseWriter, r *http.Request) {
	supportCase, access, ok := s.loadCase(w, r)
	if !ok {
		return
	}
	if !access.staff {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"Only the organizer can change the status of a case.")
		return
	}

	var req setCaseStatusRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}
	if !validSupportStatuses[req.Status] {
		httpx.WriteValidationError(w, fieldErrors{
			"status": "Status must be open, in_progress, waiting_for_customer or resolved.",
		})
		return
	}

	updated, err := s.support.SetStatus(r.Context(), supportCase.ID, req.Status, mustUserID(r.Context()))
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	if updated.EventID != nil {
		s.appendAudit(r, *updated.EventID, mustUserID(r.Context()),
			"support_case."+req.Status, "support_case", updated.ID.String(),
			"Case "+updated.CaseNumber+" set to "+req.Status)
	}

	// SRS 4.10: the requester is told when their case moves, so somebody
	// waiting on an answer finds out without polling the page.
	if supportCase.Status != updated.Status {
		s.sendSupportStatusChanged(updated, mustUserID(r.Context()))
	}

	messages, err := s.support.Messages(r.Context(), updated.ID, updated.RequesterID, true)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, caseResponse{
		Case: updated, Messages: messages, CanReply: true, CanModerate: true,
	})
}

// caseAccess is what the caller is allowed to do with a case.
type caseAccess struct {
	// staff means the organizer of the case's event, or a platform admin: the
	// side that answers, moderates and can see internal notes.
	staff bool
}

// loadCase resolves {id} and enforces SRS 4.13's access rule: a case is visible
// to the person who opened it, to the organizer of the event it concerns, and
// to platform admins - and to nobody else.
func (s *Server) loadCase(w http.ResponseWriter, r *http.Request) (store.SupportCase, caseAccess, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The case id must be a UUID.")
		return store.SupportCase{}, caseAccess{}, false
	}

	supportCase, err := s.support.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No support case with this id.")
		return store.SupportCase{}, caseAccess{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.SupportCase{}, caseAccess{}, false
	}

	userID := mustUserID(r.Context())
	claims, _ := claimsFromContext(r.Context())

	if claims != nil && claims.HasRole(store.RolePlatformAdmin) {
		return supportCase, caseAccess{staff: true}, true
	}

	if supportCase.EventID != nil {
		event, eventErr := s.events.GetByID(r.Context(), *supportCase.EventID)
		if eventErr == nil && event.OrganizerID == userID {
			return supportCase, caseAccess{staff: true}, true
		}
	}

	if supportCase.RequesterID == userID {
		return supportCase, caseAccess{staff: false}, true
	}

	// Deliberately 404, not 403: confirming a case exists would leak that
	// someone complained about an order this caller has nothing to do with.
	httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No support case with this id.")
	return store.SupportCase{}, caseAccess{}, false
}

type supportCategoriesResponse struct {
	Categories []string `json:"categories"`
}

// handleSupportCategories lets the UI build its category picker from the
// server's list rather than hard-coding one that could drift from the enum.
func (s *Server) handleSupportCategories(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, supportCategoriesResponse{Categories: store.SupportCategories})
}

type assignCaseRequest struct {
	// Email names the person the case goes to. An empty string unassigns it,
	// which is how a case is handed back to the pool.
	Email string `json:"email"`
}

// handleAssignCase puts a case in a named person's hands (SRS 4.13:
// "Authorized staff shall be able to assign a case and change its status").
//
// Assignment used to happen only as a side effect of the first reply, so a
// case could not be handed to a colleague who had not spoken yet - which is
// exactly when handing it over is useful.
func (s *Server) handleAssignCase(w http.ResponseWriter, r *http.Request) {
	supportCase, access, ok := s.loadCase(w, r)
	if !ok {
		return
	}
	if !access.staff {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"Only the organizer can assign a case.")
		return
	}

	var req assignCaseRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	var (
		assignee     *uuid.UUID
		assigneeName string
	)
	if !blank(req.Email) {
		id, name, err := s.support.UserByEmail(r.Context(), strings.TrimSpace(req.Email))
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteValidationError(w, fieldErrors{
				"email": "No BiletFlow account uses this address.",
			})
			return
		}
		if err != nil {
			httpx.WriteInternalError(w, r, err)
			return
		}
		assignee, assigneeName = &id, name
	}

	updated, err := s.support.Assign(r.Context(), supportCase.ID, assignee)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	action, description := "support_case.assigned", "Case "+updated.CaseNumber+" assigned to "+assigneeName
	if assignee == nil {
		action, description = "support_case.unassigned", "Case "+updated.CaseNumber+" unassigned"
	}
	if updated.EventID != nil {
		s.appendAudit(r, *updated.EventID, mustUserID(r.Context()),
			action, "support_case", updated.ID.String(), description)
	}

	// SRS 4.10 lists "support-case assignment" among the notifications. There
	// is nothing to announce when a case is handed back to the pool.
	if assignee != nil {
		s.sendSupportCaseAssigned(updated, assigneeName)
	}

	messages, err := s.support.Messages(r.Context(), updated.ID, updated.RequesterID, true)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, caseResponse{
		Case: updated, Messages: messages, CanReply: true, CanModerate: true,
	})
}

// validateAttachment checks that a claimed attachment is one this server
// actually stored.
//
// The client sends back the URL the upload endpoint gave it, and a client can
// send anything - so the URL is required to be one of ours before it is
// written to a message. Otherwise a support thread would be a way to make
// BiletFlow render a link to any address somebody chose, which is a phishing
// vector with the platform's name on it.
func (s *Server) validateAttachment(
	w http.ResponseWriter, req *attachmentRequest,
) (*store.MessageAttachment, bool) {
	if req == nil || req.URL == "" {
		return nil, true
	}

	prefix := s.cfg.APIBaseURL + uploadURLPrefix
	name := strings.TrimPrefix(req.URL, prefix)

	if name == req.URL || name == "" || strings.ContainsAny(name, `/\`) {
		httpx.WriteValidationError(w, fieldErrors{
			"attachment": "Attach a file by uploading it first.",
		})
		return nil, false
	}

	// And it has to exist: a URL for a file that was never stored would be a
	// broken paperclip in the thread.
	if _, err := os.Stat(filepath.Join(s.cfg.UploadDir, name)); err != nil {
		httpx.WriteValidationError(w, fieldErrors{
			"attachment": "That upload could not be found. Try attaching it again.",
		})
		return nil, false
	}

	if req.Bytes <= 0 {
		httpx.WriteValidationError(w, fieldErrors{"attachment": "That file is empty."})
		return nil, false
	}

	filename := req.Filename
	if filename == "" {
		filename = name
	}
	mime := req.MimeType
	if mime == "" {
		mime = "application/octet-stream"
	}

	return &store.MessageAttachment{
		URL: req.URL, Filename: filename, MimeType: mime, Bytes: req.Bytes,
	}, true
}
