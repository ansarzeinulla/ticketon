package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/auth"
	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type ctxKey string

const claimsKey ctxKey = "claims"

// claimsFromContext returns the authenticated caller's token claims.
func claimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*auth.Claims)
	return c, ok
}

// mustUserID returns the authenticated user's id. It is only called from
// handlers behind requireAuth, where the claims are guaranteed present.
func mustUserID(ctx context.Context) uuid.UUID {
	claims, ok := claimsFromContext(ctx)
	if !ok {
		return uuid.Nil
	}
	id, _ := claims.UserID()
	return id
}

// requestID attaches a correlation id to every request and echoes it back.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(httpx.WithRequestID(r.Context(), id)))
	})
}

// statusRecorder captures the status code for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// logRequests writes one structured line per request.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", httpx.RequestIDFromContext(r.Context()),
		)
	})
}

// recoverPanics turns a panic into a 500 instead of dropping the connection.
func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"panic", rec,
					"path", r.URL.Path,
					"request_id", httpx.RequestIDFromContext(r.Context()),
					"stack", string(debug.Stack()),
				)
				httpx.WriteError(w, http.StatusInternalServerError,
					httpx.CodeInternal, "Something went wrong on our side.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withCORS allows browser clients during development. Production deployments
// should put a real origin allow-list in front of this.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "300")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth rejects a request unless it carries a valid bearer token for an
// account that is still allowed to sign in.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := bearerToken(r)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, err.Error())
			return
		}

		claims, err := s.tokens.Parse(token)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrTokenExpired):
				httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized,
					"The access token has expired. Sign in again.")
			default:
				httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized,
					"The access token is not valid.")
			}
			return
		}

		// The token is only a claim about the past. Re-read the account so a
		// suspended user cannot keep using a token issued before suspension.
		userID, _ := claims.UserID()
		user, err := s.users.GetByID(r.Context(), userID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized,
				"The account for this token no longer exists.")
			return
		}
		if err != nil {
			httpx.WriteInternalError(w, r, err)
			return
		}
		if user.Status == store.StatusSuspended || user.Status == store.StatusDeactivated {
			httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
				"This account is not allowed to sign in.")
			return
		}

		// Use the roles stored on the account, not the ones baked into the
		// token, so a role granted or revoked mid-session takes effect at once.
		claims.Roles = user.Roles
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
	}
}

// optionalAuth parses a bearer token when present but never rejects a request.
// Public endpoints use it to show an owner their own unpublished events.
func (s *Server) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := bearerToken(r)
		if err != nil {
			next(w, r)
			return
		}
		claims, err := s.tokens.Parse(token)
		if err != nil {
			next(w, r)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
	}
}

// bearerToken extracts the credential from the Authorization header.
func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", errors.New("An Authorization header is required.")
	}
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", errors.New("The Authorization header must be in the form 'Bearer <token>'.")
	}
	return strings.TrimSpace(token), nil
}

// jsonRouterErrors rewrites the plain-text 404 and 405 responses that
// http.ServeMux produces for unmatched routes into the API's JSON error shape.
//
// Handler-generated errors already set a JSON Content-Type before writing their
// status, so only the stdlib's text/plain replies are rewritten.
func jsonRouterErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&routerErrorWriter{ResponseWriter: w, req: r}, r)
	})
}

type routerErrorWriter struct {
	http.ResponseWriter
	req       *http.Request
	rewritten bool
}

func (w *routerErrorWriter) WriteHeader(status int) {
	if !w.rewritten && (status == http.StatusNotFound || status == http.StatusMethodNotAllowed) &&
		strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {

		w.rewritten = true

		code, message := httpx.CodeNotFound,
			"No route matches "+w.req.Method+" "+w.req.URL.Path+"."
		if status == http.StatusMethodNotAllowed {
			code = httpx.CodeMethodNotAllowed
			message = w.req.Method + " is not allowed on " + w.req.URL.Path +
				". Allowed: " + w.Header().Get("Allow") + "."
		}

		w.Header().Del("Content-Length")
		// httpx.WriteJSON sets the JSON content type and writes the body.
		httpx.WriteError(w.ResponseWriter, status, code, message)
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

// Write swallows the stdlib's plain-text body once a JSON body has replaced it.
func (w *routerErrorWriter) Write(b []byte) (int, error) {
	if w.rewritten {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
