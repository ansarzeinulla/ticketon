-- =============================================================================
-- BiletFlow demo data.
--
-- Not loaded automatically. Run it with:  make seed
-- Re-runnable: it clears its own rows first, so it never duplicates.
--
-- Contents: 2 organizers, 3 attendees, 1 scanner, 1 support agent,
--           a predefined venue layout (3 sections, 11 rows, 126 seats),
--           1 published free event and 1 published assigned-seating paid event
--           with a campaign, orders, tickets, a check-in, a refund,
--           a support case and an activity timeline.
--
-- All demo rows live in the d0000000-... UUID space.
-- =============================================================================

BEGIN;

-- -----------------------------------------------------------------------------
-- Clear a previous load (children go with their parents through ON DELETE CASCADE).
-- audit_logs is append-only and is therefore never deleted - see the guarded
-- insert at the end of this file.
-- -----------------------------------------------------------------------------
-- A row is cleared if it lives in the demo id space *or* points at a demo
-- event - a real order placed against a demo event while testing has to go
-- before the event it references can be removed, and its own id is random.
-- refunds and tickets RESTRICT order/event deletion, so they are cleared first;
-- order_items, attendees, payments, campaigns, ticket_types and check-ins all
-- carry ON DELETE CASCADE and go with their parents.
DELETE FROM refunds
 WHERE order_id IN (SELECT id FROM orders
                     WHERE id::text LIKE 'd0000000%' OR event_id::text LIKE 'd0000000%');
DELETE FROM tickets       WHERE event_id::text LIKE 'd0000000%';
DELETE FROM support_cases WHERE id::text       LIKE 'd0000000%';
DELETE FROM orders        WHERE id::text LIKE 'd0000000%' OR event_id::text LIKE 'd0000000%';
DELETE FROM events        WHERE id::text       LIKE 'd0000000%';
DELETE FROM venues        WHERE id::text       LIKE 'd0000000%';
DELETE FROM notifications WHERE user_id::text  LIKE 'd0000000%';
-- Demo users are refreshed in place, never deleted: a demo account (dana) may
-- own events a tester created by hand, and events RESTRICT that delete. The
-- INSERTs below upsert users, roles, profiles and payout accounts instead.

-- -----------------------------------------------------------------------------
-- 1. People
-- -----------------------------------------------------------------------------
-- Every demo account shares one password: biletflow-demo
--
-- The hash below is a real bcrypt digest at cost 12, the same cost the API
-- uses. It was previously a placeholder string, which meant `make seed`
-- produced seven accounts that looked usable and could not be signed into -
-- every login returned 401 and there was no documented password anywhere.
--
-- This is demonstration data for a local database, so a shared, published
-- password is the point: it is what makes the dataset usable. Nothing here
-- should ever reach an environment that matters.
INSERT INTO users (id, email, password_hash, full_name, phone, locale, status, email_verified_at) VALUES
    ('d0000000-0000-4000-8000-000000000001', 'dana@biletflow.kz',    '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Dana Amirova',    '+7 701 111 11 11', 'kk', 'active', now() - interval '90 days'),
    ('d0000000-0000-4000-8000-000000000002', 'timur@biletflow.kz',   '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Timur Bekov',     '+7 701 222 22 22', 'ru', 'active', now() - interval '60 days'),
    ('d0000000-0000-4000-8000-000000000003', 'nurlan@example.kz',    '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Nurlan Sagyndyk', '+7 705 333 33 33', 'kk', 'active', now() - interval '30 days'),
    ('d0000000-0000-4000-8000-000000000004', 'aigerim@example.kz',   '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Aigerim Zhaksy',  '+7 705 444 44 44', 'ru', 'active', now() - interval '25 days'),
    ('d0000000-0000-4000-8000-000000000005', 'olzhas@example.kz',    '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Olzhas Serik',    '+7 705 555 55 55', 'en', 'active', now() - interval '20 days'),
    ('d0000000-0000-4000-8000-000000000006', 'scanner@biletflow.kz', '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Askar Kassym',    '+7 707 666 66 66', 'kk', 'active', now() - interval '15 days'),
    ('d0000000-0000-4000-8000-000000000007', 'support@biletflow.kz', '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Sofia Ivanova',   '+7 707 777 77 77', 'ru', 'active', now() - interval '80 days')
