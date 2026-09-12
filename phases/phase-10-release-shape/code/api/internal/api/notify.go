package api

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/email"
	"github.com/biletflow/api/internal/store"
)

// notifyTimeout bounds the outbox writes that happen after a response has
// already gone out. Nothing is waiting on them, so they must not hang forever
// on a database that has gone away.
const notifyTimeout = 10 * time.Second

// sendOrderConfirmation notifies the buyer that their tickets are ready
// (SRS 4.10: registration or purchase confirmation, and ticket delivery).
//
// It is called after the checkout response has been written. A notification
// that fails must never fail the purchase: the attendee has paid, the tickets
// exist, and the confirmation is a convenience on top of that. Failures are
// logged and recorded against the outbox row instead of surfacing.
func (s *Server) sendOrderConfirmation(result store.CheckoutResult, event store.Event) {
	tickets := make([]email.TicketLine, 0, len(result.Tickets))
	for _, t := range result.Tickets {
		tickets = append(tickets, email.TicketLine{
			TicketCode: t.TicketCode,
			TypeName:   t.TicketTypeName,
			PDFURL:     s.cfg.APIBaseURL + "/api/v1/tickets/" + t.ID.String() + "/pdf",
		})
	}

	msg := email.OrderConfirmation(email.OrderDetails{
		BuyerName:   result.Order.BuyerName,
		BuyerEmail:  result.Order.BuyerEmail,
		OrderNumber: result.Order.OrderNumber,
		EventTitle:  event.Title,
		EventVenue:  venueLine(event),
		EventWhen:   eventWhen(event),
		TotalKZT:    result.Order.TotalKZT,
		Tickets:     tickets,
		OrderURL:    s.cfg.WebBaseURL + "/orders/" + result.Order.ID.String(),
	})

	s.dispatch(msg, store.NotificationParams{
		UserID:         result.Order.BuyerUserID,
		RecipientEmail: result.Order.BuyerEmail,
		EventID:        &event.ID,
		OrderID:        &result.Order.ID,
	})
}

// sendRefundConfirmation tells the buyer their money is coming back and their
// tickets are void (SRS 4.10: refund completion).
func (s *Server) sendRefundConfirmation(result store.RefundResult, reason string) {
	msg := email.RefundCompleted(email.RefundDetails{
		BuyerName:   result.BuyerName,
		BuyerEmail:  result.BuyerEmail,
		OrderNumber: result.Order.OrderNumber,
		EventTitle:  result.EventTitle,
		AmountKZT:   result.Refund.AmountKZT,
		TicketCount: result.VoidedTickets,
		Reason:      reason,
	})

	eventID := result.Order.EventID
	s.dispatch(msg, store.NotificationParams{
		UserID:         result.Order.BuyerUserID,
		RecipientEmail: result.BuyerEmail,
		EventID:        &eventID,
		OrderID:        &result.Order.ID,
	})
}

// sendCancellationConfirmation tells the attendee their free place is gone
// (SRS 4.9, and SRS 4.10's notification list).
//
// Deliberately not sendRefundConfirmation with a zero amount: an email saying
// "0.00 KZT has been refunded" raises a question that has no answer.
func (s *Server) sendCancellationConfirmation(result store.CancelResult, reason string) {
	msg := email.RegistrationCancelled(email.CancellationDetails{
		BuyerName:   result.BuyerName,
		BuyerEmail:  result.BuyerEmail,
		OrderNumber: result.Order.OrderNumber,
		EventTitle:  result.EventTitle,
		TicketCount: result.CancelledTickets,
		Reason:      reason,
	})

	eventID := result.Order.EventID
	s.dispatch(msg, store.NotificationParams{
		UserID:         result.Order.BuyerUserID,
		RecipientEmail: result.BuyerEmail,
		EventID:        &eventID,
		OrderID:        &result.Order.ID,
	})
}

// sendPaymentFailure tells a buyer their payment did not go through and that
// no tickets were issued (SRS 4.10, "payment failure").
//
// There is no order row to point at: the decline happens before the order is
// written, precisely so a failed transaction leaves nothing behind that could
// be mistaken for a purchase (SRS 4.6).
func (s *Server) sendPaymentFailure(
	event store.Event, buyerName, buyerEmail, amountKZT, reason string,
) {
	msg := email.PaymentFailed(email.PaymentFailureDetails{
		BuyerName:   buyerName,
		BuyerEmail:  buyerEmail,
		OrderNumber: "-",
		EventTitle:  event.Title,
		AmountKZT:   amountKZT,
		Reason:      reason,
		RetryURL:    s.cfg.WebBaseURL + "/events/" + event.Slug,
	})

	eventID := event.ID
	s.dispatch(msg, store.NotificationParams{
		RecipientEmail: buyerEmail,
		EventID:        &eventID,
	})
}

