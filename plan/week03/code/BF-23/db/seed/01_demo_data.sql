-- =============================================================================
-- BiletFlow demo data.
--
-- Not loaded automatically. Run it with:  make seed
-- Re-runnable: it deletes its own rows first, so it never duplicates.
--
-- Contents: 2 organizers, 2 attendees, 1 published free event and 1 published
--           paid event, with one order on each.
--
-- All demo rows live in the d0000000-... UUID space.
-- =============================================================================

BEGIN;

-- -----------------------------------------------------------------------------
-- Clear a previous load. Children go first; order_items, attendees and
-- payments follow their orders through ON DELETE CASCADE.
-- -----------------------------------------------------------------------------
DELETE FROM tickets      WHERE id::text LIKE 'd0000000%';
DELETE FROM orders       WHERE id::text LIKE 'd0000000%';
DELETE FROM ticket_types WHERE id::text LIKE 'd0000000%';
DELETE FROM events       WHERE id::text LIKE 'd0000000%';
DELETE FROM users        WHERE id::text LIKE 'd0000000%';

-- -----------------------------------------------------------------------------
-- 1. People
-- -----------------------------------------------------------------------------
-- Every demo account shares one password: biletflow-demo
-- The hash is a real bcrypt digest at cost 12, the same cost the API uses, so
-- these accounts can actually be signed into.
INSERT INTO users (id, email, password_hash, full_name, phone, locale, status, email_verified_at) VALUES
    ('d0000000-0000-4000-8000-000000000001', 'dana@biletflow.kz',  '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Dana Amirova',    '+7 701 111 11 11', 'kk', 'active', now() - interval '90 days'),
    ('d0000000-0000-4000-8000-000000000002', 'timur@biletflow.kz', '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Timur Bekov',     '+7 701 222 22 22', 'ru', 'active', now() - interval '60 days'),
    ('d0000000-0000-4000-8000-000000000003', 'nurlan@example.kz',  '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Nurlan Sagyndyk', '+7 705 333 33 33', 'kk', 'active', now() - interval '30 days'),
    ('d0000000-0000-4000-8000-000000000004', 'aigerim@example.kz', '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Aigerim Zhaksy',  '+7 705 444 44 44', 'ru', 'active', now() - interval '25 days');

INSERT INTO user_roles (user_id, role) VALUES
    ('d0000000-0000-4000-8000-000000000001', 'organizer'),
    ('d0000000-0000-4000-8000-000000000001', 'attendee'),
    ('d0000000-0000-4000-8000-000000000002', 'organizer'),
    ('d0000000-0000-4000-8000-000000000003', 'attendee'),
    ('d0000000-0000-4000-8000-000000000004', 'attendee');

-- -----------------------------------------------------------------------------
-- 2. Events
-- -----------------------------------------------------------------------------
INSERT INTO events (
    id, organizer_id, title, slug, description, category,
    venue_name, venue_address, starts_at, ends_at, timezone, status, visibility,
    seating_mode, capacity, registration_opens_at, registration_closes_at,
    refund_policy, published_at
) VALUES
-- A free event (SRS 3.1: publishing and free tickets cost the organizer nothing).
('d0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000002',
 'AITU Open Lecture: Building for Kazakhstan', 'aitu-open-lecture',
 'A free public lecture for students and the wider tech community.', 'education',
 'Almaty Demo Hall', 'Abay Avenue 44, Almaty',
 now() + interval '14 days', now() + interval '14 days 2 hours', 'Asia/Almaty',
 'published', 'public', 'general_admission', 150,
 now() - interval '10 days', now() + interval '13 days',
 'Free registrations may be cancelled at any time.', now() - interval '10 days'),

-- A paid event, general admission.
('d0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000001',
 'Almaty Winter Jazz Night', 'almaty-winter-jazz-night',
 'An evening of live jazz.', 'music',
 'Almaty Demo Hall', 'Abay Avenue 44, Almaty',
 now() + interval '45 days', now() + interval '45 days 3 hours', 'Asia/Almaty',
 'published', 'public', 'general_admission', 126,
 now() - interval '5 days', now() + interval '44 days',
 'Full refunds up to 7 days before the event.', now() - interval '5 days');

