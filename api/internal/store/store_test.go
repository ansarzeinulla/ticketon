package store

import (
	"encoding/json"
	"testing"
	"time"
)

// Optional must tell three cases apart, because PATCH semantics depend on it:
// the key was absent, the key was null, or the key had a value.
func TestOptionalDistinguishesAbsentFromNull(t *testing.T) {
	type payload struct {
		Title       Optional[string] `json:"title"`
		Description Optional[string] `json:"description"`
		Capacity    Optional[int]    `json:"capacity"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"title":"New title","description":null}`), &p); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !p.Title.Set || !p.Title.Valid || p.Title.Value != "New title" {
		t.Errorf("Title = %+v, want set, valid, \"New title\"", p.Title)
	}
	if !p.Description.Set {
		t.Error("Description should be marked as present in the body")
	}
	if p.Description.Valid {
		t.Error("Description was explicitly null, so Valid should be false")
	}
	if p.Capacity.Set {
		t.Error("Capacity was absent from the body, so Set should be false")
	}
}

func TestOptionalPtr(t *testing.T) {
	var absent Optional[string]
	if absent.Ptr() != nil {
		t.Error("an absent Optional must produce a nil pointer")
	}

	explicitNull := Optional[string]{Set: true, Valid: false}
	if explicitNull.Ptr() != nil {
		t.Error("an explicit null must produce a nil pointer, so the column is cleared")
	}

	value := Optional[string]{Set: true, Valid: true, Value: "kept"}
	got := value.Ptr()
	if got == nil || *got != "kept" {
		t.Errorf("Ptr() = %v, want a pointer to \"kept\"", got)
	}
}

func TestOptionalTimeRoundTrip(t *testing.T) {
	type payload struct {
		StartsAt Optional[time.Time] `json:"starts_at"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"starts_at":"2026-10-01T18:00:00Z"}`), &p); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := time.Date(2026, 10, 1, 18, 0, 0, 0, time.UTC)
	if !p.StartsAt.Set || !p.StartsAt.Valid || !p.StartsAt.Value.Equal(want) {
		t.Errorf("StartsAt = %+v, want %v", p.StartsAt, want)
	}
}

func TestOptionalRejectsWrongType(t *testing.T) {
	type payload struct {
		Capacity Optional[int] `json:"capacity"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"capacity":"lots"}`), &p); err == nil {
		t.Error("Unmarshal() accepted a string for an integer field")
	}
}
