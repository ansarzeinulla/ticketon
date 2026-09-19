-- =============================================================================
-- The SRS invariants that must hold no matter which service writes the row:
-- no double admission, no double-sold seat, no forged campaign QR, no edited
-- audit trail, no redemption past the campaign limit.
-- =============================================================================
BEGIN;
\ir _helpers.sql
\ir _fixture.sql

DO $$
DECLARE
    v_audit_id bigint;
BEGIN
    PERFORM t_section('05 - business rules');

    -- ======== SRS 4.8: a ticket is admitted once ============================
    INSERT INTO check_in_records (ticket_id, event_id, checked_in_by, device_label)
    VALUES ('60000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001',
            '10000000-0000-4000-8000-000000000003', 'Scanner-01');
    UPDATE tickets SET status = 'checked_in', checked_in_at = now()
    WHERE id = '60000000-0000-4000-8000-000000000001';
    PERFORM t_ok(true, 'the first scan checks the attendee in');

    PERFORM t_throws($q$
        INSERT INTO check_in_records (ticket_id, event_id, checked_in_by)
        VALUES ('60000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001',
                '10000000-0000-4000-8000-000000000003') $q$,
        'the same ticket cannot be checked in twice', '23505');

    -- an authorised reversal releases the ticket for a legitimate re-scan
    UPDATE check_in_records
       SET reversed_at = now(), reversed_by = '10000000-0000-4000-8000-000000000001',
           reversal_reason = 'scanned by mistake'
     WHERE ticket_id = '60000000-0000-4000-8000-000000000001';
    UPDATE tickets SET status = 'valid', checked_in_at = NULL
     WHERE id = '60000000-0000-4000-8000-000000000001';

    INSERT INTO check_in_records (ticket_id, event_id, checked_in_by)
    VALUES ('60000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001',
            '10000000-0000-4000-8000-000000000003');
    PERFORM t_ok(true, 'after a reversal the ticket can be checked in again');

    PERFORM t_eq((SELECT count(*)::int FROM check_in_records
                  WHERE ticket_id = '60000000-0000-4000-8000-000000000001'), 2,
        'both the reversed and the active check-in are retained for the audit trail');
    PERFORM t_eq((SELECT count(*)::int FROM check_in_records
                  WHERE ticket_id = '60000000-0000-4000-8000-000000000001'
                    AND reversed_at IS NULL), 1,
        'exactly one check-in is active');

    -- ======== SRS 4.3.1: a seat is sold once =================================
    INSERT INTO order_items (id, order_id, ticket_type_id, quantity, unit_price_kzt,
                             line_total_kzt, seat_id, seat_section, seat_row, seat_number)
    VALUES ('51000000-0000-4000-8000-000000000001', '50000000-0000-4000-8000-000000000001',
            '40000000-0000-4000-8000-000000000002', 1, 5000, 5000,
            '20000000-0000-4000-8000-000000000004', 'Parterre', 'A', '1');

    INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                         qr_token, status, seat_id, seat_section, seat_row, seat_number)
    VALUES ('61000000-0000-4000-8000-000000000001', 'BF-TKT-000002',
            '50000000-0000-4000-8000-000000000001', '51000000-0000-4000-8000-000000000001',
            '30000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000002',
            'TKT_SEATTICKET0001', 'valid',
            '20000000-0000-4000-8000-000000000004', 'Parterre', 'A', '1');
    PERFORM t_ok(true, 'seat A1 is sold to the first order');

    PERFORM t_throws($q$
        INSERT INTO tickets (ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                             qr_token, status, seat_id)
        VALUES ('BF-TKT-000003', '50000000-0000-4000-8000-000000000001',
                '51000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001',
                '40000000-0000-4000-8000-000000000002', 'TKT_SEATTICKET0002', 'valid',
                '20000000-0000-4000-8000-000000000004') $q$,
        'a second live ticket for the same seat is rejected', '23505');

    -- refunding the ticket returns the seat to the pool
    UPDATE tickets SET status = 'refunded' WHERE id = '61000000-0000-4000-8000-000000000001';
    INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                         qr_token, status, seat_id)
    VALUES ('61000000-0000-4000-8000-000000000002', 'BF-TKT-000004',
            '50000000-0000-4000-8000-000000000001', '51000000-0000-4000-8000-000000000001',
            '30000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000002',
            'TKT_SEATTICKET0003', 'valid', '20000000-0000-4000-8000-000000000004');
    PERFORM t_ok(true, 'a refunded ticket releases its seat for resale');

    -- ======== SRS 4.3.1: one active hold per seat ============================
    INSERT INTO seat_holds (id, seat_id, event_id, session_token, expires_at)
    VALUES ('52000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000005',
            '30000000-0000-4000-8000-000000000001', 'sess-a', now() + interval '10 minutes');

    PERFORM t_throws($q$
        INSERT INTO seat_holds (seat_id, event_id, session_token, expires_at)
        VALUES ('20000000-0000-4000-8000-000000000005', '30000000-0000-4000-8000-000000000001',
                'sess-b', now() + interval '10 minutes') $q$,
        'a second attendee cannot hold a seat that is already held', '23505');

    UPDATE seat_holds SET status = 'expired', released_at = now()
     WHERE id = '52000000-0000-4000-8000-000000000001';
    INSERT INTO seat_holds (seat_id, event_id, session_token, expires_at)
    VALUES ('20000000-0000-4000-8000-000000000005', '30000000-0000-4000-8000-000000000001',
            'sess-b', now() + interval '10 minutes');
    PERFORM t_ok(true, 'an expired hold releases the seat to the next attendee');

    -- ======== SRS 4.14: a campaign QR is never an admission QR ===============
    PERFORM t_throws($q$
        INSERT INTO tickets (ticket_code, order_id, order_item_id, event_id, ticket_type_id, qr_token)
        VALUES ('BF-TKT-000009', '50000000-0000-4000-8000-000000000001',
                '50000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000001',
                '40000000-0000-4000-8000-000000000002', 'CMP_STUDENT10DEMO') $q$,
        'a campaign token cannot be stored as an admission ticket token', '23514');

    PERFORM t_throws($q$
        INSERT INTO campaigns (event_id, name, discount_type, discount_value, qr_token)
        VALUES ('30000000-0000-4000-8000-000000000001', 'Fake', 'percentage', 5, 'TKT_DEMOTICKET0001') $q$,
        'an admission token cannot be stored as a campaign token', '23514');

    PERFORM t_eq(
        (SELECT count(*)::int FROM tickets t JOIN campaigns c ON c.qr_token = t.qr_token),
        0, 'the ticket and campaign QR namespaces never overlap');
    PERFORM t_eq(
        (SELECT count(*)::int FROM tickets WHERE qr_token NOT LIKE 'TKT\_%'),
        0, 'every admission token carries the TKT_ prefix');
    PERFORM t_eq(
        (SELECT count(*)::int FROM campaigns WHERE qr_token NOT LIKE 'CMP\_%'),
        0, 'every campaign token carries the CMP_ prefix');

    -- ======== SRS 4.14: redemption limits ====================================
    PERFORM t_throws($q$
        INSERT INTO promo_redemptions (campaign_id, promo_code_id, order_id, discount_kzt)
        VALUES ('70000000-0000-4000-8000-000000000001', '70000000-0000-4000-8000-000000000002',
                '50000000-0000-4000-8000-000000000001', 500) $q$,
        'one order can redeem a campaign only once', '23505');

    PERFORM t_throws($q$
        UPDATE campaigns SET redemption_count = 51
        WHERE id = '70000000-0000-4000-8000-000000000001' $q$,
        'redemptions cannot exceed the campaign maximum', '23514');

    -- ======== SRS 4.5: the activation checklist ==============================
    PERFORM t_throws($q$
        INSERT INTO paid_sales_activations (event_id, organizer_profile_id, status, activated_at)
        VALUES ('30000000-0000-4000-8000-000000000001', '80000000-0000-4000-8000-000000000001',
                'active', now()) $q$,
        'paid sales cannot go active with an incomplete checklist', '23514');

    INSERT INTO payments (id, purpose, event_id, payer_user_id, amount_kzt, status, paid_at)
    VALUES ('53000000-0000-4000-8000-000000000001', 'paid_sales_activation',
            '30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001',
            10000, 'succeeded', now());

    INSERT INTO paid_sales_activations (
        event_id, organizer_profile_id, payout_account_id, activation_fee_kzt,
        activation_payment_id, status, identity_verified_at, payout_verified_at,
        terms_accepted_at, activated_at)
    VALUES ('30000000-0000-4000-8000-000000000001', '80000000-0000-4000-8000-000000000001',
            '80000000-0000-4000-8000-000000000002', 10000,
            '53000000-0000-4000-8000-000000000001', 'active', now(), now(), now(), now());
    PERFORM t_ok(true, 'a complete checklist activates paid sales for the event');

    PERFORM t_throws($q$
        INSERT INTO paid_sales_activations (event_id, organizer_profile_id)
        VALUES ('30000000-0000-4000-8000-000000000001', '80000000-0000-4000-8000-000000000001') $q$,
        'activation applies to one event only, once', '23505');

    -- ======== SRS 4.16: the audit trail is append-only =======================
    INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description)
    VALUES ('30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001',
            'event.published', 'event', '30000000-0000-4000-8000-000000000001',
            'Organizer published the event')
    RETURNING id INTO v_audit_id;
    PERFORM t_ok(v_audit_id IS NOT NULL, 'an audit entry can be appended');

    PERFORM t_throws(
        format('UPDATE audit_logs SET description = %L WHERE id = %s', 'tampered', v_audit_id),
        'an audit entry cannot be edited', 'P0001');
    PERFORM t_throws(
        format('DELETE FROM audit_logs WHERE id = %s', v_audit_id),
        'an audit entry cannot be deleted', 'P0001');
    PERFORM t_eq((SELECT description FROM audit_logs WHERE id = v_audit_id),
        'Organizer published the event', 'the original audit entry is intact');

    -- ======== SRS 4.8: an Event Admin only sees assigned events ==============
    PERFORM t_eq(
        (SELECT count(*)::int FROM staff_assignments
          WHERE user_id = '10000000-0000-4000-8000-000000000003' AND revoked_at IS NULL),
        1, 'the scanner account is assigned to exactly one event');
END;
$$;

ROLLBACK;
