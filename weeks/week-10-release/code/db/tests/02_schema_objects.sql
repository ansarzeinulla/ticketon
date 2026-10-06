-- =============================================================================
-- Success criterion 2: running the schema script creates every table.
--
-- The expected list is the SRS section 6 "Core Data Entities" list, plus the
-- join tables and paid_sales_activations that the entities imply.
-- =============================================================================
BEGIN;
\ir _helpers.sql

DO $$
DECLARE
    v_expected_tables text[] := ARRAY[
        'attendees', 'audit_logs', 'campaign_ticket_types', 'campaigns',
        'check_in_records', 'event_reports', 'events', 'notifications',
        'order_items', 'orders', 'organizer_profiles', 'paid_sales_activations',
        'payments', 'payout_accounts', 'platform_settings', 'promo_codes',
        'promo_redemptions', 'refunds',
        'seat_holds', 'seat_rows', 'seats', 'staff_assignments',
        'support_cases', 'support_messages', 'ticket_types', 'tickets',
        'user_roles', 'user_tokens', 'users', 'venue_sections', 'venues'
    ];
    v_expected_enums text[] := ARRAY[
        'activation_status', 'campaign_status', 'discount_type',
        'event_report_reason', 'event_report_status', 'event_status',
        'event_visibility', 'notification_channel', 'notification_status',
        'order_status', 'payment_purpose', 'payment_status',
        'payout_account_status', 'refund_status', 'seat_hold_status',
        'seating_mode', 'staff_role', 'support_case_category',
        'support_case_kind', 'support_case_status', 'ticket_status',
        'user_role', 'user_status', 'user_token_purpose'
    ];
    v_actual   text[];
    v_missing  text[];
    v_extra    text[];
    v_name     text;
BEGIN
    PERFORM t_section('02 - schema objects');

    -- ---- tables -------------------------------------------------------------
    SELECT array_agg(tablename ORDER BY tablename) INTO v_actual
    FROM pg_tables WHERE schemaname = 'public';

    SELECT array_agg(x ORDER BY x) INTO v_missing
    FROM unnest(v_expected_tables) x WHERE NOT (x = ANY (COALESCE(v_actual, '{}')));

    SELECT array_agg(x ORDER BY x) INTO v_extra
    FROM unnest(COALESCE(v_actual, '{}')) x WHERE NOT (x = ANY (v_expected_tables));

    PERFORM t_ok(v_missing IS NULL, format('no missing tables (missing: %s)', v_missing));
    PERFORM t_ok(v_extra IS NULL, format('no unexpected tables (extra: %s)', v_extra));
    PERFORM t_eq(cardinality(v_actual), cardinality(v_expected_tables), 'table count matches');

    -- ---- enum types ---------------------------------------------------------
    FOREACH v_name IN ARRAY v_expected_enums LOOP
        PERFORM t_ok(
            EXISTS (SELECT 1 FROM pg_type WHERE typname = v_name AND typtype = 'e'),
            format('enum type %s exists', v_name));
    END LOOP;

    -- ---- every table has a primary key --------------------------------------
    SELECT array_agg(t.tablename ORDER BY t.tablename) INTO v_missing
    FROM pg_tables t
    WHERE t.schemaname = 'public'
      AND NOT EXISTS (
          SELECT 1 FROM pg_constraint c
          WHERE c.conrelid = format('public.%I', t.tablename)::regclass
            AND c.contype = 'p');
    PERFORM t_ok(v_missing IS NULL, format('every table has a primary key (without: %s)', v_missing));

    -- ---- no foreign key points at a missing index-able target ---------------
    PERFORM t_ok(
        (SELECT count(*) FROM pg_constraint WHERE contype = 'f'
           AND connamespace = 'public'::regnamespace) >= 40,
        'foreign keys are defined across the schema');

    -- ---- guard rails that later phases depend on ----------------------------
    PERFORM t_ok(
        EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public'
                AND indexname = 'check_in_one_active_per_ticket_uidx'),
        'unique index preventing a second live check-in exists');
    PERFORM t_ok(
        EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public'
                AND indexname = 'tickets_one_live_ticket_per_seat_uidx'),
        'unique index preventing a seat being sold twice exists');
    PERFORM t_ok(
        EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public'
                AND indexname = 'seat_holds_one_active_per_seat_uidx'),
        'unique index allowing only one active seat hold exists');
    PERFORM t_ok(
        EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_audit_logs_append_only'),
        'append-only trigger on audit_logs exists');

    -- ---- updated_at triggers ------------------------------------------------
    FOREACH v_name IN ARRAY ARRAY['users', 'events', 'orders', 'tickets', 'ticket_types'] LOOP
        PERFORM t_ok(
            EXISTS (SELECT 1 FROM pg_trigger
                    WHERE tgname = 'trg_set_updated_at'
                      AND tgrelid = format('public.%I', v_name)::regclass),
            format('updated_at trigger on %s exists', v_name));
    END LOOP;
END;
$$;

ROLLBACK;
