-- =============================================================================
-- The core end-to-end flow from SRS 13.4, exercised purely at the data layer:
-- free registration -> ticket issue -> check-in -> support case -> refund ->
-- the analytics figures the organizer dashboard (SRS 4.15) must be able to read.
-- =============================================================================
BEGIN;
\ir _helpers.sql
\ir _fixture.sql

DO $$
DECLARE
    v_case_id      uuid;
    v_capacity     int;
    v_sold         int;
    v_gross        numeric;
    v_discounts    numeric;
    v_refunded     numeric;
    v_checked_in   int;
BEGIN
    PERFORM t_section('06 - end-to-end flow');

    -- ======== SRS 4.4: free registration creates a zero-value order ==========
    INSERT INTO orders (id, order_number, event_id, buyer_user_id, buyer_email, buyer_name,
                        status, subtotal_kzt, discount_kzt, processing_fee_kzt, total_kzt,
                        placed_at, completed_at)
    VALUES ('50000000-0000-4000-8000-000000000010', 'BF-TEST-0002',
            '30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002',
            'attendee.fixture@biletflow.test', 'Nurlan Attendee', 'completed', 0, 0, 0, 0, now(), now());

    INSERT INTO order_items (id, order_id, ticket_type_id, quantity, unit_price_kzt, line_total_kzt)
    VALUES ('50000000-0000-4000-8000-000000000011', '50000000-0000-4000-8000-000000000010',
            '40000000-0000-4000-8000-000000000001', 1, 0, 0);

    INSERT INTO attendees (id, order_id, user_id, full_name, email)
    VALUES ('50000000-0000-4000-8000-000000000012', '50000000-0000-4000-8000-000000000010',
            '10000000-0000-4000-8000-000000000002', 'Nurlan Attendee', 'attendee.fixture@biletflow.test');

    INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                         attendee_id, qr_token, status)
    VALUES ('60000000-0000-4000-8000-000000000010', 'BF-TKT-000010',
            '50000000-0000-4000-8000-000000000010', '50000000-0000-4000-8000-000000000011',
            '30000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000001',
            '50000000-0000-4000-8000-000000000012', 'TKT_FREETICKET0001', 'valid');

    UPDATE ticket_types SET quantity_sold = quantity_sold + 1
     WHERE id = '40000000-0000-4000-8000-000000000001';

    PERFORM t_eq((SELECT total_kzt FROM orders WHERE id = '50000000-0000-4000-8000-000000000010'),
        0::numeric, 'a free registration produces a zero-value order');
    PERFORM t_eq((SELECT count(*)::int FROM payments WHERE order_id = '50000000-0000-4000-8000-000000000010'),
        0, 'a free registration takes no payment record');

    -- notification for ticket delivery (SRS 4.10)
    INSERT INTO notifications (user_id, recipient_email, channel, type, subject,
                               event_id, order_id, ticket_id, status, sent_at)
    VALUES ('10000000-0000-4000-8000-000000000002', 'attendee.fixture@biletflow.test', 'email',
            'ticket_delivery', 'Your ticket for BiletFlow Demo Concert',
            '30000000-0000-4000-8000-000000000001', '50000000-0000-4000-8000-000000000010',
            '60000000-0000-4000-8000-000000000010', 'sent', now());
    PERFORM t_eq((SELECT count(*)::int FROM notifications
                  WHERE order_id = '50000000-0000-4000-8000-000000000010'), 1,
        'a ticket-delivery notification is recorded');

    -- ======== SRS 4.8: the Event Admin scans the free ticket =================
    INSERT INTO check_in_records (ticket_id, event_id, checked_in_by, device_label)
    VALUES ('60000000-0000-4000-8000-000000000010', '30000000-0000-4000-8000-000000000001',
            '10000000-0000-4000-8000-000000000003', 'Scanner-01');
    UPDATE tickets SET status = 'checked_in', checked_in_at = now()
     WHERE id = '60000000-0000-4000-8000-000000000010';
    PERFORM t_eq((SELECT status FROM tickets WHERE id = '60000000-0000-4000-8000-000000000010'),
        'checked_in'::ticket_status, 'the attendee is checked in');

    -- ======== SRS 4.13: a contextual support case ============================
    INSERT INTO support_cases (case_number, kind, category, subject, requester_user_id,
                               event_id, order_id, ticket_id, status)
    VALUES ('SC-000001', 'attendee', 'ticket_delivery', 'I did not receive my ticket email',
            '10000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000001',
            '50000000-0000-4000-8000-000000000010', '60000000-0000-4000-8000-000000000010', 'open')
    RETURNING id INTO v_case_id;

    INSERT INTO support_messages (support_case_id, sender_user_id, body) VALUES
        (v_case_id, '10000000-0000-4000-8000-000000000002', 'Hello, my ticket email never arrived.'),
        (v_case_id, '10000000-0000-4000-8000-000000000004', 'We have re-sent it, please check spam.');
    INSERT INTO support_messages (support_case_id, sender_user_id, body, is_internal_note)
    VALUES (v_case_id, '10000000-0000-4000-8000-000000000004', 'Bounced once, retried.', true);

    UPDATE support_cases
       SET status = 'in_progress',
           assigned_to_user_id = '10000000-0000-4000-8000-000000000004',
           last_message_at = now()
     WHERE id = v_case_id;

    PERFORM t_eq((SELECT count(*)::int FROM support_messages WHERE support_case_id = v_case_id), 3,
        'the case thread keeps every message');
    PERFORM t_eq((SELECT count(*)::int FROM support_messages
                  WHERE support_case_id = v_case_id AND is_internal_note = false), 2,
        'internal notes are separable from the attendee-visible thread');
    PERFORM t_eq((SELECT event_id FROM support_cases WHERE id = v_case_id),
        '30000000-0000-4000-8000-000000000001'::uuid,
        'the case carries its event context');

    UPDATE support_cases SET status = 'resolved', resolved_at = now() WHERE id = v_case_id;
    PERFORM t_eq((SELECT status FROM support_cases WHERE id = v_case_id),
        'resolved'::support_case_status, 'the case can be resolved');

    -- a different attendee must not match the "my cases" query
    PERFORM t_eq((SELECT count(*)::int FROM support_cases
                  WHERE requester_user_id = '10000000-0000-4000-8000-000000000001'), 0,
        'a case is only visible through its own requester relationship');

    -- ======== SRS 4.9: refunding the paid order ==============================
    INSERT INTO refunds (payment_id, order_id, amount_kzt, status, reason, initiated_by, processed_at)
    VALUES ('50000000-0000-4000-8000-000000000004', '50000000-0000-4000-8000-000000000001',
            4650, 'succeeded', 'Attendee cancelled', '10000000-0000-4000-8000-000000000001', now());

    UPDATE orders SET status = 'refunded', refunded_kzt = 4650
     WHERE id = '50000000-0000-4000-8000-000000000001';
    UPDATE payments SET status = 'refunded' WHERE id = '50000000-0000-4000-8000-000000000004';
    UPDATE tickets SET status = 'refunded' WHERE id = '60000000-0000-4000-8000-000000000001';
    UPDATE ticket_types SET quantity_sold = quantity_sold - 1, quantity_refunded = quantity_refunded + 1
     WHERE id = '40000000-0000-4000-8000-000000000002';

    PERFORM t_eq((SELECT status FROM tickets WHERE id = '60000000-0000-4000-8000-000000000001'),
        'refunded'::ticket_status, 'a refunded ticket is invalidated');

    -- ======== SRS 4.15: the figures the dashboard reads ======================
    SELECT e.capacity INTO v_capacity FROM events e WHERE e.id = '30000000-0000-4000-8000-000000000001';

    SELECT count(*)::int INTO v_sold FROM tickets
     WHERE event_id = '30000000-0000-4000-8000-000000000001'
       AND status IN ('valid', 'checked_in');

    SELECT COALESCE(sum(total_kzt), 0), COALESCE(sum(discount_kzt), 0), COALESCE(sum(refunded_kzt), 0)
      INTO v_gross, v_discounts, v_refunded
      FROM orders
     WHERE event_id = '30000000-0000-4000-8000-000000000001'
       AND status IN ('paid', 'completed', 'refunded', 'partially_refunded');

    SELECT count(*)::int INTO v_checked_in FROM tickets
     WHERE event_id = '30000000-0000-4000-8000-000000000001' AND status = 'checked_in';

    PERFORM t_eq(v_capacity, 200, 'total capacity');
    PERFORM t_eq(v_sold, 1, 'tickets currently sold (the refunded one no longer counts)');
    PERFORM t_eq(v_capacity - v_sold, 199, 'tickets remaining');
    PERFORM t_eq(v_gross, 4650::numeric, 'gross sales in KZT');
    PERFORM t_eq(v_discounts, 500::numeric, 'discounts in KZT');
    PERFORM t_eq(v_refunded, 4650::numeric, 'refunds in KZT');
    PERFORM t_eq(v_gross - v_refunded, 0::numeric, 'net demonstration revenue in KZT');
    PERFORM t_eq(v_checked_in, 1, 'checked-in attendees');

    -- sales by ticket type
    PERFORM t_eq(
        (SELECT count(*)::int FROM (
            SELECT oi.ticket_type_id, sum(oi.quantity) AS qty
              FROM order_items oi JOIN orders o ON o.id = oi.order_id
             WHERE o.event_id = '30000000-0000-4000-8000-000000000001'
             GROUP BY oi.ticket_type_id) s),
        2, 'sales can be broken down by ticket type');

    -- campaign attribution
    PERFORM t_eq(
        (SELECT count(*)::int FROM orders
          WHERE campaign_id = '70000000-0000-4000-8000-000000000001'),
        1, 'the paid order is attributed to its campaign');
    PERFORM t_eq(
        (SELECT COALESCE(sum(discount_kzt), 0) FROM promo_redemptions
          WHERE campaign_id = '70000000-0000-4000-8000-000000000001'),
        500::numeric, 'the campaign records the exact discount granted');

    -- ======== SRS 4.16: the event activity timeline ==========================
    INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description) VALUES
        ('30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001',
         'event.published', 'event', '30000000-0000-4000-8000-000000000001', 'Event published'),
        ('30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000003',
         'ticket.checked_in', 'ticket', '60000000-0000-4000-8000-000000000010', 'Attendee checked in'),
        ('30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001',
         'order.refunded', 'order', '50000000-0000-4000-8000-000000000001', 'Full refund issued');

    PERFORM t_eq(
        (SELECT count(*)::int FROM audit_logs
          WHERE event_id = '30000000-0000-4000-8000-000000000001'
            AND created_at >= now() - interval '1 hour'), 3,
        'the timeline can be filtered by event and date range');
    PERFORM t_eq(
        (SELECT count(*)::int FROM audit_logs
          WHERE event_id = '30000000-0000-4000-8000-000000000001'
            AND action LIKE 'ticket.%'), 1,
        'the timeline can be filtered by activity type');

    -- ======== SRS 4.16: duplicating a past event =============================
    INSERT INTO events (organizer_id, venue_id, title, slug, starts_at, ends_at, timezone,
                        status, visibility, seating_mode, capacity, duplicated_from_event_id)
    SELECT organizer_id, venue_id, title || ' (2026)', slug || '-2026',
           starts_at + interval '365 days', ends_at + interval '365 days', timezone,
           'draft', visibility, seating_mode, capacity, id
      FROM events WHERE id = '30000000-0000-4000-8000-000000000001';

    PERFORM t_eq(
        (SELECT count(*)::int FROM events
          WHERE duplicated_from_event_id = '30000000-0000-4000-8000-000000000001'), 1,
        'a past event can be duplicated into a new draft');
    PERFORM t_eq(
        (SELECT count(*)::int FROM orders o
          JOIN events e ON e.id = o.event_id
         WHERE e.duplicated_from_event_id = '30000000-0000-4000-8000-000000000001'), 0,
        'the duplicate carries no orders from the original event');
    PERFORM t_eq(
        (SELECT count(*)::int FROM tickets t
          JOIN events e ON e.id = t.event_id
         WHERE e.duplicated_from_event_id = '30000000-0000-4000-8000-000000000001'), 0,
        'the duplicate carries no tickets from the original event');
END;
$$;

ROLLBACK;