ON CONFLICT (id) DO UPDATE SET
    email             = EXCLUDED.email,
    password_hash     = EXCLUDED.password_hash,
    full_name         = EXCLUDED.full_name,
    phone             = EXCLUDED.phone,
    locale            = EXCLUDED.locale,
    status            = EXCLUDED.status,
    email_verified_at = EXCLUDED.email_verified_at;

INSERT INTO user_roles (user_id, role) VALUES
    ('d0000000-0000-4000-8000-000000000001', 'organizer'),
    ('d0000000-0000-4000-8000-000000000001', 'attendee'),
    ('d0000000-0000-4000-8000-000000000002', 'organizer'),
    ('d0000000-0000-4000-8000-000000000003', 'attendee'),
    ('d0000000-0000-4000-8000-000000000004', 'attendee'),
    ('d0000000-0000-4000-8000-000000000005', 'attendee'),
    ('d0000000-0000-4000-8000-000000000006', 'event_admin'),
    ('d0000000-0000-4000-8000-000000000007', 'support_staff'),
    ('d0000000-0000-4000-8000-000000000007', 'platform_admin')
ON CONFLICT (user_id, role) DO NOTHING;

INSERT INTO organizer_profiles (id, user_id, display_name, legal_name, contact_email, description, identity_verified_at) VALUES
    ('d0000000-0000-4000-8000-000000000101', 'd0000000-0000-4000-8000-000000000001',
     'Dana Events', 'IP Amirova D.', 'dana@biletflow.kz',
     'Independent organizer of student and community events in Almaty.', now() - interval '80 days'),
    ('d0000000-0000-4000-8000-000000000102', 'd0000000-0000-4000-8000-000000000002',
     'AITU Student Union', 'AITU Student Union', 'timur@biletflow.kz',
     'Student union running free campus events.', NULL)
ON CONFLICT (id) DO UPDATE SET
    display_name         = EXCLUDED.display_name,
    legal_name           = EXCLUDED.legal_name,
    contact_email        = EXCLUDED.contact_email,
    description          = EXCLUDED.description,
    identity_verified_at = EXCLUDED.identity_verified_at;

INSERT INTO payout_accounts (id, organizer_profile_id, provider, provider_account_ref, masked_account, status, is_simulated, verified_at) VALUES
    ('d0000000-0000-4000-8000-000000000111', 'd0000000-0000-4000-8000-000000000101',
     'simulated', 'sim_acct_dana_0001', '**** **** **** 4242', 'verified', true, now() - interval '75 days')
ON CONFLICT (id) DO NOTHING;

-- -----------------------------------------------------------------------------
-- 2. Predefined venue layout (SRS 4.3.1 - one layout is enough for the MVP)
-- -----------------------------------------------------------------------------
INSERT INTO venues (id, name, address_line, city, country_code, latitude, longitude, is_predefined_layout) VALUES
    ('d0000000-0000-4000-8000-000000000201', 'Almaty Demo Hall', 'Abay Avenue 44', 'Almaty', 'KZ', 43.238949, 76.889709, true);

INSERT INTO venue_sections (id, venue_id, name, price_category, display_order) VALUES
    ('d0000000-0000-4000-8000-000000000211', 'd0000000-0000-4000-8000-000000000201', 'Orchestra',      'premium',  1),
    ('d0000000-0000-4000-8000-000000000212', 'd0000000-0000-4000-8000-000000000201', 'Balcony',        'standard', 2),
    ('d0000000-0000-4000-8000-000000000213', 'd0000000-0000-4000-8000-000000000201', 'Accessible Box', 'standard', 3);

