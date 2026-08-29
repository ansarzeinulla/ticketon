-- =============================================================================
-- A small, deterministic dataset used by the constraint and business-rule
-- tests. Fixed UUIDs make the assertions readable. Always loaded inside a
-- transaction that is rolled back afterwards.
--
--   users    10000000-0000-4000-8000-00000000000x
--   venue    20000000-...                          (2 seats: A1, A2)
--   event    30000000-...
--   tickets  40000000-... ticket types, 60000000-... issued ticket
--   order    50000000-...
--   campaign 70000000-...
--   organizer profile / payout account 80000000-...
-- =============================================================================

INSERT INTO users (id, email, password_hash, full_name, status, email_verified_at) VALUES
    ('10000000-0000-4000-8000-000000000001', 'organizer.fixture@biletflow.test', 'hash', 'Dana Organizer', 'active', now()),
    ('10000000-0000-4000-8000-000000000002', 'attendee.fixture@biletflow.test',  'hash', 'Nurlan Attendee', 'active', now()),
    ('10000000-0000-4000-8000-000000000003', 'scanner.fixture@biletflow.test',   'hash', 'Askar Scanner',  'active', now()),
    ('10000000-0000-4000-8000-000000000004', 'support.fixture@biletflow.test',   'hash', 'Sofia Support',  'active', now());

INSERT INTO user_roles (user_id, role) VALUES
    ('10000000-0000-4000-8000-000000000001', 'organizer'),
    ('10000000-0000-4000-8000-000000000002', 'attendee'),
    ('10000000-0000-4000-8000-000000000003', 'event_admin'),
    ('10000000-0000-4000-8000-000000000004', 'support_staff');

INSERT INTO organizer_profiles (id, user_id, display_name, contact_email, identity_verified_at) VALUES
    ('80000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001',
     'Dana Events', 'organizer.fixture@biletflow.test', now());

INSERT INTO payout_accounts (id, organizer_profile_id, provider_account_ref, masked_account, status, verified_at) VALUES
    ('80000000-0000-4000-8000-000000000002', '80000000-0000-4000-8000-000000000001',
     'sim_acct_0001', '**** 4242', 'verified', now());

INSERT INTO venues (id, name, address_line, city, is_predefined_layout) VALUES
    ('20000000-0000-4000-8000-000000000001', 'Almaty Demo Hall', 'Abay Ave 1', 'Almaty', true);

INSERT INTO venue_sections (id, venue_id, name, price_category) VALUES
    ('20000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000001', 'Parterre', 'standard');

INSERT INTO seat_rows (id, section_id, label) VALUES
    ('20000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000002', 'A');

INSERT INTO seats (id, row_id, seat_number, is_accessible) VALUES
    ('20000000-0000-4000-8000-000000000004', '20000000-0000-4000-8000-000000000003', '1', false),
    ('20000000-0000-4000-8000-000000000005', '20000000-0000-4000-8000-000000000003', '2', true);

INSERT INTO events (
    id, organizer_id, venue_id, title, slug, description, category,
    venue_name, venue_address, starts_at, ends_at, timezone,
    status, visibility, seating_mode, capacity, paid_sales_enabled, published_at
) VALUES (
    '30000000-0000-4000-8000-000000000001',
    '10000000-0000-4000-8000-000000000001',
    '20000000-0000-4000-8000-000000000001',
    'BiletFlow Demo Concert', 'biletflow-demo-concert',
    'Fixture event used by the automated database tests.', 'music',
    'Almaty Demo Hall', 'Abay Ave 1, Almaty',
    now() + interval '30 days', now() + interval '30 days 3 hours', 'Asia/Almaty',
    'published', 'public', 'general_admission', 200, true, now()
);

INSERT INTO staff_assignments (event_id, user_id, role, assigned_by) VALUES
    ('30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000003',
     'event_admin', '10000000-0000-4000-8000-000000000001');

INSERT INTO ticket_types (id, event_id, name, price_kzt, quantity_total, quantity_sold, max_per_order) VALUES
    ('40000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', 'Free Entry', 0,     100, 0, 4),
    ('40000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000001', 'Standard',   5000,  100, 1, 6);

INSERT INTO campaigns (id, event_id, name, discount_type, discount_value,
                       max_redemptions, redemption_count, status, qr_token, created_by) VALUES
    ('70000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001',
     'Student Promo', 'percentage', 10, 50, 1, 'active', 'CMP_STUDENT10DEMO',
     '10000000-0000-4000-8000-000000000001');

INSERT INTO promo_codes (id, campaign_id, code) VALUES
    ('70000000-0000-4000-8000-000000000002', '70000000-0000-4000-8000-000000000001', 'STUDENT10');

INSERT INTO orders (
    id, order_number, event_id, buyer_user_id, buyer_email, buyer_name, status,
    subtotal_kzt, discount_kzt, processing_fee_kzt, total_kzt,
    campaign_id, promo_code_id, placed_at, completed_at
) VALUES (
    '50000000-0000-4000-8000-000000000001', 'BF-TEST-0001',
    '30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002',
    'attendee.fixture@biletflow.test', 'Nurlan Attendee', 'paid',
    5000, 500, 150, 4650,
    '70000000-0000-4000-8000-000000000001', '70000000-0000-4000-8000-000000000002',
    now(), now()
);

INSERT INTO order_items (id, order_id, ticket_type_id, quantity, unit_price_kzt, discount_kzt, line_total_kzt) VALUES
    ('50000000-0000-4000-8000-000000000002', '50000000-0000-4000-8000-000000000001',
     '40000000-0000-4000-8000-000000000002', 1, 5000, 500, 4500);

INSERT INTO attendees (id, order_id, user_id, full_name, email) VALUES
    ('50000000-0000-4000-8000-000000000003', '50000000-0000-4000-8000-000000000001',
     '10000000-0000-4000-8000-000000000002', 'Nurlan Attendee', 'attendee.fixture@biletflow.test');

INSERT INTO promo_redemptions (campaign_id, promo_code_id, order_id, user_id, discount_kzt) VALUES
    ('70000000-0000-4000-8000-000000000001', '70000000-0000-4000-8000-000000000002',
     '50000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 500);

INSERT INTO payments (id, purpose, order_id, payer_user_id, amount_kzt, status,
                      provider, provider_payment_ref, is_simulated, paid_at) VALUES
    ('50000000-0000-4000-8000-000000000004', 'ticket_order', '50000000-0000-4000-8000-000000000001',
     '10000000-0000-4000-8000-000000000002', 4650, 'succeeded', 'simulated', 'sim_pay_0001', true, now());

INSERT INTO tickets (id, ticket_code, order_id, order_item_id, event_id, ticket_type_id,
                     attendee_id, qr_token, status) VALUES
    ('60000000-0000-4000-8000-000000000001', 'BF-TKT-000001',
     '50000000-0000-4000-8000-000000000001', '50000000-0000-4000-8000-000000000002',
     '30000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000002',
     '50000000-0000-4000-8000-000000000003', 'TKT_DEMOTICKET0001', 'valid');
