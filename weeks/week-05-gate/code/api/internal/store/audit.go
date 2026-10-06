package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEntry is one line of an event's activity timeline (SRS 4.16).
type AuditEntry struct {
	EventID     *uuid.UUID
	ActorUserID *uuid.UUID
	Action      string
	EntityType  string
	EntityID    string
	Description string
	Metadata    map[string]any
}

// AuditStore appends to the append-only audit_logs table.
type AuditStore struct {
	pool *pgxpool.Pool
}

// NewAuditStore builds an AuditStore.
func NewAuditStore(pool *pgxpool.Pool) *AuditStore { return &AuditStore{pool: pool} }

// Append writes one entry. audit_logs rejects UPDATE and DELETE by trigger, so
// this is the only way its contents ever change.
func (s *AuditStore) Append(ctx context.Context, e AuditEntry) error {
	metadata := []byte("{}")
	if e.Metadata != nil {
		encoded, err := json.Marshal(e.Metadata)
		if err != nil {
			return err
		}
		metadata = encoded
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.EventID, e.ActorUserID, e.Action, e.EntityType, e.EntityID, e.Description, metadata)
	return mapError(err)
}

// ListForEvent returns an event's timeline, newest first.
func (s *AuditStore) ListForEvent(ctx context.Context, eventID uuid.UUID, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT event_id, actor_user_id, action, entity_type, entity_id, description
		  FROM audit_logs
		 WHERE event_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	entries := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var entityID, description *string
		if err := rows.Scan(&e.EventID, &e.ActorUserID, &e.Action, &e.EntityType, &entityID, &description); err != nil {
			return nil, err
		}
		if entityID != nil {
			e.EntityID = *entityID
		}
		if description != nil {
			e.Description = *description
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