-- Orchestra rows A-E, Balcony rows F-J (12 seats each), Accessible Box row K (6 seats).
DO $seed_seats$
DECLARE
    v_row_id  uuid;
    v_section uuid;
    v_label   text;
    v_seats   int;
    v_access  boolean;
    v_order   int := 0;
BEGIN
    FOR v_label, v_section, v_seats, v_access IN
        SELECT * FROM (VALUES
            ('A', 'd0000000-0000-4000-8000-000000000211'::uuid, 12, false),
            ('B', 'd0000000-0000-4000-8000-000000000211'::uuid, 12, false),
            ('C', 'd0000000-0000-4000-8000-000000000211'::uuid, 12, false),
            ('D', 'd0000000-0000-4000-8000-000000000211'::uuid, 12, false),
            ('E', 'd0000000-0000-4000-8000-000000000211'::uuid, 12, false),
            ('F', 'd0000000-0000-4000-8000-000000000212'::uuid, 12, false),
            ('G', 'd0000000-0000-4000-8000-000000000212'::uuid, 12, false),
            ('H', 'd0000000-0000-4000-8000-000000000212'::uuid, 12, false),
            ('I', 'd0000000-0000-4000-8000-000000000212'::uuid, 12, false),
            ('J', 'd0000000-0000-4000-8000-000000000212'::uuid, 12, false),
            ('K', 'd0000000-0000-4000-8000-000000000213'::uuid,  6, true)
        ) AS r(label, section_id, seats, accessible)
    LOOP
        v_order := v_order + 1;
        INSERT INTO seat_rows (section_id, label, display_order)
        VALUES (v_section, v_label, v_order)
        RETURNING id INTO v_row_id;

        INSERT INTO seats (row_id, seat_number, is_accessible, map_x, map_y)
        SELECT v_row_id, n::text, v_access, n * 30, v_order * 30
          FROM generate_series(1, v_seats) AS n;
    END LOOP;
END;
$seed_seats$;

-- -----------------------------------------------------------------------------
-- 3. Events
-- -----------------------------------------------------------------------------
INSERT INTO events (
    id, organizer_id, venue_id, title, slug, description, category, cover_image_url,
    venue_name, venue_address, starts_at, ends_at, timezone, status, visibility,
    seating_mode, capacity, registration_opens_at, registration_closes_at,
    paid_sales_enabled, refund_policy, published_at
) VALUES
-- A free, general-admission event (SRS 3.1: costs the organizer nothing).
('d0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000002',
 'd0000000-0000-4000-8000-000000000201',
 'AITU Open Lecture: Building for Kazakhstan', 'aitu-open-lecture',
 'A free public lecture for students and the wider tech community.', 'education', NULL,
 'Almaty Demo Hall', 'Abay Avenue 44, Almaty',
 now() + interval '14 days', now() + interval '14 days 2 hours', 'Asia/Almaty',
 'published', 'public', 'general_admission', 150,
 now() - interval '10 days', now() + interval '13 days',
 false, 'Free registrations may be cancelled at any time.', now() - interval '10 days'),

-- A paid, assigned-seating event with activation completed (SRS 3.2, 4.3.1).
('d0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000001',
 'd0000000-0000-4000-8000-000000000201',
 'Almaty Winter Jazz Night', 'almaty-winter-jazz-night',
 'An evening of live jazz with assigned seating.', 'music', NULL,
 'Almaty Demo Hall', 'Abay Avenue 44, Almaty',
 now() + interval '45 days', now() + interval '45 days 3 hours', 'Asia/Almaty',
 'published', 'public', 'assigned_seating', 126,
 now() - interval '5 days', now() + interval '44 days',
 true, 'Full refunds up to 7 days before the event.', now() - interval '5 days'),