INSERT INTO ticket_types (id, event_id, name, description, price_kzt, quantity_total,
                          quantity_sold, max_per_order, sales_start_at, sales_end_at,
                          display_order) VALUES
    ('d0000000-0000-4000-8000-000000000401', 'd0000000-0000-4000-8000-000000000301',
     'Free Entry', 'General admission, first come first served.', 0, 150, 1, 4,
     now() - interval '10 days', now() + interval '13 days', 1),
    ('d0000000-0000-4000-8000-000000000402', 'd0000000-0000-4000-8000-000000000302',
     'VIP', 'Front tables, closest to the stage.', 12000, 20, 0, 6,
     now() - interval '5 days', now() + interval '44 days', 1),
    ('d0000000-0000-4000-8000-000000000403', 'd0000000-0000-4000-8000-000000000302',
     'Standard', 'General admission.', 7000, 100, 2, 6,
     now() - interval '5 days', now() + interval '44 days', 2);

-- -----------------------------------------------------------------------------
-- 3. Orders and tickets
-- -----------------------------------------------------------------------------

-- 3a. A free registration.
INSERT INTO orders (id, order_number, event_id, buyer_user_id, buyer_email, buyer_name, buyer_phone,
                    status, subtotal_kzt, discount_kzt, processing_fee_kzt, total_kzt,
                    placed_at, completed_at) VALUES
    ('d0000000-0000-4000-8000-000000000701', 'BF-2026-000001',
     'd0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000003',
     'nurlan@example.kz', 'Nurlan Sagyndyk', '+7 705 333 33 33',
     'paid', 0, 0, 0, 0, now() - interval '9 days', now() - interval '9 days');

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

-- 3b. A paid order: two Standard tickets, 2 x 7000 = 14000 KZT, simulated payment.
INSERT INTO orders (id, order_number, event_id, buyer_user_id, buyer_email, buyer_name, buyer_phone,
                    status, subtotal_kzt, discount_kzt, processing_fee_kzt, total_kzt,
                    placed_at, completed_at) VALUES
    ('d0000000-0000-4000-8000-000000000702', 'BF-2026-000002',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000004',
     'aigerim@example.kz', 'Aigerim Zhaksy', '+7 705 444 44 44',
     'paid', 14000, 0, 0, 14000, now() - interval '4 days', now() - interval '4 days');

INSERT INTO order_items (id, order_id, ticket_type_id, quantity, unit_price_kzt, discount_kzt, line_total_kzt) VALUES
    ('d0000000-0000-4000-8000-000000000712', 'd0000000-0000-4000-8000-000000000702',
     'd0000000-0000-4000-8000-000000000403', 2, 7000, 0, 14000);

INSERT INTO attendees (id, order_id, user_id, full_name, email, phone) VALUES
    ('d0000000-0000-4000-8000-000000000722', 'd0000000-0000-4000-8000-000000000702',
     'd0000000-0000-4000-8000-000000000004', 'Aigerim Zhaksy', 'aigerim@example.kz', '+7 705 444 44 44');

INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                     attendee_id, qr_token, status, issued_at) VALUES
    ('d0000000-0000-4000-8000-000000000732', 'BF-TKT-2026-000002',
     'd0000000-0000-4000-8000-000000000702', 'd0000000-0000-4000-8000-000000000712',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000403',
     'd0000000-0000-4000-8000-000000000722', 'TKT_DEMOPAID0000000001', 'valid',
     now() - interval '4 days'),
    ('d0000000-0000-4000-8000-000000000733', 'BF-TKT-2026-000003',
     'd0000000-0000-4000-8000-000000000702', 'd0000000-0000-4000-8000-000000000712',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000403',
     'd0000000-0000-4000-8000-000000000722', 'TKT_DEMOPAID0000000002', 'valid',
     now() - interval '4 days');

INSERT INTO payments (id, purpose, order_id, payer_user_id, amount_kzt, status, provider,
                      provider_payment_ref, is_simulated, paid_at) VALUES
    ('d0000000-0000-4000-8000-000000000741', 'ticket_order',
     'd0000000-0000-4000-8000-000000000702', 'd0000000-0000-4000-8000-000000000004',
     14000, 'succeeded', 'simulated', 'sim_BF-2026-000002', true, now() - interval '4 days');

COMMIT;
