package api

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// moneyPattern accepts a plain decimal amount with at most two places, which is
// what numeric(14,2) stores. Prices arrive as strings so no float ever rounds
// them on the way in.
var moneyPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,11})(\.[0-9]{1,2})?$`)

type ticketTypeRequest struct {
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	PriceKZT      *string `json:"price_kzt"`
	QuantityTotal *int    `json:"quantity_total"`
	MaxPerOrder   *int    `json:"max_per_order"`
	// PriceCategory links this tier to the venue sections it prices, for an
	// assigned-seating event (SRS 4.3.1).
	PriceCategory *string    `json:"price_category"`
	SalesStartAt  *time.Time `json:"sales_start_at"`
	SalesEndAt    *time.Time `json:"sales_end_at"`
	IsHidden      *bool      `json:"is_hidden"`
	DisplayOrder  *int       `json:"display_order"`
}

type ticketTypeResponse struct {
	TicketType store.TicketType `json:"ticket_type"`
}

type ticketTypeListResponse struct {
	TicketTypes []store.TicketType `json:"ticket_types"`
}

func (s *Server) handleCreateTicketType(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	var req ticketTypeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}

	if msg := validateLine("Name", req.Name, minTicketTypeNameLength, maxTicketTypeNameLength); msg != "" {
		errs.add("name", msg)
	}
	checkOptionalMultiline(errs, "description", "Description", req.Description, maxDescriptionLength)
	checkOptionalLine(errs, "price_category", "Price category", req.PriceCategory, maxCategoryLength)

	// A free ticket type is simply one priced at zero (SRS 1.3).
	price := "0"
	if req.PriceKZT != nil {
		if !moneyPattern.MatchString(*req.PriceKZT) {
			errs.add("price_kzt", "Price must be a non-negative amount in KZT, such as 5000 or 5000.00.")
		} else {
			price = *req.PriceKZT
		}
	}

	if req.QuantityTotal == nil {
		errs.add("quantity_total", "Total quantity is required.")
	} else if *req.QuantityTotal < 0 {
		errs.add("quantity_total", "Total quantity must not be negative.")
	}

	maxPerOrder := 10
	if req.MaxPerOrder != nil {
		if *req.MaxPerOrder <= 0 {
			errs.add("max_per_order", "The per-order limit must be greater than zero.")
		} else {
			maxPerOrder = *req.MaxPerOrder
		}
	}

	if req.SalesStartAt != nil && req.SalesEndAt != nil && !req.SalesEndAt.After(*req.SalesStartAt) {
		errs.add("sales_end_at", "Sales must end after they start.")
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	created, err := s.ticketTypes.Create(r.Context(), store.CreateTicketTypeParams{
		EventID:       event.ID,
		Name:          req.Name,
		Description:   req.Description,
		PriceKZT:      price,
		QuantityTotal: *req.QuantityTotal,
		MaxPerOrder:   maxPerOrder,
		PriceCategory: req.PriceCategory,
		SalesStartAt:  req.SalesStartAt,
		SalesEndAt:    req.SalesEndAt,
		IsHidden:      req.IsHidden != nil && *req.IsHidden,
		DisplayOrder:  valueOr(req.DisplayOrder, 0),
	})
	if err != nil {
		s.writeTicketTypeError(w, r, err)
		return
	}

	s.appendAudit(r, event.ID, mustUserID(r.Context()), "ticket_type.created", "ticket_type",
		created.ID.String(), "Created ticket type "+created.Name)

	w.Header().Set("Location", "/api/v1/ticket-types/"+created.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, ticketTypeResponse{TicketType: created})
}

// handleListTicketTypes returns every type for the event, hidden ones included,
// because this is the organizer's own view.
func (s *Server) handleListTicketTypes(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	types, err := s.ticketTypes.ListForEvent(r.Context(), event.ID, false)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ticketTypeListResponse{TicketTypes: types})
}

type updateTicketTypeRequest struct {
	Name          store.Optional[string]    `json:"name"`
	Description   store.Optional[string]    `json:"description"`
	PriceKZT      store.Optional[string]    `json:"price_kzt"`
	QuantityTotal store.Optional[int]       `json:"quantity_total"`
	MaxPerOrder   store.Optional[int]       `json:"max_per_order"`
	SalesStartAt  store.Optional[time.Time] `json:"sales_start_at"`
	SalesEndAt    store.Optional[time.Time] `json:"sales_end_at"`
	IsHidden      store.Optional[bool]      `json:"is_hidden"`
	DisplayOrder  store.Optional[int]       `json:"display_order"`
}

func (s *Server) handleUpdateTicketType(w http.ResponseWriter, r *http.Request) {
	ticketType, _, ok := s.loadOwnedTicketType(w, r)
	if !ok {
		return
	}

	var req updateTicketTypeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}

	if req.Name.Set {
		if !req.Name.Valid || blank(req.Name.Value) {
			errs.add("name", "Name must not be blank.")
		} else if msg := validateLine("Name", req.Name.Value, minTicketTypeNameLength, maxTicketTypeNameLength); msg != "" {
			errs.add("name", msg)
		}
	}
	validateOptionalText(errs, "description", "Description", req.Description, maxDescriptionLength, true)
	if req.PriceKZT.Set {
		if !req.PriceKZT.Valid || !moneyPattern.MatchString(req.PriceKZT.Value) {
			errs.add("price_kzt", "Price must be a non-negative amount in KZT.")
		}
	}
	if req.QuantityTotal.Set {
		switch {
		case !req.QuantityTotal.Valid || req.QuantityTotal.Value < 0:
			errs.add("quantity_total", "Total quantity must not be negative.")
		case req.QuantityTotal.Value < ticketType.QuantitySold+ticketType.QuantityReserved:
			// Caught here so the organizer gets a clear message rather than a
			// constraint violation from ticket_types_inventory_chk.
			errs.add("quantity_total",
				"Total quantity cannot be lower than the number already sold or reserved.")
		}
	}
	if req.MaxPerOrder.Set && (!req.MaxPerOrder.Valid || req.MaxPerOrder.Value <= 0) {
		errs.add("max_per_order", "The per-order limit must be greater than zero.")
	}

	// Validate the resulting window, mixing new values with stored ones.
	start, end := ticketType.SalesStartAt, ticketType.SalesEndAt
	if req.SalesStartAt.Set {
		start = req.SalesStartAt.Ptr()
	}
	if req.SalesEndAt.Set {
		end = req.SalesEndAt.Ptr()
	}
	if start != nil && end != nil && !end.After(*start) {
		errs.add("sales_end_at", "Sales must end after they start.")
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	updated, err := s.ticketTypes.Update(r.Context(), ticketType.ID, store.UpdateTicketTypeParams{
		Name:          req.Name,
		Description:   req.Description,
		PriceKZT:      req.PriceKZT,
		QuantityTotal: req.QuantityTotal,
		MaxPerOrder:   req.MaxPerOrder,
		SalesStartAt:  req.SalesStartAt,
		SalesEndAt:    req.SalesEndAt,
		IsHidden:      req.IsHidden,
		DisplayOrder:  req.DisplayOrder,
	})
	if err != nil {
		s.writeTicketTypeError(w, r, err)
		return
	}

	s.appendAudit(r, updated.EventID, mustUserID(r.Context()), "ticket_type.updated", "ticket_type",
		updated.ID.String(), "Updated ticket type "+updated.Name)

	httpx.WriteJSON(w, http.StatusOK, ticketTypeResponse{TicketType: updated})
}

func (s *Server) handleDeleteTicketType(w http.ResponseWriter, r *http.Request) {
	ticketType, _, ok := s.loadOwnedTicketType(w, r)
	if !ok {
		return
	}

	// Tickets already sold keep their history, so a type with sales is hidden
	// rather than deleted (SRS 4.3: "Hide ticket types without deleting them").
	hasSales, err := s.ticketTypes.HasSales(r.Context(), ticketType.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if hasSales {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"This ticket type has sales and cannot be deleted. Hide it instead.")
		return
	}

	if err := s.ticketTypes.Delete(r.Context(), ticketType.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No ticket type with this id.")
			return
		}
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, ticketType.EventID, mustUserID(r.Context()), "ticket_type.deleted",
		"ticket_type", ticketType.ID.String(), "Deleted ticket type "+ticketType.Name)

	w.WriteHeader(http.StatusNoContent)
}

// loadOwnedTicketType resolves {id} and checks the caller owns its event.
func (s *Server) loadOwnedTicketType(w http.ResponseWriter, r *http.Request) (store.TicketType, store.Event, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The ticket type id must be a UUID.")
		return store.TicketType{}, store.Event{}, false
	}

	ticketType, err := s.ticketTypes.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No ticket type with this id.")
		return store.TicketType{}, store.Event{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.TicketType{}, store.Event{}, false
	}

	event, err := s.events.GetByID(r.Context(), ticketType.EventID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.TicketType{}, store.Event{}, false
	}

	claims, _ := claimsFromContext(r.Context())
	if event.OrganizerID != mustUserID(r.Context()) &&
		!(claims != nil && claims.HasRole(store.RolePlatformAdmin)) {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"Only the organizer of this event can manage its ticket types.")
		return store.TicketType{}, store.Event{}, false
	}

	return ticketType, event, true
}

func (s *Server) writeTicketTypeError(w http.ResponseWriter, r *http.Request, err error) {
	var constraintErr *store.ConstraintError

	switch {
	case errors.Is(err, store.ErrTicketTypeNameTaken):
		httpx.WriteValidationError(w, fieldErrors{
			"name": "This event already has a ticket type with that name.",
		})
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No ticket type with this id.")
	case errors.As(err, &constraintErr):
		httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeValidationFailed,
			"The ticket type violates a database rule: "+constraintErr.Constraint+".")
	default:
		httpx.WriteInternalError(w, r, err)
	}
}

// valueOr dereferences p, or returns fallback when it is nil.
func valueOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