-- A completed event, so the organizer history view has something to show.
('d0000000-0000-4000-8000-000000000303', 'd0000000-0000-4000-8000-000000000001',
 'd0000000-0000-4000-8000-000000000201',
 'Autumn Poetry Evening', 'autumn-poetry-evening',
 'Past event retained for the organizer history view.', 'arts', NULL,
 'Almaty Demo Hall', 'Abay Avenue 44, Almaty',
 now() - interval '40 days', now() - interval '40 days' + interval '2 hours', 'Asia/Almaty',
 'completed', 'public', 'general_admission', 80,
 now() - interval '70 days', now() - interval '41 days',
 false, NULL, now() - interval '70 days');

INSERT INTO staff_assignments (event_id, user_id, role, assigned_by) VALUES
    ('d0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000006', 'event_admin', 'd0000000-0000-4000-8000-000000000002'),
    ('d0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000006', 'event_admin', 'd0000000-0000-4000-8000-000000000001');

INSERT INTO ticket_types (id, event_id, name, description, price_kzt, quantity_total,
                          quantity_sold, quantity_refunded, max_per_order, sales_start_at,
                          sales_end_at, price_category, display_order) VALUES
    ('d0000000-0000-4000-8000-000000000401', 'd0000000-0000-4000-8000-000000000301',
     'Free Entry', 'General admission, first come first served.', 0, 150, 1, 0, 4,
     now() - interval '10 days', now() + interval '13 days', NULL, 1),
    ('d0000000-0000-4000-8000-000000000402', 'd0000000-0000-4000-8000-000000000302',
     'Orchestra', 'Rows A-E, closest to the stage.', 12000, 60, 2, 0, 6,
     now() - interval '5 days', now() + interval '44 days', 'premium', 1),
    ('d0000000-0000-4000-8000-000000000403', 'd0000000-0000-4000-8000-000000000302',
     'Balcony', 'Rows F-J.', 7000, 60, 0, 1, 6,
     now() - interval '5 days', now() + interval '44 days', 'standard', 2),
    ('d0000000-0000-4000-8000-000000000404', 'd0000000-0000-4000-8000-000000000302',
     'Accessible Box', 'Row K, step-free access.', 7000, 6, 0, 0, 2,
     now() - interval '5 days', now() + interval '44 days', 'standard', 3);

-- -----------------------------------------------------------------------------
-- 4. Paid sales activation for the jazz night (SRS 4.5)
-- -----------------------------------------------------------------------------
INSERT INTO payments (id, purpose, event_id, payer_user_id, amount_kzt, status, provider,
                      provider_payment_ref, is_simulated, paid_at) VALUES
    ('d0000000-0000-4000-8000-000000000501', 'paid_sales_activation',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000001',
     10000, 'succeeded', 'simulated', 'sim_activation_0001', true, now() - interval '6 days');

INSERT INTO paid_sales_activations (
    event_id, organizer_profile_id, payout_account_id, activation_fee_kzt,
    activation_payment_id, status, identity_verified_at, payout_verified_at,
    terms_accepted_at, activated_at
) VALUES (
    'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000101',
    'd0000000-0000-4000-8000-000000000111', 10000,
    'd0000000-0000-4000-8000-000000000501', 'active',
    now() - interval '7 days', now() - interval '7 days',
    now() - interval '6 days', now() - interval '6 days');

-- -----------------------------------------------------------------------------
-- 5. Campaign and promo code (SRS 4.14)
-- -----------------------------------------------------------------------------
INSERT INTO campaigns (id, event_id, name, discount_type, discount_value, starts_at, ends_at,
                       max_redemptions, redemption_count, status, qr_token, created_by) VALUES
    ('d0000000-0000-4000-8000-000000000601', 'd0000000-0000-4000-8000-000000000302',
     'Student Discount 15%', 'percentage', 15,
     now() - interval '5 days', now() + interval '40 days',
     100, 1, 'active', 'CMP_STUDENT15ALMATYJAZZ', 'd0000000-0000-4000-8000-000000000001');

