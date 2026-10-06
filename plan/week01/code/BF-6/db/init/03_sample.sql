-- =============================================================================
-- BiletFlow - 03_sample.sql
-- A couple of rows so the schema can be queried by hand before real seed data
-- exists. Idempotent: re-running it changes nothing.
-- =============================================================================

INSERT INTO users (id, email, password_hash, full_name, locale, status, email_verified_at)
VALUES
    ('a0000000-0000-4000-8000-000000000001', 'organizer@example.kz',
     'not-a-real-hash', 'Sample Organizer', 'kk', 'active', now()),
    ('a0000000-0000-4000-8000-000000000002', 'attendee@example.kz',
     'not-a-real-hash', 'Sample Attendee', 'ru', 'active', now())
ON CONFLICT (id) DO NOTHING;
