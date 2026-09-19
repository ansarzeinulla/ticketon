// Package api wires the HTTP routes, middleware and handlers together.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/biletflow/api/internal/auth"
	"github.com/biletflow/api/internal/config"
	"github.com/biletflow/api/internal/email"
	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// Server holds the dependencies shared by every handler.
type Server struct {
	cfg           config.Config
	pool          *pgxpool.Pool
	users         *store.UserStore
	accountTokens *store.TokenStore
	mailer        *email.Mailer

	events      *store.EventStore
	ticketTypes *store.TicketTypeStore
	checkout    *store.CheckoutStore
	seating     *store.SeatingStore
	activations *store.ActivationStore
	tickets     *store.TicketStore

	notifications *store.NotificationStore
	guests        *store.GuestStore
	profiles      *store.ProfileStore
	audit         *store.AuditStore
	staff         *store.StaffStore
	checkIns      *store.CheckInStore

	hasher *auth.Hasher
	tokens *auth.TokenService
	now    func() time.Time
}

// New builds a server that prints account emails to stdout.
func New(cfg config.Config, pool *pgxpool.Pool) *Server {
	return NewWithSender(cfg, pool, email.NewConsoleSender(nil))
}

// NewWithSender builds a server with an explicit email sender, so tests can
// capture messages instead of printing them.
func NewWithSender(cfg config.Config, pool *pgxpool.Pool, sender email.Sender) *Server {
	s := &Server{
		cfg:           cfg,
		pool:          pool,
		users:         store.NewUserStore(pool),
		accountTokens: store.NewTokenStore(pool),
		events:        store.NewEventStore(pool),
		ticketTypes:   store.NewTicketTypeStore(pool),
		checkout: store.NewCheckoutStoreWithFees(pool, store.Fees{
			Percent:  cfg.ProcessingFeePercent,
			FixedKZT: cfg.ProcessingFeeFixedKZT,
		}),
		seating:       store.NewSeatingStore(pool),
		activations:   store.NewActivationStore(pool),
		tickets:       store.NewTicketStore(pool),
		notifications: store.NewNotificationStore(pool),
		guests:        store.NewGuestStore(pool),
		profiles:      store.NewProfileStore(pool),
		audit:         store.NewAuditStore(pool),
		staff:         store.NewStaffStore(pool),
		checkIns:      store.NewCheckInStore(pool),
		hasher:        auth.NewHasher(cfg.BcryptCost),
		tokens:        auth.NewTokenService(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL),
		now:           time.Now,
	}

	// The mailer is built last: its completion callback needs the Server that
	// owns the notification store it marks.
	s.mailer = email.NewMailer(sender, s.markNotification)
	return s
}

// StartHoldSweeper releases abandoned baskets on a timer (SRS 4.6).
//
// It returns immediately; the sweep runs until the context is cancelled. The
// opportunistic release inside the hold transaction already guarantees that a
// stale basket never blocks a real sale - this keeps the counters honest for
// everything else, so an event page shows the right number remaining even when
// nobody is shopping.
func (s *Server) StartHoldSweeper(ctx context.Context) {
	sweeper := store.NewSweeper(s.checkout, time.Minute, func(released int, err error) {
		if err != nil {
			slog.Error("hold sweeper", "error", err)
			return
		}
		slog.Info("released abandoned reservations", "ticket_types_touched", released)
	})
	go sweeper.Run(ctx)
}

// Mailer exposes the email dispatcher so main can drain it on shutdown and
// tests can wait for an asynchronous send to land.
func (s *Server) Mailer() *email.Mailer { return s.mailer }

// Tokens exposes the token service so tests can mint tokens directly.
func (s *Server) Tokens() *auth.TokenService { return s.tokens }

