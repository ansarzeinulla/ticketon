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
	cfg         config.Config
	pool        *pgxpool.Pool
	users       *store.UserStore
	events      *store.EventStore
	ticketTypes *store.TicketTypeStore
	tickets     *store.TicketStore
	campaigns   *store.CampaignStore
	checkIns    *store.CheckInStore
	staff       *store.StaffStore
	support     *store.SupportStore
	analytics   *store.AnalyticsStore
	checkout    *store.CheckoutStore
	audit       *store.AuditStore

	// Phase 10: refunds, paid-sales activation and notifications.
	refunds       *store.RefundStore
	activations   *store.ActivationStore
	notifications *store.NotificationStore
	// Phase 13: the moderation queue and platform settings (SRS 4.12).
	moderation *store.ModerationStore
	// SRS 4.1: organizer profiles and self-service password change.
	profiles *store.ProfileStore
	mailer   *email.Mailer

	// Phase 12: account tokens, the admin portal and uploads.
	seating       *store.SeatingStore
	offline       *store.OfflineStore
	accountTokens *store.TokenStore
	admin         *store.AdminStore
	attendees     *store.AttendeeStore

	hasher *auth.Hasher
	tokens *auth.TokenService
	now    func() time.Time
}

// New builds a Server from a live connection pool. Notifications go to the
// console; NewWithSender swaps that for a test double.
func New(cfg config.Config, pool *pgxpool.Pool) *Server {
	return NewWithSender(cfg, pool, email.NewConsoleSender(nil))
}

