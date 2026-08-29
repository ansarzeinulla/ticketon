// Package store contains the PostgreSQL data access for the API. It maps the
// Phase 1 schema onto Go types and turns database constraint violations into
// domain errors the HTTP layer can translate into status codes.
package store

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinel errors returned by the stores.
var (
	ErrNotFound   = errors.New("record not found")
	ErrEmailTaken = errors.New("email is already registered")
	ErrSlugTaken  = errors.New("slug is already in use")
	// ErrStatusUnchanged reports a moderation action that would not change
	// anything - suspending an already-suspended account, for instance. It is
	// a 409 rather than a silent success, so a double click is answered
	// honestly instead of implying a second action took place.
	ErrStatusUnchanged = errors.New("record is already in that status")
)

// ConstraintError reports a database constraint that the application layer
// should have caught first. Surfacing the constraint name makes the cause
// obvious in logs instead of showing up as a generic 500.
type ConstraintError struct {
	Constraint string
	Code       string
	Detail     string
}

func (e *ConstraintError) Error() string {
	return fmt.Sprintf("constraint %q violated (sqlstate %s): %s", e.Constraint, e.Code, e.Detail)
}

// PostgreSQL error codes used by the mapping below.
const (
	codeUniqueViolation     = "23505"
	codeCheckViolation      = "23514"
	codeForeignKeyViolation = "23503"
	codeNotNullViolation    = "23502"
)

// mapError converts a pgx error into a domain error.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case codeUniqueViolation:
		switch pgErr.ConstraintName {
		case "users_email_key":
			return ErrEmailTaken
		case "events_slug_key":
			return ErrSlugTaken
		}
	case codeCheckViolation, codeForeignKeyViolation, codeNotNullViolation:
		return &ConstraintError{Constraint: pgErr.ConstraintName, Code: pgErr.Code, Detail: pgErr.Detail}
	}
	return err
}

// Optional distinguishes "field absent from the JSON body" from "field
// explicitly set to null", which PATCH semantics require: an absent field is
// left alone, an explicit null clears the column.
type Optional[T any] struct {
	Set   bool // the key was present in the request body
	Valid bool // the value was not null
	Value T
}

// UnmarshalJSON records presence and nullness alongside the value.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Valid = false
		var zero T
		o.Value = zero
		return nil
	}
	if err := json.Unmarshal(data, &o.Value); err != nil {
		return err
	}
	o.Valid = true
	return nil
}

// Ptr returns the value as a pointer, or nil when absent or null.
func (o Optional[T]) Ptr() *T {
	if !o.Set || !o.Valid {
		return nil
	}
	v := o.Value
	return &v
}

// isUniqueViolation reports whether err is a unique violation on the named
// constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == codeUniqueViolation &&
		pgErr.ConstraintName == constraint
}
