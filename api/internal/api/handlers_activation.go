package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// CodePaidSalesNotActive is returned when a paid ticket is bought before the
// organizer has activated paid sales (SRS 4.5).
const CodePaidSalesNotActive = "paid_sales_not_active"

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

// notifyPayoutRegistered confirms the (simulated) payout destination is on
// file, quoting the masked value so the organizer can check it is the right one.
func (s *Server) notifyPayoutRegistered(r *http.Request, event store.Event) {
	organizer, profile, ok := s.payoutContext(r, event)
	if !ok {
		return
	}

	masked := ""
	if len(profile.PayoutAccounts) > 0 && profile.PayoutAccounts[0].MaskedAccount != nil {
		masked = *profile.PayoutAccounts[0].MaskedAccount
	}
	s.sendPayoutStatus(organizer, event, "destination registered", "", masked,
		"Ticket revenue for this event will be settled here.")
}

// notifyPayoutReady confirms the organizer is cleared to take money.
func (s *Server) notifyPayoutReady(r *http.Request, event store.Event) {
	organizer, profile, ok := s.payoutContext(r, event)
	if !ok {
		return
	}

	masked := ""
	if len(profile.PayoutAccounts) > 0 && profile.PayoutAccounts[0].MaskedAccount != nil {
		masked = *profile.PayoutAccounts[0].MaskedAccount
	}
	s.sendPayoutStatus(organizer, event, "ready", "", masked,
		"Paid sales are open. Settlement is simulated for this release.")
}

func (s *Server) payoutContext(
	r *http.Request, event store.Event,
) (store.User, store.OrganizerProfile, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	organizer, err := s.users.GetByID(ctx, event.OrganizerID)
	if err != nil {
		return store.User{}, store.OrganizerProfile{}, false
	}
	profile, err := s.profiles.Get(ctx, event.OrganizerID)
	if err != nil {
		// A missing profile is not a reason to withhold the notification; it
		// only means there is no masked value to quote.
		return organizer, store.OrganizerProfile{}, true
	}
	return organizer, profile, true
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

	// Read the state before the change, so the notifications below fire once
	// on the transition rather than on every re-submission of the form.
	wasActive := false
	if before, err := s.activations.ForEvent(r.Context(), event.ID); err == nil {
		wasActive = before.IsActive
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

	// SRS 4.10: "Organizer payout status." Registering the payout destination
	// is the first thing that happens to an organizer's money, and completing
	// the checklist is what clears them to take any - so both are worth
	// telling them about, and neither has been until now.
	if req.ConfirmPayout && !wasActive {
		s.notifyPayoutRegistered(r, event)
	}
	if activation.IsActive && !wasActive {
		s.notifyPayoutReady(r, event)
	}
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
