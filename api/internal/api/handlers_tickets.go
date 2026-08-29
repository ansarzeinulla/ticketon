package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
	"github.com/biletflow/api/internal/ticketpdf"
)

// handleTicketPDF returns a print-ready A4 PDF for one ticket.
//
// Access is by ticket UUID, the same capability model as the order
// confirmation: a guest checkout has no account to authenticate against, so the
// unguessable id in the link is what grants access. Anyone holding that link can
// print the ticket, which is exactly the property a ticket needs - and why the
// id is a v4 UUID rather than anything sequential.
func (s *Server) handleTicketPDF(w http.ResponseWriter, r *http.Request) {
	detail, ok := s.loadTicket(w, r)
	if !ok {
		return
	}

	pdf, err := ticketpdf.Render(ticketpdf.Ticket{
		EventTitle:     detail.EventTitle,
		StartsAt:       detail.StartsAt,
		EndsAt:         detail.EndsAt,
		Timezone:       detail.Timezone,
		VenueName:      detail.VenueName,
		VenueAddress:   detail.VenueAddress,
		AttendeeName:   detail.AttendeeName,
		AttendeeEmail:  detail.AttendeeEmail,
		TicketTypeName: detail.TicketTypeName,
		TicketCode:     detail.TicketCode,
		TicketID:       detail.ID.String(),
		OrderNumber:    detail.OrderNumber,
		SeatSection:    detail.SeatSection,
		SeatRow:        detail.SeatRow,
		SeatNumber:     detail.SeatNumber,
		QRToken:        detail.QRToken,
	})
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	filename := "biletflow-" + safeFilename(detail.TicketCode) + ".pdf"

	w.Header().Set("Content-Type", "application/pdf")
	// attachment makes the browser save it rather than render it inline, which
	// is what the "Download PDF ticket" button relies on across origins.
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	// A ticket can be cancelled or refunded, so it must never be cached.
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

// handleTicketQR returns just the admission QR as a PNG, for embedding a
// preview on the order page. It encodes the identical string the PDF does,
// because both call ticketpdf.QRPNG with the same token.
func (s *Server) handleTicketQR(w http.ResponseWriter, r *http.Request) {
	detail, ok := s.loadTicket(w, r)
	if !ok {
		return
	}

	png, err := ticketpdf.QRPNG(detail.QRToken)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(png)))
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

type ticketResponse struct {
	Ticket ticketView `json:"ticket"`
}

// ticketView is the JSON shape of a ticket, with the links the UI needs.
type ticketView struct {
	ID             string `json:"id"`
	TicketCode     string `json:"ticket_code"`
	QRToken        string `json:"qr_token"`
	Status         string `json:"status"`
	TicketTypeName string `json:"ticket_type_name"`
	AttendeeName   string `json:"attendee_name"`
	EventTitle     string `json:"event_title"`
	PDFURL         string `json:"pdf_url"`
	QRURL          string `json:"qr_url"`
}

func (s *Server) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	detail, ok := s.loadTicket(w, r)
	if !ok {
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ticketResponse{Ticket: ticketView{
		ID:             detail.ID.String(),
		TicketCode:     detail.TicketCode,
		QRToken:        detail.QRToken,
		Status:         detail.Status,
		TicketTypeName: detail.TicketTypeName,
		AttendeeName:   detail.AttendeeName,
		EventTitle:     detail.EventTitle,
		PDFURL:         "/api/v1/tickets/" + detail.ID.String() + "/pdf",
		QRURL:          "/api/v1/tickets/" + detail.ID.String() + "/qr.png",
	}})
}

// loadTicket resolves the {id} path value and fetches the ticket.
func (s *Server) loadTicket(w http.ResponseWriter, r *http.Request) (store.TicketDetail, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The ticket id must be a UUID.")
		return store.TicketDetail{}, false
	}

	detail, err := s.tickets.GetDetail(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No ticket with this id.")
		return store.TicketDetail{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.TicketDetail{}, false
	}

	return detail, true
}

// safeFilename strips anything that would need quoting in a Content-Disposition
// header or would be awkward on a filesystem.
func safeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	trimmed := strings.Trim(b.String(), "-")
	if trimmed == "" {
		return "ticket"
	}
	return trimmed
}
