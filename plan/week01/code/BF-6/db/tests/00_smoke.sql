-- =============================================================================
-- Smoke test: the database answers queries at all. If this fails, every other
-- file will fail too, so it runs first.
-- =============================================================================
BEGIN;
\ir _helpers.sql

DO $$
BEGIN
    PERFORM t_section('00 - smoke');

    PERFORM t_eq((SELECT 1), 1, 'the server answers a trivial query');
    PERFORM t_eq(current_database(), 'biletflow', 'connected to the biletflow database');
    PERFORM t_ok(to_regclass('public.users') IS NOT NULL, 'the users table exists');
END;
$$;

ROLLBACK;
