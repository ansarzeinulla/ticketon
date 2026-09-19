// Package api wires the HTTP routes, middleware and handlers together.
package api

import (
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
	return &Server{
		cfg:           cfg,
		pool:          pool,
		users:         store.NewUserStore(pool),
		accountTokens: store.NewTokenStore(pool),
		mailer:        email.NewMailer(sender, nil),
		hasher:        auth.NewHasher(cfg.BcryptCost),
		tokens:        auth.NewTokenService(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL),
		now:           time.Now,
	}
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
	if !s.cfg.IsProduction() {
		mux.HandleFunc("GET /dev/config", s.handleDevConfig)
	}

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