// NewWithSender builds a Server with an explicit notification transport.
func NewWithSender(cfg config.Config, pool *pgxpool.Pool, sender email.Sender) *Server {
	s := &Server{
		cfg:         cfg,
		pool:        pool,
		users:       store.NewUserStore(pool),
		events:      store.NewEventStore(pool),
		ticketTypes: store.NewTicketTypeStore(pool),
		tickets:     store.NewTicketStore(pool),
		campaigns:   store.NewCampaignStore(pool),
		checkIns:    store.NewCheckInStore(pool),
		staff:       store.NewStaffStore(pool),
		support:     store.NewSupportStore(pool),
		analytics:   store.NewAnalyticsStore(pool),
		checkout: store.NewCheckoutStoreWithFees(pool, store.Fees{
			Percent:  cfg.ProcessingFeePercent,
			FixedKZT: cfg.ProcessingFeeFixedKZT,
		}),
		audit: store.NewAuditStore(pool),

		refunds:       store.NewRefundStore(pool),
		activations:   store.NewActivationStore(pool),
		notifications: store.NewNotificationStore(pool),
		moderation:    store.NewModerationStore(pool),
		profiles:      store.NewProfileStore(pool),

		seating:       store.NewSeatingStore(pool),
		offline:       store.NewOfflineStore(pool),
		accountTokens: store.NewTokenStore(pool),
		admin:         store.NewAdminStore(pool),
		attendees:     store.NewAttendeeStore(pool),

		hasher: auth.NewHasher(cfg.BcryptCost),
		tokens: auth.NewTokenService(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL),
		now:    time.Now,
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

// Mailer exposes the notification dispatcher so main can drain it on shutdown
// and tests can wait for an asynchronous send to land.
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

	// --- account recovery and verification (SRS 4.1) ------------------------
	// All public: a password-reset link is followed from an inbox, where
	// nobody is signed in. The token is the credential.
	mux.HandleFunc("POST /api/v1/auth/password-reset/request", s.handleRequestPasswordReset)
	mux.HandleFunc("POST /api/v1/auth/password-reset", s.handleResetPassword)
	mux.HandleFunc("POST /api/v1/auth/verify-email", s.handleVerifyEmail)
	// SRS 4.1: "Organizers shall have a profile containing contact and payout
	// information", and password management for a signed-in account.
	mux.HandleFunc("GET /api/v1/auth/profile", s.requireAuth(s.handleGetProfile))
	mux.HandleFunc("PATCH /api/v1/auth/profile", s.requireAuth(s.handleUpdateProfile))
	mux.HandleFunc("POST /api/v1/auth/password", s.requireAuth(s.handleChangePassword))

	mux.HandleFunc("POST /api/v1/auth/verify-email/request",
		s.requireAuth(s.handleRequestEmailVerification))

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
	mux.HandleFunc("GET /api/v1/events/{id}/timeline", s.requireAuth(s.handleEventTimeline))
	mux.HandleFunc("GET /api/v1/events/{id}/analytics", s.requireAuth(s.handleEventAnalytics))
	mux.HandleFunc("POST /api/v1/events/{id}/duplicate", s.requireAuth(s.handleDuplicateEvent))

	// --- ticket types (organizer) -------------------------------------------
	mux.HandleFunc("GET /api/v1/events/{id}/ticket-types", s.requireAuth(s.handleListTicketTypes))
	mux.HandleFunc("POST /api/v1/events/{id}/ticket-types", s.requireAuth(s.handleCreateTicketType))
	mux.HandleFunc("PATCH /api/v1/ticket-types/{id}", s.requireAuth(s.handleUpdateTicketType))
	mux.HandleFunc("DELETE /api/v1/ticket-types/{id}", s.requireAuth(s.handleDeleteTicketType))

	// --- attendee-facing ----------------------------------------------------
	// Addressed by slug, because that is what appears in a shareable link.
	mux.HandleFunc("GET /api/v1/public/events/{slug}", s.handleGetPublicEvent)

	// --- calendar export (SRS 4.11) -----------------------------------------
	// Addressed by id or slug, because the dashboard knows one and the public
	// page knows the other. Public: a calendar file carries the same
	// information the event page already shows.
	mux.HandleFunc("GET /api/v1/events/{id}/calendar.ics", s.optionalAuth(s.handleEventCalendar))

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

	// --- refunds (SRS 4.9) --------------------------------------------------
	// Addressed by order, authorised by the order's event.
	mux.HandleFunc("POST /api/v1/orders/{id}/refund", s.requireAuth(s.handleRefundOrder))
	// SRS 4.9: a free registration is cancelled rather than refunded - there is
	// no money to reverse, and refunds_amount_chk requires a positive amount.
	mux.HandleFunc("POST /api/v1/orders/{id}/cancel", s.requireAuth(s.handleCancelOrder))
	mux.HandleFunc("GET /api/v1/events/{id}/orders", s.requireAuth(s.handleListEventOrders))

	// --- paid-sales activation (SRS 4.5) ------------------------------------
	mux.HandleFunc("GET /api/v1/events/{id}/activation", s.requireAuth(s.handleGetActivation))
	mux.HandleFunc("POST /api/v1/events/{id}/activation", s.requireAuth(s.handleAdvanceActivation))

	// --- digital ticket delivery --------------------------------------------
	// Addressed by the ticket's UUID, which is the capability that lets a guest
	// buyer reach their own ticket without an account.
	mux.HandleFunc("GET /api/v1/tickets/{id}", s.handleGetTicket)
	mux.HandleFunc("GET /api/v1/tickets/{id}/pdf", s.handleTicketPDF)
	mux.HandleFunc("GET /api/v1/tickets/{id}/qr.png", s.handleTicketQR)

	// --- promotional campaigns ----------------------------------------------
	mux.HandleFunc("GET /api/v1/events/{id}/campaigns", s.requireAuth(s.handleListCampaigns))
	mux.HandleFunc("POST /api/v1/events/{id}/campaigns", s.requireAuth(s.handleCreateCampaign))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}", s.requireAuth(s.handleSetCampaignStatus))
	mux.HandleFunc("DELETE /api/v1/campaigns/{id}", s.requireAuth(s.handleDeleteCampaign))
	// The campaign QR image is public, like the ticket QR: an <img> tag cannot
	// send a bearer header, and a code destined for a poster has nothing to
	// hide. Managing campaigns still requires the organizer's token.
	mux.HandleFunc("GET /api/v1/campaigns/{id}/qr.png", s.handleCampaignQR)

	// Attendee-facing: price a promo code against a basket before paying.
	mux.HandleFunc("POST /api/v1/events/{id}/promo/preview", s.optionalAuth(s.handlePromoPreview))

	// --- check-in (the scanner app) -----------------------------------------
	// "/events/scannable" is a literal segment, so ServeMux prefers it over
	// "/events/{id}".
	mux.HandleFunc("GET /api/v1/events/scannable", s.requireAuth(s.handleListScannableEvents))
	mux.HandleFunc("POST /api/v1/events/{id}/check-in", s.requireAuth(s.handleCheckIn))
	// SRS 4.8: staff can find somebody by name when a QR will not scan.
	mux.HandleFunc("GET /api/v1/events/{id}/attendees", s.requireAuth(s.handleSearchAttendees))
	mux.HandleFunc("POST /api/v1/events/{id}/check-in/manual",
		s.requireAuth(s.handleManualCheckIn))

	// SRS 4.8: work offline, then reconcile. The roster is the guest list plus
	// the means to validate it, so it is gated exactly like scanning.
	mux.HandleFunc("GET /api/v1/events/{id}/roster", s.requireAuth(s.handleEventRoster))
	mux.HandleFunc("POST /api/v1/events/{id}/check-in/sync",
		s.requireAuth(s.handleSyncCheckIns))
	mux.HandleFunc("GET /api/v1/events/{id}/check-in/stats", s.requireAuth(s.handleCheckInStats))
	mux.HandleFunc("POST /api/v1/tickets/{id}/check-in/reverse", s.requireAuth(s.handleReverseCheckIn))

	// --- support cases (SRS 4.13) -------------------------------------------
	mux.HandleFunc("GET /api/v1/support/categories", s.handleSupportCategories)
	mux.HandleFunc("GET /api/v1/support/cases", s.requireAuth(s.handleListMyCases))
	mux.HandleFunc("POST /api/v1/support/cases", s.requireAuth(s.handleOpenCase))
	mux.HandleFunc("GET /api/v1/support/cases/{id}", s.requireAuth(s.handleGetCase))
	mux.HandleFunc("POST /api/v1/support/cases/{id}/messages", s.requireAuth(s.handlePostCaseMessage))
	mux.HandleFunc("PATCH /api/v1/support/cases/{id}", s.requireAuth(s.handleSetCaseStatus))
	// SRS 4.13: "Authorized staff shall be able to assign a case."
	mux.HandleFunc("POST /api/v1/support/cases/{id}/assign", s.requireAuth(s.handleAssignCase))
	mux.HandleFunc("GET /api/v1/events/{id}/support/cases", s.requireAuth(s.handleListEventCases))

	// --- platform moderation (SRS 4.12) -------------------------------------
	mux.HandleFunc("POST /api/v1/admin/events/{id}/suspend",
		s.requirePlatformAdmin(s.handleSuspendEvent))
	mux.HandleFunc("POST /api/v1/admin/events/{id}/unsuspend",
		s.requirePlatformAdmin(s.handleUnsuspendEvent))
	// Narrower than suspending the event: this stops the money while leaving
	// free registration open (SRS 4.5).
	mux.HandleFunc("POST /api/v1/admin/events/{id}/paid-sales/suspend",
		s.requirePlatformAdmin(s.handleSuspendPaidSales))
	mux.HandleFunc("POST /api/v1/admin/events/{id}/paid-sales/unsuspend",
		s.requirePlatformAdmin(s.handleUnsuspendPaidSales))

	// --- the administrative portal (SRS 2.1, 4.12) --------------------------
	// SRS 4.12: "Suspend users or events." Events have been suspendable since
	// Phase 8; this is the other half.
	mux.HandleFunc("POST /api/v1/admin/users/{id}/suspend",
		s.requirePlatformAdmin(s.handleSuspendUser))
	mux.HandleFunc("POST /api/v1/admin/users/{id}/unsuspend",
		s.requirePlatformAdmin(s.handleUnsuspendUser))

	// SRS 4.12: "Review reported events." Anyone signed in can file a report;
	// only platform staff can see or decide the queue.
	mux.HandleFunc("POST /api/v1/events/{id}/report", s.requireAuth(s.handleReportEvent))
	mux.HandleFunc("GET /api/v1/admin/event-reports", s.requirePlatformAdmin(s.handleListReports))
	mux.HandleFunc("PATCH /api/v1/admin/event-reports/{id}",
		s.requirePlatformAdmin(s.handleReviewReport))

	// SRS 4.12: "Configure activation fees and platform settings."
	mux.HandleFunc("GET /api/v1/admin/settings", s.requirePlatformAdmin(s.handleListSettings))
	mux.HandleFunc("PATCH /api/v1/admin/settings/{key}",
		s.requirePlatformAdmin(s.handleUpdateSetting))

	mux.HandleFunc("GET /api/v1/admin/search", s.requirePlatformAdmin(s.handleAdminSearch))
	mux.HandleFunc("GET /api/v1/admin/reports/events.csv",
		s.requirePlatformAdmin(s.handleAdminReport))

	// --- uploads (SRS 4.2) --------------------------------------------------
	// Uploading needs an account; reading does not, because a banner is shown
	// on a public event page to people who are not signed in.
	mux.HandleFunc("POST /api/v1/uploads/images", s.requireAuth(s.handleUploadImage))
	mux.Handle("GET "+uploadURLPrefix+"{file}", s.uploadsHandler())

	// --- event staff --------------------------------------------------------
	mux.HandleFunc("GET /api/v1/events/{id}/staff", s.requireAuth(s.handleListStaff))
	mux.HandleFunc("POST /api/v1/events/{id}/staff", s.requireAuth(s.handleAssignStaff))
	mux.HandleFunc("DELETE /api/v1/events/{id}/staff/{assignmentId}", s.requireAuth(s.handleRevokeStaff))

	// No catch-all route is registered: one would shadow ServeMux's own 405
	// handling and turn every wrong-method request into a 404. jsonRouterErrors
	// converts the stdlib's plain-text 404/405 replies into the JSON envelope
	// used everywhere else, keeping the Allow header intact.
	return recoverPanics(requestID(logRequests(withCORS(jsonRouterErrors(mux)))))
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Time     string `json:"time"`
}

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
