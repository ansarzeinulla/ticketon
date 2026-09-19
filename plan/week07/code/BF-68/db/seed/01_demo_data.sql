-- =============================================================================
-- BiletFlow demo data.
--
-- Not loaded automatically. Run it with:  make seed
-- Re-runnable: it deletes its own rows first, so it never duplicates.
--
-- Contents: 2 organizers with profiles, 2 attendees, 1 door scanner assigned to
--           both events, 1 support agent who is also the platform
--           administrator, 1 published free event and 1 published paid event
--           whose paid sales are activated, with one order on each, a
--           15% student campaign on the paid one and a support case. The free ticket has already been used
--           at the door.
--
-- All demo rows live in the d0000000-... UUID space.
-- =============================================================================

BEGIN;

-- -----------------------------------------------------------------------------
-- Clear a previous load. Children go first; order_items, attendees and
-- payments follow their orders through ON DELETE CASCADE.
-- -----------------------------------------------------------------------------
DELETE FROM support_cases WHERE id::text LIKE 'd0000000%';
DELETE FROM tickets      WHERE id::text LIKE 'd0000000%';
DELETE FROM orders       WHERE id::text LIKE 'd0000000%';
DELETE FROM paid_sales_activations WHERE event_id::text LIKE 'd0000000%';
DELETE FROM payments     WHERE id::text LIKE 'd0000000%';
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
    ('d0000000-0000-4000-8000-000000000004', 'aigerim@example.kz', '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Aigerim Zhaksy',  '+7 705 444 44 44', 'ru', 'active', now() - interval '25 days'),
    ('d0000000-0000-4000-8000-000000000006', 'scanner@biletflow.kz', '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Askar Kassym', '+7 707 666 66 66', 'kk', 'active', now() - interval '15 days'),
    ('d0000000-0000-4000-8000-000000000007', 'support@biletflow.kz', '$2a$12$rFCQOCmARbTQxdRMSB7wde/xelwF29B8BnCrEk5tyV3fsUvI8NI06', 'Sofia Ivanova', '+7 707 777 77 77', 'ru', 'active', now() - interval '80 days');

INSERT INTO user_roles (user_id, role) VALUES
    ('d0000000-0000-4000-8000-000000000001', 'organizer'),
    ('d0000000-0000-4000-8000-000000000001', 'attendee'),
    ('d0000000-0000-4000-8000-000000000002', 'organizer'),
    ('d0000000-0000-4000-8000-000000000003', 'attendee'),
    ('d0000000-0000-4000-8000-000000000004', 'attendee'),
    ('d0000000-0000-4000-8000-000000000006', 'event_admin'),
    ('d0000000-0000-4000-8000-000000000007', 'support_staff'),
    ('d0000000-0000-4000-8000-000000000007', 'platform_admin');

INSERT INTO organizer_profiles (id, user_id, display_name, legal_name, contact_email, description, identity_verified_at) VALUES
    ('d0000000-0000-4000-8000-000000000101', 'd0000000-0000-4000-8000-000000000001',
     'Dana Events', 'IP Amirova D.', 'dana@biletflow.kz',
     'Independent organizer of student and community events in Almaty.', now() - interval '80 days'),
    ('d0000000-0000-4000-8000-000000000102', 'd0000000-0000-4000-8000-000000000002',
     'AITU Student Union', 'AITU Student Union', 'timur@biletflow.kz',
     'Student union running free campus events.', NULL);

INSERT INTO payout_accounts (id, organizer_profile_id, provider, provider_account_ref, masked_account, status, is_simulated, verified_at) VALUES
    ('d0000000-0000-4000-8000-000000000111', 'd0000000-0000-4000-8000-000000000101',
     'simulated', 'sim_acct_dana_0001', '**** **** **** 4242', 'verified', true, now() - interval '75 days');

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

-- The paid event has completed the activation checklist (SRS 4.5), so its paid
-- tickets are on sale.
UPDATE events SET paid_sales_enabled = true WHERE id = 'd0000000-0000-4000-8000-000000000302';

INSERT INTO payments (id, purpose, event_id, payer_user_id, amount_kzt, status, provider,
                      provider_payment_ref, is_simulated, paid_at) VALUES
    ('d0000000-0000-4000-8000-000000000501', 'paid_sales_activation',
     'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000001',
     5000, 'succeeded', 'simulated', 'sim_activation_0001', true, now() - interval '6 days');

INSERT INTO paid_sales_activations (
    event_id, organizer_profile_id, payout_account_id, activation_fee_kzt,
    activation_payment_id, status, identity_verified_at, payout_verified_at,
    terms_accepted_at, activated_at
) VALUES (
    'd0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000101',
    'd0000000-0000-4000-8000-000000000111', 5000,
    'd0000000-0000-4000-8000-000000000501', 'active',
    now() - interval '7 days', now() - interval '7 days',
    now() - interval '6 days', now() - interval '6 days');

-- Door staff (SRS 4.8): the scanner account may check attendees in at both
-- events, each assigned by that event's organizer.
INSERT INTO staff_assignments (event_id, user_id, role, assigned_by) VALUES
    ('d0000000-0000-4000-8000-000000000301', 'd0000000-0000-4000-8000-000000000006', 'event_admin', 'd0000000-0000-4000-8000-000000000002'),
    ('d0000000-0000-4000-8000-000000000302', 'd0000000-0000-4000-8000-000000000006', 'event_admin', 'd0000000-0000-4000-8000-000000000001');

-- Campaign and promo code (SRS 4.14): typed as STUDENT15 or scanned from a
-- poster as CMP_STUDENT15ALMATYJAZZ.
INSERT INTO campaigns (id, event_id, name, discount_type, discount_value, starts_at, ends_at,
                       max_redemptions, redemption_count, status, qr_token, created_by) VALUES
    ('d0000000-0000-4000-8000-000000000601', 'd0000000-0000-4000-8000-000000000302',
     'Student Discount 15%', 'percentage', 15,
     now() - interval '5 days', now() + interval '40 days',
     100, 0, 'active', 'CMP_STUDENT15ALMATYJAZZ', 'd0000000-0000-4000-8000-000000000001');

INSERT INTO promo_codes (id, campaign_id, code) VALUES
    ('d0000000-0000-4000-8000-000000000602', 'd0000000-0000-4000-8000-000000000601', 'STUDENT15');

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

-- The free ticket has already been through the door, so the scanner has a
-- repeat entry to refuse (5.md).
INSERT INTO check_in_records (ticket_id, event_id, checked_in_by, checked_in_at, device_label) VALUES
    ('d0000000-0000-4000-8000-000000000731', 'd0000000-0000-4000-8000-000000000301',
     'd0000000-0000-4000-8000-000000000006', now() - interval '2 hours', 'ios scanner');
UPDATE tickets SET status = 'checked_in', checked_in_at = now() - interval '2 hours'
 WHERE id = 'd0000000-0000-4000-8000-000000000731';

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

-- -----------------------------------------------------------------------------
-- 4. Support case (SRS 4.13)
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
     'Hello, I paid for two tickets but never received the ticket email.', false, now() - interval '3 days'),
    ('d0000000-0000-4000-8000-000000000a01', 'd0000000-0000-4000-8000-000000000007',
     'Thank you, we have re-sent both tickets. Please also check your spam folder.', false, now() - interval '2 days'),
    ('d0000000-0000-4000-8000-000000000a01', 'd0000000-0000-4000-8000-000000000007',
     'Delivery bounced once on the first attempt; retried successfully.', true, now() - interval '2 days');

COMMIT;