INSERT INTO promo_codes (id, campaign_id, code) VALUES
    ('d0000000-0000-4000-8000-000000000602', 'd0000000-0000-4000-8000-000000000601', 'STUDENT15');

-- -----------------------------------------------------------------------------
-- 6. Orders and tickets
-- -----------------------------------------------------------------------------

-- 6a. Free registration, already checked in (SRS 4.4 + 4.8).
INSERT INTO orders (id, order_number, event_id, buyer_user_id, buyer_email, buyer_name, buyer_phone,
                    status, subtotal_kzt, discount_kzt, processing_fee_kzt, total_kzt,
                    placed_at, completed_at) VALUES
    ('d0000000-0000-4000-8000-000000000701', 'BF-2026-000001',
     'd0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000003',
     'nurlan@example.kz', 'Nurlan Sagyndyk', '+7 705 333 33 33',
     'completed', 0, 0, 0, 0, now() - interval '9 days', now() - interval '9 days');

INSERT INTO order_items (id, order_id, ticket_type_id, quantity, unit_price_kzt, discount_kzt, line_total_kzt) VALUES
    ('d0000000-0000-4000-8000-000000000711', 'd0000000-0000-4000-8000-000000000701',
     'd0000000-0000-4000-8000-000000000401', 1, 0, 0, 0);

INSERT INTO attendees (id, order_id, user_id, full_name, email, phone) VALUES
    ('d0000000-0000-4000-8000-000000000721', 'd0000000-0000-4000-8000-000000000701',
     'd0000000-0000-4000-8000-000000000003', 'Nurlan Sagyndyk', 'nurlan@example.kz', '+7 705 333 33 33');

INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                     attendee_id, qr_token, status, issued_at) VALUES
    ('d0000000-0000-4000-8000-000000000731', 'BF-TKT-2026-000001',
     'd0000000-0000-4000-8000-000000000701', 'd0000000-0000-4000-8000-000000000711',
     'd0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000401',
     'd0000000-0000-4000-8000-000000000721', 'TKT_DEMOFREE0000000001', 'valid',
     now() - interval '9 days');

-- 6b. Paid order with the promo code applied, two assigned Orchestra seats.
--     24000 subtotal - 3600 (15%) + 720 processing = 21120 KZT.
INSERT INTO orders (id, order_number, event_id, buyer_user_id, buyer_email, buyer_name, buyer_phone,
                    status, subtotal_kzt, discount_kzt, processing_fee_kzt, total_kzt,
                    campaign_id, promo_code_id, placed_at, completed_at) VALUES
    ('d0000000-0000-4000-8000-000000000702', 'BF-2026-000002',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000004',
     'aigerim@example.kz', 'Aigerim Zhaksy', '+7 705 444 44 44',
     'paid', 24000, 3600, 720, 21120,
     'd0000000-0000-4000-8000-000000000601', 'd0000000-0000-4000-8000-000000000602',
     now() - interval '4 days', now() - interval '4 days');

INSERT INTO order_items (id, order_id, ticket_type_id, quantity, unit_price_kzt, discount_kzt,
                         line_total_kzt, seat_id, seat_section, seat_row, seat_number)
SELECT ids.item_id, 'd0000000-0000-4000-8000-000000000702', 'd0000000-0000-4000-8000-000000000402',
       1, 12000, 1800, 10200, s.id, 'Orchestra', 'A', ids.seat_number
FROM (VALUES
        ('d0000000-0000-4000-8000-000000000712'::uuid, '1'),
        ('d0000000-0000-4000-8000-000000000713'::uuid, '2')
     ) AS ids(item_id, seat_number)