// sendPayoutStatus tells an organizer where their (simulated) money is
// (SRS 4.10, "organizer payout status").
func (s *Server) sendPayoutStatus(
	organizer store.User, event store.Event, status, amountKZT, masked, note string,
) {
	msg := email.PayoutStatus(email.PayoutStatusDetails{
		OrganizerName:  organizer.FullName,
		OrganizerEmail: organizer.Email,
		EventTitle:     event.Title,
		Status:         status,
		AmountKZT:      amountKZT,
		MaskedAccount:  masked,
		Note:           note,
	})

	eventID := event.ID
	organizerID := organizer.ID
	s.dispatch(msg, store.NotificationParams{
		UserID:         &organizerID,
		RecipientEmail: organizer.Email,
		EventID:        &eventID,
	})
}

// sendEventUpdate tells every ticket holder that something they planned around
// has moved (SRS 4.10, "event updates").
//
// Fan-out is one message per distinct buyer address, not one per ticket: a
// person who bought four seats gets told once.
func (s *Server) sendEventUpdate(event store.Event, changes []string) {
	if len(changes) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	holders, err := s.events.TicketHolders(ctx, event.ID)
	if err != nil {
		log.Printf("notification: could not list ticket holders for %s: %v", event.ID, err)
		return
	}

	for _, h := range holders {
		msg := email.EventUpdated(email.EventUpdateDetails{
			AttendeeName:  h.BuyerName,
			AttendeeEmail: h.BuyerEmail,
			EventTitle:    event.Title,
			Changes:       changes,
			EventURL:      s.cfg.WebBaseURL + "/events/" + event.Slug,
		})
		eventID := event.ID
		orderID := h.OrderID
		s.dispatch(msg, store.NotificationParams{
			UserID:         h.UserID,
			RecipientEmail: h.BuyerEmail,
			EventID:        &eventID,
			OrderID:        &orderID,
		})
	}
}

// sendSupportCaseAssigned tells the requester who picked their case up
// (SRS 4.10, 4.13: "support-case assignment").
func (s *Server) sendSupportCaseAssigned(supportCase store.SupportCase, assigneeName string) {
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	requester, err := s.support.RequesterContact(ctx, supportCase.ID)
	if err != nil {
		return
	}
	// Assigning a case to its own requester is not news to them.
	if supportCase.AssignedToID != nil && requester.UserID != nil &&
		*supportCase.AssignedToID == *requester.UserID {
		return
	}

	msg := email.SupportCaseAssigned(email.SupportCaseDetails{
		RecipientName:  requester.FullName,
		RecipientEmail: requester.Email,
		CaseNumber:     supportCase.CaseNumber,
		Subject:        supportCase.Subject,
		AssigneeName:   assigneeName,
		CaseURL:        s.cfg.WebBaseURL + "/support/" + supportCase.ID.String(),
	})

	s.dispatch(msg, store.NotificationParams{
		UserID:         requester.UserID,
		RecipientEmail: requester.Email,
		EventID:        supportCase.EventID,
		SupportCaseID:  &supportCase.ID,
	})
}

// sendSupportStatusChanged tells the requester their case moved
// (SRS 4.10, 4.13: "support-case status changes").
//
// Only the requester is told. Staff changing the status already know, and the
// point of the notification is that somebody waiting on an answer finds out
// without having to poll the page.
func (s *Server) sendSupportStatusChanged(supportCase store.SupportCase, actorID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	requester, err := s.support.RequesterContact(ctx, supportCase.ID)
	if err != nil {
		return
	}
	if requester.UserID != nil && *requester.UserID == actorID {
		return
	}

	msg := email.SupportCaseStatusChanged(email.SupportCaseDetails{
		RecipientName:  requester.FullName,
		RecipientEmail: requester.Email,
		CaseNumber:     supportCase.CaseNumber,
		Subject:        supportCase.Subject,
		Status:         supportCase.Status,
		CaseURL:        s.cfg.WebBaseURL + "/support/" + supportCase.ID.String(),
	})

	s.dispatch(msg, store.NotificationParams{
		UserID:         requester.UserID,
		RecipientEmail: requester.Email,
		EventID:        supportCase.EventID,
		SupportCaseID:  &supportCase.ID,
	})
}

// dispatch records the message in the outbox and hands it to the mailer.
//
// The row is written synchronously so a test - or an operator - can see that a
// notification was raised, even if the send itself is still in flight. Only
// the delivery is asynchronous.
func (s *Server) dispatch(msg email.Message, params store.NotificationParams) {
	params.Type = msg.Type
	params.Subject = msg.Subject
	params.Body = msg.Body

	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	var id uuid.UUID
	if s.notifications != nil {
		var err error
		id, err = s.notifications.Queue(ctx, params)
		if err != nil {
			// Losing the outbox row is not a reason to lose the email.
			log.Printf("notification: could not record %s for %s: %v",
				msg.Type, msg.To, err)
		}
	}

	msg.Ref = id.String()
	s.mailer.SendAsync(msg)
}

