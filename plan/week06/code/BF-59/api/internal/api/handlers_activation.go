package api

import (
	"errors"
	"net/http"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// CodePaidSalesNotActive is returned when a paid ticket is bought before the
// organizer has activated paid sales (SRS 4.5). CodePaidSalesSuspended is the
// narrower case where a platform admin has stopped an already-active event's
// paid sales (SRS 4.12): the UI says "suspended", not "not yet on sale".
const (
	CodePaidSalesNotActive = "paid_sales_not_active"
	CodePaidSalesSuspended = "paid_sales_suspended"
)

type activationResponse struct {
	Activation store.Activation `json:"activation"`
}

// activationRequest is the checklist submission.
//
// Every step is its own flag rather than one "activate: true", because SRS 4.5
// asks for a checklist: the organizer must be shown, and must confirm, what
// they are agreeing to. A single button that silently ticked all four boxes
// would satisfy the endpoint and not the requirement.
type activationRequest struct {
	// ConfirmIdentity stands in for the identity check a real platform would
	// run against a document or a company registry.
	ConfirmIdentity bool `json:"confirm_identity"`
	// ConfirmPayout stands in for verifying a payout destination.
	ConfirmPayout bool `json:"confirm_payout"`
	// AcceptTerms is the organizer accepting the seller terms.
	AcceptTerms bool `json:"accept_terms"`
	// PayActivationFee records the simulated fee payment.
	PayActivationFee bool `json:"pay_activation_fee"`
}

// handleGetActivation reports the checklist state for the organizer's banner.
func (s *Server) handleGetActivation(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	activation, err := s.activations.ForEvent(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, activationResponse{Activation: activation})

}

// handleAdvanceActivation completes checklist steps and, once every step is
// done, opens paid sales (SRS 4.5).
func (s *Server) handleAdvanceActivation(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	var req activationRequest
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(w, r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
			return
		}
	}

	if !req.ConfirmIdentity && !req.ConfirmPayout && !req.AcceptTerms && !req.PayActivationFee {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"Complete at least one checklist step.")
		return
	}

	activation, err := s.activations.Advance(r.Context(), store.AdvanceParams{
		EventID:     event.ID,
		OrganizerID: mustUserID(r.Context()),
		Steps: store.ActivationSteps{
			Identity: req.ConfirmIdentity,
			Payout:   req.ConfirmPayout,
			Terms:    req.AcceptTerms,
			Fee:      req.PayActivationFee,
		},
	})
	if errors.Is(err, store.ErrPaidSalesSuspended) {
		httpx.WriteError(w, http.StatusForbidden, CodePaidSalesNotActive,
			"Paid sales for this event have been suspended by BiletFlow.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, activationResponse{Activation: activation})
}

type suspendPaidSalesRequest struct {
	Reason string `json:"reason"`
}

// handleSuspendPaidSales stops an event taking money without stopping free
// registration (SRS 4.5: platform administrators shall be able to suspend paid
// sales when fraud or policy violations are suspected).
func (s *Server) handleSuspendPaidSales(w http.ResponseWriter, r *http.Request) {
	s.setPaidSalesSuspended(w, r, true)
}

// handleUnsuspendPaidSales lifts that suspension.
func (s *Server) handleUnsuspendPaidSales(w http.ResponseWriter, r *http.Request) {
	s.setPaidSalesSuspended(w, r, false)
}

func (s *Server) setPaidSalesSuspended(w http.ResponseWriter, r *http.Request, suspended bool) {
	event, ok := s.loadEvent(w, r)
	if !ok {
		return
	}

	var req suspendPaidSalesRequest
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

	activation, err := s.activations.SetSuspended(
		r.Context(), event.ID, mustUserID(r.Context()), suspended, req.Reason)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound,
			"This event has never started paid-sales activation.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, activationResponse{Activation: activation})
}