JOIN seat_rows  r ON r.label = 'A' AND r.section_id = 'd0000000-0000-4000-8000-000000000211'
JOIN seats      s ON s.row_id = r.id AND s.seat_number = ids.seat_number;

INSERT INTO attendees (id, order_id, user_id, full_name, email, phone) VALUES
    ('d0000000-0000-4000-8000-000000000722', 'd0000000-0000-4000-8000-000000000702',
     'd0000000-0000-4000-8000-000000000004', 'Aigerim Zhaksy', 'aigerim@example.kz', '+7 705 444 44 44'),
    ('d0000000-0000-4000-8000-000000000723', 'd0000000-0000-4000-8000-000000000702',
     NULL, 'Marat Zhaksy', 'aigerim@example.kz', NULL);

INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                     attendee_id, qr_token, status, seat_id, seat_section, seat_row, seat_number, issued_at)
SELECT t.ticket_id, t.ticket_code, 'd0000000-0000-4000-8000-000000000702', oi.id,
       'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000402',
       t.attendee_id, t.qr_token, 'valid', oi.seat_id, oi.seat_section, oi.seat_row, oi.seat_number,
       now() - interval '4 days'
FROM (VALUES
        ('d0000000-0000-4000-8000-000000000732'::uuid, 'BF-TKT-2026-000002',
         'd0000000-0000-4000-8000-000000000712'::uuid,
         'd0000000-0000-4000-8000-000000000722'::uuid, 'TKT_DEMOSEAT00000000A1'),
        ('d0000000-0000-4000-8000-000000000733'::uuid, 'BF-TKT-2026-000003',
         'd0000000-0000-4000-8000-000000000713'::uuid,
         'd0000000-0000-4000-8000-000000000723'::uuid, 'TKT_DEMOSEAT00000000A2')
     ) AS t(ticket_id, ticket_code, order_item_id, attendee_id, qr_token)
JOIN order_items oi ON oi.id = t.order_item_id;

INSERT INTO payments (id, purpose, order_id, payer_user_id, amount_kzt, status, provider,
                      provider_payment_ref, is_simulated, paid_at) VALUES
    ('d0000000-0000-4000-8000-000000000502', 'ticket_order', 'd0000000-0000-4000-8000-000000000702',
     'd0000000-0000-4000-8000-000000000004', 21120, 'succeeded', 'simulated',
     'sim_pay_0002', true, now() - interval '4 days');

INSERT INTO promo_redemptions (campaign_id, promo_code_id, order_id, user_id, discount_kzt, redeemed_at) VALUES
    ('d0000000-0000-4000-8000-000000000601', 'd0000000-0000-4000-8000-000000000602',
     'd0000000-0000-4000-8000-000000000702', 'd0000000-0000-4000-8000-000000000004',
     3600, now() - interval '4 days');

-- 6c. A refunded Balcony order (SRS 4.9), so the dashboard has a refund to show.
INSERT INTO orders (id, order_number, event_id, buyer_user_id, buyer_email, buyer_name,
                    status, subtotal_kzt, discount_kzt, processing_fee_kzt, total_kzt, refunded_kzt,
                    placed_at, completed_at, cancelled_at) VALUES
    ('d0000000-0000-4000-8000-000000000703', 'BF-2026-000003',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000005',
     'olzhas@example.kz', 'Olzhas Serik',
     'refunded', 7000, 0, 210, 7210, 7210,
     now() - interval '3 days', now() - interval '3 days', now() - interval '1 day');

INSERT INTO order_items (id, order_id, ticket_type_id, quantity, unit_price_kzt, discount_kzt,
                         line_total_kzt, seat_id, seat_section, seat_row, seat_number)
SELECT 'd0000000-0000-4000-8000-000000000714', 'd0000000-0000-4000-8000-000000000703',
       'd0000000-0000-4000-8000-000000000403', 1, 7000, 0, 7000, s.id, 'Balcony', 'F', '1'
