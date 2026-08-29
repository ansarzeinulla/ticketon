-- =============================================================================
-- Column- and table-level integrity: the database refuses invalid rows even if
-- the application layer forgets to validate them.
-- =============================================================================
BEGIN;
\ir _helpers.sql
\ir _fixture.sql

DO $$
BEGIN
    PERFORM t_section('04 - constraints');

    -- ---- users --------------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO users (email, password_hash, full_name)
        VALUES ('ORGANIZER.FIXTURE@BILETFLOW.TEST', 'hash', 'Duplicate') $q$,
        'email is unique regardless of letter case', '23505');

    PERFORM t_throws($q$
        INSERT INTO users (email, password_hash, full_name)
        VALUES ('not-an-email', 'hash', 'Bad Email') $q$,
        'a malformed email address is rejected', '23514');

    PERFORM t_throws($q$
        INSERT INTO users (email, password_hash, full_name)
        VALUES ('blank@biletflow.kz', 'hash', '   ') $q$,
        'a blank full_name is rejected', '23514');

    PERFORM t_throws($q$
        INSERT INTO users (email, full_name) VALUES ('nohash@biletflow.kz', 'No Hash') $q$,
        'password_hash is mandatory', '23502');

    PERFORM t_throws($q$
        INSERT INTO users (email, password_hash, full_name, locale)
        VALUES ('locale@biletflow.kz', 'hash', 'Bad Locale', 'fr') $q$,
        'only kk / ru / en locales are accepted', '23514');

    PERFORM t_throws($q$
        INSERT INTO users (email, password_hash, full_name, status)
        VALUES ('enum@biletflow.kz', 'hash', 'Bad Status', 'vip') $q$,
        'an unknown user_status value is rejected', '22P02');

    PERFORM t_throws($q$
        INSERT INTO organizer_profiles (user_id, display_name)
        VALUES ('10000000-0000-4000-8000-0000000000ff', 'Ghost') $q$,
        'an organizer profile cannot reference a missing user', '23503');

    -- ---- events -------------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO events (organizer_id, title, slug, starts_at, ends_at)
        VALUES ('10000000-0000-4000-8000-000000000001', 'Backwards', 'backwards',
                now() + interval '2 days', now() + interval '1 day') $q$,
        'an event cannot end before it starts', '23514');

    PERFORM t_throws($q$
        INSERT INTO events (organizer_id, title, slug, starts_at, ends_at, capacity)
        VALUES ('10000000-0000-4000-8000-000000000001', 'Zero Cap', 'zero-cap',
                now() + interval '1 day', now() + interval '2 days', 0) $q$,
        'event capacity must be positive when set', '23514');

    PERFORM t_throws($q$
        INSERT INTO events (organizer_id, title, slug, starts_at, ends_at, seating_mode)
        VALUES ('10000000-0000-4000-8000-000000000001', 'No Venue', 'no-venue',
                now() + interval '1 day', now() + interval '2 days', 'assigned_seating') $q$,
        'an assigned-seating event must have a venue', '23514');

    PERFORM t_throws($q$
        INSERT INTO events (organizer_id, title, slug, starts_at, ends_at)
        VALUES ('10000000-0000-4000-8000-000000000001', 'Clash', 'BiletFlow-Demo-Concert',
                now() + interval '1 day', now() + interval '2 days') $q$,
        'the event slug is unique and case-insensitive', '23505');

    -- ---- ticket types -------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO ticket_types (event_id, name, price_kzt, quantity_total)
        VALUES ('30000000-0000-4000-8000-000000000001', 'Negative', -1, 10) $q$,
        'a negative ticket price is rejected', '23514');

    PERFORM t_throws($q$
        INSERT INTO ticket_types (event_id, name, price_kzt, quantity_total, quantity_sold)
        VALUES ('30000000-0000-4000-8000-000000000001', 'Oversold', 100, 10, 11) $q$,
        'sold + reserved may not exceed the total inventory', '23514');

    PERFORM t_throws($q$
        UPDATE ticket_types SET quantity_reserved = 100
        WHERE id = '40000000-0000-4000-8000-000000000002' $q$,
        'reserving beyond remaining inventory is rejected', '23514');

    PERFORM t_throws($q$
        INSERT INTO ticket_types (event_id, name, price_kzt, quantity_total)
        VALUES ('30000000-0000-4000-8000-000000000001', 'Standard', 5000, 10) $q$,
        'ticket type names are unique within an event', '23505');

    PERFORM t_eq((SELECT is_free FROM ticket_types WHERE id = '40000000-0000-4000-8000-000000000001'),
        true,  'a zero-price ticket type is flagged free');
    PERFORM t_eq((SELECT is_free FROM ticket_types WHERE id = '40000000-0000-4000-8000-000000000002'),
        false, 'a priced ticket type is not flagged free');

    -- ---- orders -------------------------------------------------------------
    PERFORM t_throws($q$
        UPDATE orders SET total_kzt = 1 WHERE id = '50000000-0000-4000-8000-000000000001' $q$,
        'the order total must equal subtotal - discount + fee', '23514');

    PERFORM t_throws($q$
        UPDATE orders SET discount_kzt = 99999, total_kzt = -94849
        WHERE id = '50000000-0000-4000-8000-000000000001' $q$,
        'a discount larger than the subtotal is rejected', '23514');

    PERFORM t_throws($q$
        UPDATE orders SET currency = 'USD' WHERE id = '50000000-0000-4000-8000-000000000001' $q$,
        'only KZT is accepted in the initial release', '23514');

    PERFORM t_throws($q$
        UPDATE orders SET refunded_kzt = 999999 WHERE id = '50000000-0000-4000-8000-000000000001' $q$,
        'refunding more than the order total is rejected', '23514');

    PERFORM t_throws($q$
        INSERT INTO orders (order_number, event_id, buyer_email, buyer_name, promo_code_id)
        VALUES ('BF-TEST-9999', '30000000-0000-4000-8000-000000000001', 'x@y.kz', 'X',
                '70000000-0000-4000-8000-000000000002') $q$,
        'a promo code cannot be applied without campaign attribution', '23514');

    PERFORM t_throws($q$
        INSERT INTO orders (order_number, event_id, buyer_email, buyer_name)
        VALUES ('BF-TEST-0001', '30000000-0000-4000-8000-000000000001', 'x@y.kz', 'X') $q$,
        'order numbers are unique', '23505');

    -- ---- order items --------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO order_items (order_id, ticket_type_id, quantity, unit_price_kzt, line_total_kzt)
        VALUES ('50000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000002',
                2, 5000, 999) $q$,
        'the line total must equal price x quantity - discount', '23514');

    PERFORM t_throws($q$
        INSERT INTO order_items (order_id, ticket_type_id, quantity, unit_price_kzt, line_total_kzt, seat_id)
        VALUES ('50000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000002',
                2, 5000, 10000, '20000000-0000-4000-8000-000000000004') $q$,
        'an assigned seat always maps to exactly one ticket', '23514');

    PERFORM t_throws($q$
        INSERT INTO order_items (order_id, ticket_type_id, quantity, unit_price_kzt, line_total_kzt)
        VALUES ('50000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000002',
                0, 5000, 0) $q$,
        'an order item must contain at least one ticket', '23514');

    -- ---- payments -----------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO payments (purpose, order_id, event_id, amount_kzt, status, paid_at)
        VALUES ('paid_sales_activation', '50000000-0000-4000-8000-000000000001',
                '30000000-0000-4000-8000-000000000001', 1000, 'succeeded', now()) $q$,
        'an activation fee is never attached to a ticket order', '23514');

    PERFORM t_throws($q$
        INSERT INTO payments (purpose, order_id, amount_kzt, status)
        VALUES ('ticket_order', '50000000-0000-4000-8000-000000000001', 1000, 'succeeded') $q$,
        'a succeeded payment must record when it was paid', '23514');

    PERFORM t_eq((SELECT is_simulated FROM payments WHERE id = '50000000-0000-4000-8000-000000000004'),
        true, 'payments are flagged as simulated by default');

    -- ---- campaigns ----------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO campaigns (event_id, name, discount_type, discount_value, qr_token)
        VALUES ('30000000-0000-4000-8000-000000000001', 'Too Much', 'percentage', 120, 'CMP_TOOMUCH01') $q$,
        'a percentage discount above 100% is rejected', '23514');

    PERFORM t_throws($q$
        INSERT INTO campaigns (event_id, name, discount_type, discount_value, qr_token)
        VALUES ('30000000-0000-4000-8000-000000000001', 'Free Money', 'fixed_amount', 0, 'CMP_FREEMONEY1') $q$,
        'a zero discount is rejected', '23514');

    PERFORM t_throws($q$
        INSERT INTO promo_codes (campaign_id, code)
        VALUES ('70000000-0000-4000-8000-000000000001', 'student10') $q$,
        'promo codes are unique regardless of letter case', '23505');

    PERFORM t_throws($q$
        INSERT INTO promo_codes (campaign_id, code)
        VALUES ('70000000-0000-4000-8000-000000000001', 'no spaces allowed') $q$,
        'a promo code must be URL-safe', '23514');

    -- ---- support ------------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO support_cases (case_number, kind, category, subject, requester_user_id, status)
        VALUES ('SC-9001', 'attendee', 'refund', 'Where is my refund?',
                '10000000-0000-4000-8000-000000000002', 'resolved') $q$,
        'a resolved case must record when it was resolved', '23514');

    PERFORM t_throws($q$
        INSERT INTO support_cases (case_number, kind, category, subject, requester_user_id)
        VALUES ('SC-9002', 'attendee', 'lost_wallet', 'Help',
                '10000000-0000-4000-8000-000000000002') $q$,
        'an unknown support category is rejected', '22P02');

    -- ---- seat holds ---------------------------------------------------------
    PERFORM t_throws($q$
        INSERT INTO seat_holds (seat_id, event_id, expires_at, held_at)
        VALUES ('20000000-0000-4000-8000-000000000004', '30000000-0000-4000-8000-000000000001',
                now() - interval '1 minute', now()) $q$,
        'a seat hold cannot expire before it starts', '23514');
END;
$$;

ROLLBACK;
