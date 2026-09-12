// Package httpx contains the JSON request/response plumbing shared by every
// handler: one decode helper, one response envelope and one error shape.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// maxBodyBytes caps request bodies so a malicious client cannot exhaust memory.
const maxBodyBytes = 1 << 20 // 1 MiB

// Error codes returned in the "code" field. Clients should switch on these
// rather than on the human-readable message.
const (
	CodeValidationFailed   = "validation_failed"
	CodeInvalidJSON        = "invalid_json"
	CodeUnauthorized       = "unauthorized"
	CodeInvalidCredentials = "invalid_credentials"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeConflict           = "conflict"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeInternal           = "internal_error"
)

// ErrorBody is the payload of every non-2xx response.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type errorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// WriteJSON serialises v as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		slog.Error("encode response", "error", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"Failed to encode the response."}}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_, _ = w.Write(body)
	}
}

// WriteError sends a structured error response.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: ErrorBody{Code: code, Message: message}})
}

// WriteValidationError sends a 422 listing the offending fields.
func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteJSON(w, http.StatusUnprocessableEntity, errorEnvelope{Error: ErrorBody{
		Code:    CodeValidationFailed,
		Message: "The request body failed validation.",
		Fields:  fields,
	}})
}

// WriteInternalError logs the underlying cause and returns a generic message,
// so internal details never leak to the client.
func WriteInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed",
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", RequestIDFromContext(r.Context()),
	)
	WriteError(w, http.StatusInternalServerError, CodeInternal, "Something went wrong on our side.")
}

// DecodeJSON reads exactly one JSON object into dst. It rejects unknown fields
// so a typo in a client payload is reported instead of silently ignored.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		if mediaType := strings.TrimSpace(strings.Split(ct, ";")[0]); mediaType != "application/json" {
			return fmt.Errorf("expected Content-Type application/json, got %q", mediaType)
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("body must contain exactly one JSON object")
	}
	return nil
}

func decodeError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var maxBytesErr *http.MaxBytesError

	switch {
	case errors.As(err, &syntaxErr):
		return fmt.Errorf("malformed JSON at character %d", syntaxErr.Offset)
	case errors.As(err, &typeErr):
		if typeErr.Field != "" {
			return fmt.Errorf("field %q must be of type %s", typeErr.Field, typeErr.Type)
		}
		return fmt.Errorf("body must be a JSON object")
	case errors.As(err, &maxBytesErr):
		return fmt.Errorf("body must not exceed %d bytes", maxBytesErr.Limit)
	case errors.Is(err, io.EOF):
		return errors.New("body must not be empty")
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return fmt.Errorf("unknown field %s", field)
	default:
		return err
	}
}

// LogAuditFailure records that an audit entry could not be written. The caller
// has already completed its work, so this must never fail the request - but it
// must never be swallowed silently either.
func LogAuditFailure(ctx context.Context, action string, err error) {
	slog.Error("audit append failed",
		"action", action,
		"error", err,
		"request_id", RequestIDFromContext(ctx),
	)
}