FROM seat_rows r
JOIN seats s ON s.row_id = r.id AND s.seat_number = '1'
WHERE r.label = 'F' AND r.section_id = 'd0000000-0000-4000-8000-000000000212';

INSERT INTO attendees (id, order_id, user_id, full_name, email) VALUES
    ('d0000000-0000-4000-8000-000000000724', 'd0000000-0000-4000-8000-000000000703',
     'd0000000-0000-4000-8000-000000000005', 'Olzhas Serik', 'olzhas@example.kz');

INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                     attendee_id, qr_token, status, seat_id, seat_section, seat_row, seat_number,
                     issued_at, cancelled_at)
SELECT 'd0000000-0000-4000-8000-000000000734', 'BF-TKT-2026-000004',
       'd0000000-0000-4000-8000-000000000703', oi.id,
       'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000403',
       'd0000000-0000-4000-8000-000000000724', 'TKT_DEMOSEAT00000000F1', 'refunded',
       oi.seat_id, oi.seat_section, oi.seat_row, oi.seat_number,
       now() - interval '3 days', now() - interval '1 day'
FROM order_items oi WHERE oi.id = 'd0000000-0000-4000-8000-000000000714';

INSERT INTO payments (id, purpose, order_id, payer_user_id, amount_kzt, status, provider,
                      provider_payment_ref, is_simulated, paid_at) VALUES
    ('d0000000-0000-4000-8000-000000000503', 'ticket_order', 'd0000000-0000-4000-8000-000000000703',
     'd0000000-0000-4000-8000-000000000005', 7210, 'refunded', 'simulated',
     'sim_pay_0003', true, now() - interval '3 days');

INSERT INTO refunds (id, payment_id, order_id, amount_kzt, status, reason, initiated_by,
                     provider_refund_ref, is_simulated, processed_at) VALUES
    ('d0000000-0000-4000-8000-000000000801', 'd0000000-0000-4000-8000-000000000503',
     'd0000000-0000-4000-8000-000000000703', 7210, 'succeeded',
     'Attendee could not attend', 'd0000000-0000-4000-8000-000000000001',
     'sim_refund_0001', true, now() - interval '1 day');

-- -----------------------------------------------------------------------------
-- 7. Check-in (SRS 4.8)
-- -----------------------------------------------------------------------------
INSERT INTO check_in_records (id, ticket_id, event_id, checked_in_by, checked_in_at, device_label) VALUES
    ('d0000000-0000-4000-8000-000000000901', 'd0000000-0000-4000-8000-000000000731',
     'd0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000006',
     now() - interval '8 days', 'Scanner-01');

UPDATE tickets SET status = 'checked_in', checked_in_at = now() - interval '8 days'
 WHERE id = 'd0000000-0000-4000-8000-000000000731';

-- -----------------------------------------------------------------------------
-- 8. Support case (SRS 4.13)
-- -----------------------------------------------------------------------------
INSERT INTO support_cases (id, case_number, kind, category, status, subject, requester_user_id,
                           assigned_to_user_id, event_id, order_id, ticket_id, last_message_at) VALUES
    ('d0000000-0000-4000-8000-000000000a01', 'SC-2026-0001', 'attendee', 'ticket_delivery',
     'in_progress', 'My ticket email never arrived',
     'd0000000-0000-4000-8000-000000000004', 'd0000000-0000-4000-8000-000000000007',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000702',
     'd0000000-0000-4000-8000-000000000732', now() - interval '2 days');

INSERT INTO support_messages (support_case_id, sender_user_id, body, is_internal_note, created_at) VALUES
    ('d0000000-0000-4000-8000-000000000a01', 'd0000000-0000-4000-8000-000000000004',
     'Hello, I paid for two seats but never received the ticket email.', false, now() - interval '3 days'),
    ('d0000000-0000-4000-8000-000000000a01', 'd0000000-0000-4000-8000-000000000007',
     'Thank you, we have re-sent both tickets. Please also check your spam folder.', false, now() - interval '2 days'),
    ('d0000000-0000-4000-8000-000000000a01', 'd0000000-0000-4000-8000-000000000007',
     'Delivery bounced once on the first attempt; retried successfully.', true, now() - interval '2 days');

