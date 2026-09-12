-- =============================================================================
-- BiletFlow test helpers.
--
-- Included by every test file with \ir. The functions are created inside the
-- test transaction and disappear when it rolls back, so running the suite never
-- leaves anything behind in the database.
-- =============================================================================

-- Custom SQLSTATE used only by failing assertions.
CREATE OR REPLACE FUNCTION t_ok(p_condition boolean, p_message text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF p_condition IS NOT TRUE THEN
        RAISE EXCEPTION 'FAIL: %', p_message USING ERRCODE = 'ZZ001';
    END IF;
    RAISE NOTICE 'ok - %', p_message;
END;
$$;

CREATE OR REPLACE FUNCTION t_eq(p_actual anyelement, p_expected anyelement, p_message text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF p_actual IS DISTINCT FROM p_expected THEN
        RAISE EXCEPTION 'FAIL: % (expected %, got %)', p_message, p_expected, p_actual
            USING ERRCODE = 'ZZ001';
    END IF;
    RAISE NOTICE 'ok - % (= %)', p_message, p_actual;
END;
$$;

-- Asserts that a statement is REJECTED by the database.
-- p_errcode, when supplied, pins the exact SQLSTATE so a test cannot pass for
-- the wrong reason (e.g. a typo in the test SQL).
CREATE OR REPLACE FUNCTION t_throws(p_sql text, p_message text, p_errcode text DEFAULT NULL)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    v_state text;
    v_msg   text;
BEGIN
    BEGIN
        EXECUTE p_sql;
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS v_state = RETURNED_SQLSTATE, v_msg = MESSAGE_TEXT;
        IF p_errcode IS NOT NULL AND v_state <> p_errcode THEN
            RAISE EXCEPTION 'FAIL: % (expected SQLSTATE %, got % - %)',
                p_message, p_errcode, v_state, v_msg USING ERRCODE = 'ZZ001';
        END IF;
        RAISE NOTICE 'ok - % [rejected: %]', p_message, v_state;
        RETURN;
    END;
    RAISE EXCEPTION 'FAIL: % (the statement was ACCEPTED but should have been rejected)',
        p_message USING ERRCODE = 'ZZ001';
END;
$$;

CREATE OR REPLACE FUNCTION t_section(p_title text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    RAISE NOTICE '%', p_title;
END;
$$;
