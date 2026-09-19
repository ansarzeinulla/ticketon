-- =============================================================================
-- Temporary trigger checks, written while the API started writing audit_logs.
-- They pin the two triggers the timeline depends on until the business-rules
-- suite covers them properly: audit rows cannot be edited or deleted, and
-- updated_at moves on every UPDATE.
-- =============================================================================
BEGIN;
\ir _helpers.sql
\ir _fixture.sql

DO $$
DECLARE
    v_audit_id bigint;
    v_before   timestamptz;
    v_after    timestamptz;
BEGIN
    PERFORM t_section('07 - trigger checks (temporary)');

    INSERT INTO audit_logs (event_id, action, entity_type, description)
    VALUES ('30000000-0000-4000-8000-000000000001', 'event.published', 'event', 'temp check')
    RETURNING id INTO v_audit_id;
    PERFORM t_ok(v_audit_id IS NOT NULL, 'an audit row can be appended');

    PERFORM t_throws(format($q$UPDATE audit_logs SET description = 'rewritten' WHERE id = %s$q$, v_audit_id),
        'an audit row cannot be edited', 'P0001');
    PERFORM t_throws(format($q$DELETE FROM audit_logs WHERE id = %s$q$, v_audit_id),
        'an audit row cannot be deleted', 'P0001');

    SELECT updated_at INTO v_before FROM events WHERE id = '30000000-0000-4000-8000-000000000001';
    PERFORM pg_sleep(0.01);
    UPDATE events SET description = 'touched by the trigger check'
     WHERE id = '30000000-0000-4000-8000-000000000001';
    SELECT updated_at INTO v_after FROM events WHERE id = '30000000-0000-4000-8000-000000000001';
    PERFORM t_ok(v_after > v_before, 'updated_at moves when an event is edited');
END;
$$;

ROLLBACK;
