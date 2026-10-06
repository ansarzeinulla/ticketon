-- =============================================================================
-- Success criterion 2: running the schema script creates every table.
-- The expected list follows the SRS section 6 "Core Data Entities".
-- =============================================================================
BEGIN;
\ir _helpers.sql

DO $$
DECLARE
    v_expected text[] := ARRAY[
        'attendees', 'audit_logs', 'campaign_ticket_types', 'campaigns',
        'check_in_records', 'event_reports', 'events', 'notifications',
        'order_items', 'orders', 'organizer_profiles', 'paid_sales_activations',
        'payments', 'payout_accounts', 'platform_settings', 'promo_codes',
        'promo_redemptions', 'refunds',
        'seat_holds', 'seat_rows', 'seats', 'staff_assignments',
        'support_cases', 'support_messages', 'ticket_types', 'tickets',
        'user_roles', 'user_tokens', 'users', 'venue_sections', 'venues'
    ];
    v_name text;
BEGIN
    PERFORM t_section('02 - tables');

    FOREACH v_name IN ARRAY v_expected LOOP
        PERFORM t_ok(to_regclass('public.' || v_name) IS NOT NULL,
            format('table %s exists', v_name));
    END LOOP;

    PERFORM t_ok(EXISTS (
        SELECT 1 FROM pg_indexes
         WHERE tablename = 'users' AND indexdef ILIKE '%UNIQUE%(email)%'),
        'users.email is unique');
END;
$$;

ROLLBACK;