// markNotification is the mailer's completion callback. It runs on the
// sending goroutine, after the request that triggered it is long gone.
//
// The outbox row is identified by the message's Ref rather than by a field on
// the Server: several requests can be in flight at once, and a shared field
// would be a race that silently marked the wrong row.
func (s *Server) markNotification(msg email.Message, err error) {
	if s.notifications == nil {
		return
	}
	id, parseErr := uuid.Parse(msg.Ref)
	if parseErr != nil || id == uuid.Nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	if err != nil {
		log.Printf("notification %s: delivery failed: %v", id, err)
		_ = s.notifications.MarkFailed(ctx, id)
		return
	}
	_ = s.notifications.MarkSent(ctx, id)
}

// venueLine renders the venue for an email, joining name and address only when
// both are present.
func venueLine(event store.Event) string {
	switch {
	case event.VenueName != nil && event.VenueAddress != nil:
		return *event.VenueName + ", " + *event.VenueAddress
	case event.VenueName != nil:
		return *event.VenueName
	case event.VenueAddress != nil:
		return *event.VenueAddress
	}
	return ""
}

// eventWhen renders the start time in the event's own timezone, which is the
// one printed on the ticket and the one the attendee will turn up in.
func eventWhen(event store.Event) string {
	location, err := time.LoadLocation(event.Timezone)
	if err != nil {
		location = time.UTC
	}
	return event.StartsAt.In(location).Format("Mon 2 Jan 2006, 15:04 (MST)")
}

// localTime renders any instant in an event's own timezone, for the
// before/after lines on an update notice.
func localTime(event store.Event, at time.Time) string {
	location, err := time.LoadLocation(event.Timezone)
	if err != nil {
		location = time.UTC
	}
	return at.In(location).Format("Mon 2 Jan 2006, 15:04 (MST)")
}

// sendEventCancelled tells every ticket holder their event is off (SRS 4.10).
//
// One message per order rather than per ticket: somebody who bought four seats
// wants one email, not four. Failures are logged and dropped - a cancellation
// that has already happened must not be undone because an inbox was
// unreachable.
func (s *Server) sendEventCancelled(event store.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	holders, err := s.events.TicketHolders(ctx, event.ID)
	if err != nil {
		log.Printf("notification: could not list ticket holders for %s: %v", event.ID, err)
		return
	}

	policy := ""
	if event.RefundPolicy != nil {
		policy = *event.RefundPolicy
	}

	for _, holder := range holders {
		msg := email.EventCancelled(email.EventCancelledDetails{
			AttendeeName: holder.BuyerName,
			Email:        holder.BuyerEmail,
			EventTitle:   event.Title,
			EventWhen:    eventWhen(event),
			OrderNumber:  holder.OrderNumber,
			TicketCount:  holder.TicketCount,
			RefundPolicy: policy,
		})

		eventID, orderID := event.ID, holder.OrderID
		s.dispatch(msg, store.NotificationParams{
			UserID:         holder.UserID,
			RecipientEmail: holder.BuyerEmail,
			EventID:        &eventID,
			OrderID:        &orderID,
		})
	}
}

// sendSupportMessageNotice tells the other side of a conversation that a reply
// has arrived (SRS 4.10, 4.13).
//
// Internal notes are deliberately never notified: they exist so staff can talk
// among themselves, and emailing the attendee about one would defeat the whole
// point of the checkbox.
func (s *Server) sendSupportMessageNotice(
	supportCase store.SupportCase, senderID uuid.UUID, senderName, body string, internal bool,
) {
	if internal {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	recipient, err := s.support.Counterpart(ctx, supportCase.ID, senderID)
	if err != nil {
		// A case with nobody on the other side is not an error worth shouting
		// about; there is simply no one to tell.
		return
	}
	if recipient.UserID != nil && *recipient.UserID == senderID {
		return
	}

	title := "your event"
	if supportCase.EventTitle != nil {
		title = *supportCase.EventTitle
	}

	msg := email.NewSupportMessage(email.SupportMessageDetails{
		RecipientName:  recipient.FullName,
		Email:          recipient.Email,
		EventTitle:     title,
		CaseSubject:    supportCase.Subject,
		SenderName:     senderName,
		MessagePreview: truncateForPreview(body, 400),
		CaseURL:        s.cfg.WebBaseURL + "/orders/" + orderIDForCase(supportCase),
	})

	s.dispatch(msg, store.NotificationParams{
		UserID:         recipient.UserID,
		RecipientEmail: recipient.Email,
		EventID:        supportCase.EventID,
		OrderID:        supportCase.OrderID,
	})
}

// orderIDForCase returns the order a case hangs off, so the link lands on the
// page where the thread is rendered.
func orderIDForCase(supportCase store.SupportCase) string {
	if supportCase.OrderID != nil {
		return supportCase.OrderID.String()
	}
	return ""
}

// truncateForPreview keeps a quoted message to a readable length, cutting on a
// word boundary rather than mid-word.
func truncateForPreview(body string, limit int) string {
	body = strings.TrimSpace(body)
	if len(body) <= limit {
		return body
	}
	cut := body[:limit]
	if space := strings.LastIndexByte(cut, ' '); space > limit/2 {
		cut = cut[:space]
	}
	return cut + "…"
}