-- -----------------------------------------------------------------------------
-- 9. Notifications (SRS 4.10)
-- -----------------------------------------------------------------------------
INSERT INTO notifications (user_id, recipient_email, channel, type, subject, event_id, order_id,
                           ticket_id, status, sent_at) VALUES
    ('d0000000-0000-4000-8000-000000000003', 'nurlan@example.kz', 'email', 'ticket_delivery',
     'Your ticket for AITU Open Lecture', 'd0000000-0000-4000-8000-000000000301',
     'd0000000-0000-4000-8000-000000000701', 'd0000000-0000-4000-8000-000000000731',
     'sent', now() - interval '9 days'),
    ('d0000000-0000-4000-8000-000000000004', 'aigerim@example.kz', 'email', 'order_confirmation',
     'Your order BF-2026-000002 is confirmed', 'd0000000-0000-4000-8000-000000000302',
     'd0000000-0000-4000-8000-000000000702', NULL, 'sent', now() - interval '4 days'),
    ('d0000000-0000-4000-8000-000000000005', 'olzhas@example.kz', 'email', 'refund_completed',
     'Your refund has been processed', 'd0000000-0000-4000-8000-000000000302',
     'd0000000-0000-4000-8000-000000000703', NULL, 'sent', now() - interval '1 day');

-- -----------------------------------------------------------------------------
-- 10. Activity timeline (SRS 4.16)
--     audit_logs is append-only, so these are inserted only once.
-- -----------------------------------------------------------------------------
INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description, metadata, created_at)
SELECT * FROM (VALUES
    ('d0000000-0000-4000-8000-000000000302'::uuid, 'd0000000-0000-4000-8000-000000000001'::uuid,
     'event.published', 'event', 'd0000000-0000-4000-8000-000000000302',
     'Organizer published Almaty Winter Jazz Night', '{"demo": true}'::jsonb, now() - interval '5 days'),
    ('d0000000-0000-4000-8000-000000000302'::uuid, 'd0000000-0000-4000-8000-000000000001'::uuid,
     'paid_sales.activated', 'event', 'd0000000-0000-4000-8000-000000000302',
     'Paid sales activated after the 10 000 KZT activation fee', '{"demo": true}'::jsonb, now() - interval '6 days'),
    ('d0000000-0000-4000-8000-000000000302'::uuid, 'd0000000-0000-4000-8000-000000000001'::uuid,
     'campaign.created', 'campaign', 'd0000000-0000-4000-8000-000000000601',
     'Created campaign Student Discount 15%', '{"demo": true}'::jsonb, now() - interval '5 days'),
    ('d0000000-0000-4000-8000-000000000301'::uuid, 'd0000000-0000-4000-8000-000000000006'::uuid,
     'ticket.checked_in', 'ticket', 'd0000000-0000-4000-8000-000000000731',
     'Nurlan Sagyndyk checked in at the door', '{"demo": true}'::jsonb, now() - interval '8 days'),
    ('d0000000-0000-4000-8000-000000000302'::uuid, 'd0000000-0000-4000-8000-000000000001'::uuid,
     'order.refunded', 'order', 'd0000000-0000-4000-8000-000000000703',
     'Full refund of 7 210 KZT issued', '{"demo": true}'::jsonb, now() - interval '1 day')
) AS v(event_id, actor_user_id, action, entity_type, entity_id, description, metadata, created_at)
WHERE NOT EXISTS (SELECT 1 FROM audit_logs WHERE metadata @> '{"demo": true}'::jsonb);

COMMIT;

\echo 'Demo data loaded.'