// Handler returns the fully wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// --- health -------------------------------------------------------------
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	// --- authentication -----------------------------------------------------
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/v1/auth/me", s.requireAuth(s.handleMe))

	// --- account recovery and verification (SRS 4.1) --------------------------
	mux.HandleFunc("POST /api/v1/auth/password-reset/request", s.handleRequestPasswordReset)
	mux.HandleFunc("POST /api/v1/auth/password-reset", s.handleResetPassword)
	mux.HandleFunc("POST /api/v1/auth/verify-email", s.handleVerifyEmail)
	mux.HandleFunc("POST /api/v1/auth/verify-email/request",
		s.requireAuth(s.handleRequestEmailVerification))

	// SRS 4.1: "Organizers shall have a profile containing contact and payout
	// information", and password management for a signed-in account.
	mux.HandleFunc("GET /api/v1/auth/profile", s.requireAuth(s.handleGetProfile))
	mux.HandleFunc("PATCH /api/v1/auth/profile", s.requireAuth(s.handleUpdateProfile))
	mux.HandleFunc("POST /api/v1/auth/password", s.requireAuth(s.handleChangePassword))

	// --- events -------------------------------------------------------------
	// The literal "/events/mine" pattern is more specific than "/events/{id}",
	// so Go's ServeMux routes it first.
	mux.HandleFunc("GET /api/v1/events", s.optionalAuth(s.handleListEvents))
	mux.HandleFunc("POST /api/v1/events", s.requireAuth(s.handleCreateEvent))
	mux.HandleFunc("GET /api/v1/events/mine", s.requireAuth(s.handleListMyEvents))
	mux.HandleFunc("GET /api/v1/events/{id}", s.optionalAuth(s.handleGetEvent))
	mux.HandleFunc("PATCH /api/v1/events/{id}", s.requireAuth(s.handleUpdateEvent))
	mux.HandleFunc("DELETE /api/v1/events/{id}", s.requireAuth(s.handleDeleteEvent))
	mux.HandleFunc("POST /api/v1/events/{id}/publish", s.requireAuth(s.handlePublishEvent))
	mux.HandleFunc("POST /api/v1/events/{id}/unpublish", s.requireAuth(s.handleUnpublishEvent))
	mux.HandleFunc("POST /api/v1/events/{id}/cancel", s.requireAuth(s.handleCancelEvent))

	// --- ticket types (organizer) -------------------------------------------
	mux.HandleFunc("GET /api/v1/events/{id}/ticket-types", s.requireAuth(s.handleListTicketTypes))
	mux.HandleFunc("POST /api/v1/events/{id}/ticket-types", s.requireAuth(s.handleCreateTicketType))
	mux.HandleFunc("PATCH /api/v1/ticket-types/{id}", s.requireAuth(s.handleUpdateTicketType))
	mux.HandleFunc("DELETE /api/v1/ticket-types/{id}", s.requireAuth(s.handleDeleteTicketType))

	// --- attendee-facing ----------------------------------------------------
	// Addressed by slug, because that is what appears in a shareable link.
	mux.HandleFunc("GET /api/v1/public/events/{slug}", s.handleGetPublicEvent)

	// --- uploads (SRS 4.2) --------------------------------------------------
	// Uploading needs an account; reading does not, because a banner is shown
	// on a public event page to people who are not signed in.
	mux.HandleFunc("POST /api/v1/uploads/images", s.requireAuth(s.handleUploadImage))
	mux.Handle("GET "+uploadURLPrefix+"{file}", s.uploadsHandler())

	// --- calendar export (SRS 4.11) -----------------------------------------
	// Addressed by id or slug, because the dashboard knows one and the public
	// page knows the other. Public: a calendar file carries the same
	// information the event page already shows.
	mux.HandleFunc("GET /api/v1/events/{id}/calendar.ics", s.optionalAuth(s.handleEventCalendar))

	// --- checkout -----------------------------------------------------------
	// Checkout takes optionalAuth: guests may buy, and a signed-in buyer gets
	// the order linked to their account.
	mux.HandleFunc("POST /api/v1/events/{id}/checkout", s.optionalAuth(s.handleCheckout))
	mux.HandleFunc("GET /api/v1/orders/{id}", s.optionalAuth(s.handleGetOrder))

	// --- assigned seating (SRS 4.3.1) ---------------------------------------
	// Public: an attendee sees what is left before deciding, and before
	// signing in.
	mux.HandleFunc("GET /api/v1/events/{id}/seats", s.optionalAuth(s.handleEventSeatMap))

	// --- cart holds (SRS 4.6, 4.3.1) ----------------------------------------
	// Anonymous, like checkout: an attendee picks seats before signing in, and
	// demanding an account to look at a seat map would lose the sale.
	mux.HandleFunc("POST /api/v1/events/{id}/holds", s.optionalAuth(s.handleCreateHold))
	mux.HandleFunc("GET /api/v1/orders/{id}/hold", s.optionalAuth(s.handleGetHold))
	mux.HandleFunc("DELETE /api/v1/orders/{id}/hold", s.optionalAuth(s.handleReleaseHold))
	mux.HandleFunc("POST /api/v1/orders/{id}/confirm", s.optionalAuth(s.handleConfirmHold))

	// --- paid-sales activation (SRS 4.5) ------------------------------------
	mux.HandleFunc("GET /api/v1/events/{id}/activation", s.requireAuth(s.handleGetActivation))
	mux.HandleFunc("POST /api/v1/events/{id}/activation", s.requireAuth(s.handleAdvanceActivation))

	// --- digital ticket delivery --------------------------------------------
	// Addressed by the ticket's UUID, which is the capability that lets a guest
	// buyer reach their own ticket without an account.
	mux.HandleFunc("GET /api/v1/tickets/{id}", s.handleGetTicket)
	mux.HandleFunc("GET /api/v1/tickets/{id}/pdf", s.handleTicketPDF)
	mux.HandleFunc("GET /api/v1/tickets/{id}/qr.png", s.handleTicketQR)

	// --- who is coming (organizer; the attendee list also serves door staff) --
	mux.HandleFunc("GET /api/v1/events/{id}/orders", s.requireAuth(s.handleListEventOrders))
	mux.HandleFunc("GET /api/v1/events/{id}/attendees", s.requireAuth(s.handleListAttendees))

	// --- check-in (the scanner app) -----------------------------------------
	// "/events/scannable" is a literal segment, so ServeMux prefers it over
	// "/events/{id}".
	mux.HandleFunc("GET /api/v1/events/scannable", s.requireAuth(s.handleListScannableEvents))
	mux.HandleFunc("POST /api/v1/events/{id}/check-in", s.requireAuth(s.handleCheckIn))
	// SRS 4.8: staff can find somebody by name when a QR will not scan. The
	// search itself is GET /events/{id}/attendees, shared with the dashboard.
	mux.HandleFunc("POST /api/v1/events/{id}/check-in/manual",
		s.requireAuth(s.handleManualCheckIn))
	mux.HandleFunc("GET /api/v1/events/{id}/check-in/stats", s.requireAuth(s.handleCheckInStats))
	mux.HandleFunc("POST /api/v1/tickets/{id}/check-in/reverse", s.requireAuth(s.handleReverseCheckIn))

	// --- event staff --------------------------------------------------------
	mux.HandleFunc("GET /api/v1/events/{id}/staff", s.requireAuth(s.handleListStaff))
	mux.HandleFunc("POST /api/v1/events/{id}/staff", s.requireAuth(s.handleAssignStaff))
	mux.HandleFunc("DELETE /api/v1/events/{id}/staff/{assignmentId}", s.requireAuth(s.handleRevokeStaff))

	// No catch-all route: it would shadow ServeMux's own 405 handling.
	// jsonRouterErrors turns the stdlib's plain-text 404/405 into the envelope.
	return recoverPanics(requestID(logRequests(withCORS(jsonRouterErrors(mux)))))
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Time     string `json:"time"`
}

// handleHealth reports whether the API and its database are up.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	body := healthResponse{
		Status:   "ok",
		Database: "up",
		Time:     s.now().UTC().Format(time.RFC3339),
	}
	if err := s.pool.Ping(r.Context()); err != nil {
		body.Status = "degraded"
		body.Database = "down"
		httpx.WriteJSON(w, http.StatusServiceUnavailable, body)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, body)
}
