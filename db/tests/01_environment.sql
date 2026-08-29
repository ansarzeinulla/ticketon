-- =============================================================================
-- Success criterion 1: the database container starts and is usable.
-- =============================================================================
BEGIN;
\ir _helpers.sql

DO $$
DECLARE
    v_version int := current_setting('server_version_num')::int;
BEGIN
    PERFORM t_section('01 - environment');

    PERFORM t_ok(v_version >= 160000,
        format('PostgreSQL server is 16 or newer (%s)', current_setting('server_version')));
    PERFORM t_eq(current_database(), 'biletflow', 'connected to the biletflow database');
    PERFORM t_ok(pg_is_in_recovery() = false, 'server accepts writes');
    PERFORM t_eq(current_setting('TimeZone'), 'UTC',
        'server time zone is UTC (events carry their own IANA zone)');

    PERFORM t_ok(EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'citext'),
        'extension citext is installed');
    PERFORM t_ok(EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto'),
        'extension pgcrypto is installed');
    PERFORM t_ok(EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'btree_gist'),
        'extension btree_gist is installed');

    PERFORM t_ok(gen_random_uuid() IS NOT NULL, 'gen_random_uuid() works');
END;
$$;

ROLLBACK;
