-- =============================================================================
-- Success criterion 3: a fake user can be INSERTed and SELECTed back.
--
-- These are exactly the statements from README.md "Verify by hand", run
-- automatically so the criterion cannot silently regress.
-- =============================================================================
BEGIN;
\ir _helpers.sql

-- --- the literal criterion, as a human would type it --------------------------
INSERT INTO users (email, password_hash, full_name, phone, locale, status)
VALUES ('aliya.test@biletflow.kz', '$2b$12$fakehashfakehashfakehashfakehashfakehashfa',
        'Aliya Testova', '+7 701 000 00 01', 'kk', 'active');

SELECT id, email, full_name, status, locale, created_at
FROM users
WHERE email = 'aliya.test@biletflow.kz';

DO $$
DECLARE
    v_id         uuid;
    v_created    timestamptz;
    v_updated    timestamptz;
    v_status     user_status;
    v_role_count int;
BEGIN
    PERFORM t_section('03 - user create / read / update / delete');

    -- ---- read back ----------------------------------------------------------
    SELECT id, created_at, updated_at, status
      INTO v_id, v_created, v_updated, v_status
      FROM users WHERE email = 'aliya.test@biletflow.kz';

    PERFORM t_ok(v_id IS NOT NULL, 'the inserted user can be selected back');
    PERFORM t_ok(v_id::text ~ '^[0-9a-f-]{36}$', 'the id was generated as a uuid');
    PERFORM t_eq(v_status, 'active'::user_status, 'status stored as an enum value');
    PERFORM t_ok(v_created IS NOT NULL AND v_updated IS NOT NULL,
        'created_at and updated_at are populated by defaults');

    -- ---- email is case-insensitive (citext) ---------------------------------
    PERFORM t_eq(
        (SELECT count(*)::int FROM users WHERE email = 'ALIYA.TEST@BILETFLOW.KZ'),
        1, 'email lookup is case-insensitive');

    -- ---- a new user gets the safe defaults ----------------------------------
    INSERT INTO users (email, password_hash, full_name)
    VALUES ('defaults.test@biletflow.kz', 'hash', 'Default Person');
    PERFORM t_eq(
        (SELECT status FROM users WHERE email = 'defaults.test@biletflow.kz'),
        'pending_verification'::user_status,
        'a new account defaults to pending_verification');
    PERFORM t_eq(
        (SELECT locale FROM users WHERE email = 'defaults.test@biletflow.kz'),
        'kk', 'a new account defaults to the kk locale');

    -- ---- roles --------------------------------------------------------------
    INSERT INTO user_roles (user_id, role) VALUES (v_id, 'attendee'), (v_id, 'organizer');
    SELECT count(*)::int INTO v_role_count FROM user_roles WHERE user_id = v_id;
    PERFORM t_eq(v_role_count, 2, 'a user can hold several roles at once');

    PERFORM t_throws(
        format('INSERT INTO user_roles (user_id, role) VALUES (%L, %L)', v_id, 'attendee'),
        'the same role cannot be granted twice', '23505');

    -- ---- update bumps updated_at -------------------------------------------
    UPDATE users SET full_name = 'Aliya Testova-Renamed' WHERE id = v_id;
    PERFORM t_ok(
        (SELECT updated_at FROM users WHERE id = v_id) > v_updated,
        'UPDATE moves updated_at forward via the trigger');
    PERFORM t_eq((SELECT full_name FROM users WHERE id = v_id),
        'Aliya Testova-Renamed', 'the update is visible on read');

    -- the application cannot backdate updated_at
    UPDATE users SET updated_at = timestamptz '2000-01-01' WHERE id = v_id;
    PERFORM t_ok(
        (SELECT updated_at FROM users WHERE id = v_id) > timestamptz '2020-01-01',
        'the trigger overrides an application-supplied updated_at');

    -- ---- delete cascades to owned rows --------------------------------------
    DELETE FROM users WHERE id = v_id;
    PERFORM t_eq((SELECT count(*)::int FROM users WHERE id = v_id), 0, 'the user was deleted');
    PERFORM t_eq((SELECT count(*)::int FROM user_roles WHERE user_id = v_id), 0,
        'deleting a user cascades to its roles');
END;
$$;

ROLLBACK;
