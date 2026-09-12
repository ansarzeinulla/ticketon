-- =============================================================================
-- BiletFlow - 02_schema.sql
-- Core relational schema for the SRS "Core Data Entities" (section 6).
--
-- This script is IDEMPOTENT: running it twice against the same database is a
-- no-op, so it can be pasted into DBeaver / pgAdmin and executed by hand.
-- =============================================================================

SET client_min_messages TO WARNING;

-- -----------------------------------------------------------------------------
-- 0. Helpers
-- -----------------------------------------------------------------------------

-- Keeps updated_at honest without relying on the application layer.
-- clock_timestamp() rather than now(): now() is frozen at transaction start, so
-- an insert+update inside one transaction would leave updated_at unchanged.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger LANGUAGE plpgsql AS $fn$
BEGIN
    NEW.updated_at := clock_timestamp();
    RETURN NEW;
END;
$fn$;

-- SRS 4.16: "Audit entries shall not be editable or deletable".
CREATE OR REPLACE FUNCTION reject_audit_mutation()
RETURNS trigger LANGUAGE plpgsql AS $fn$
BEGIN
    RAISE EXCEPTION 'audit_logs rows are append-only (attempted %)', TG_OP
        USING ERRCODE = 'raise_exception';
END;
$fn$;

-- Attaches the updated_at trigger to a table (drop-then-create = idempotent).
CREATE OR REPLACE FUNCTION attach_updated_at(p_table regclass)
RETURNS void LANGUAGE plpgsql AS $fn$
BEGIN
    EXECUTE format('DROP TRIGGER IF EXISTS trg_set_updated_at ON %s', p_table);
    EXECUTE format(
        'CREATE TRIGGER trg_set_updated_at BEFORE UPDATE ON %s
         FOR EACH ROW EXECUTE FUNCTION set_updated_at()', p_table);
END;
$fn$;

-- -----------------------------------------------------------------------------
-- 1. Enumerated types
-- -----------------------------------------------------------------------------

DO $enums$
DECLARE
    t record;
BEGIN
    FOR t IN
        SELECT * FROM (VALUES
            ('user_role',            $$'attendee','organizer','event_admin','support_staff','platform_admin'$$),
            ('user_status',          $$'pending_verification','active','suspended','deactivated'$$),
            ('event_status',         $$'draft','published','unpublished','cancelled','completed'$$),
            ('event_visibility',     $$'public','unlisted','private'$$),
            ('seating_mode',         $$'general_admission','assigned_seating'$$),
            ('order_status',         $$'pending','awaiting_payment','paid','completed','cancelled','refunded','partially_refunded','expired','failed'$$),
            ('ticket_status',        $$'valid','checked_in','cancelled','refunded'$$),
            ('payment_purpose',      $$'ticket_order','paid_sales_activation'$$),
            ('payment_status',       $$'pending','succeeded','failed','refunded','partially_refunded'$$),
            ('refund_status',        $$'pending','succeeded','failed'$$),
            ('payout_account_status',$$'unverified','pending_review','verified','rejected'$$),
            ('activation_status',    $$'not_started','in_progress','active','suspended'$$),
            ('seat_hold_status',     $$'active','converted','released','expired'$$),
            ('staff_role',           $$'event_admin','support_staff','manager'$$),
            ('support_case_kind',    $$'attendee','organizer'$$),
            ('support_case_status',  $$'open','in_progress','waiting_for_customer','resolved'$$),
            ('support_case_category',$$'ticket_delivery','payment','refund','seating','event_information','check_in','account','technical'$$),
            ('campaign_status',      $$'draft','active','disabled','exhausted','expired'$$),
            ('discount_type',        $$'percentage','fixed_amount'$$),
            ('notification_channel', $$'email','in_app'$$),
            ('notification_status',  $$'pending','sent','failed'$$)
        ) AS v(type_name, labels)
    LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = t.type_name) THEN
            EXECUTE format('CREATE TYPE %I AS ENUM (%s)', t.type_name, t.labels);
        END IF;
    END LOOP;
END;
$enums$;

-- Values added to an enum after it already exists need their own step: the
-- block above only creates types that are missing entirely.
--
-- 'suspended' lets a Platform Admin stop a rogue event selling without
-- destroying it (SRS 4.12: "Suspend users or events"). It is distinct from
-- 'cancelled', which is the organizer calling the event off.
ALTER TYPE event_status ADD VALUE IF NOT EXISTS 'suspended';

-- -----------------------------------------------------------------------------
-- 2. Identity and accounts (SRS 4.1)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS users (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email               citext NOT NULL UNIQUE,
    -- NFR: "Passwords shall be stored using secure password hashing."
    -- Only the hash is ever stored; plaintext must never reach this column.
    password_hash       text   NOT NULL,
    full_name           text   NOT NULL,
    phone               text,
    locale              text   NOT NULL DEFAULT 'kk'
                        CHECK (locale IN ('kk', 'ru', 'en')),
    status              user_status NOT NULL DEFAULT 'pending_verification',
    email_verified_at   timestamptz,
    last_login_at       timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_shape_chk CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'),
    CONSTRAINT users_full_name_not_blank_chk CHECK (btrim(full_name) <> '')
);

-- A person is frequently both an attendee and an organizer, so roles are a set.
CREATE TABLE IF NOT EXISTS user_roles (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        user_role NOT NULL,
    granted_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role)
);

CREATE TABLE IF NOT EXISTS organizer_profiles (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name        text NOT NULL,
    legal_name          text,
    contact_email       citext,
    contact_phone       text,
    description         text,
    website_url         text,
    -- Simulated KYC for the academic MVP (SRS 3.2, 8: production KYC excluded).
    identity_verified_at timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizer_display_name_not_blank_chk CHECK (btrim(display_name) <> '')
);

CREATE TABLE IF NOT EXISTS payout_accounts (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_profile_id uuid NOT NULL REFERENCES organizer_profiles(id) ON DELETE CASCADE,
    provider            text NOT NULL DEFAULT 'simulated',
    -- NFR: "Payment-card data shall not be stored directly by the platform."
    -- Only an opaque provider reference and a masked display value are kept.
    provider_account_ref text NOT NULL,
    masked_account      text,
    currency            char(3) NOT NULL DEFAULT 'KZT' CHECK (currency = 'KZT'),
    status              payout_account_status NOT NULL DEFAULT 'unverified',
    is_simulated        boolean NOT NULL DEFAULT true,
    verified_at         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organizer_profile_id, provider, provider_account_ref)
);

-- -----------------------------------------------------------------------------
-- 3. Venues and seating (SRS 4.3.1 - bonus scope, schema ready from day one)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS venues (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text NOT NULL,
    address_line  text NOT NULL,
    city          text NOT NULL DEFAULT 'Almaty',
    country_code  char(2) NOT NULL DEFAULT 'KZ',
    latitude      numeric(9,6),
    longitude     numeric(9,6),
    is_predefined_layout boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS venue_sections (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id       uuid NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    name           text NOT NULL,
    price_category text NOT NULL DEFAULT 'standard',
    display_order  integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (venue_id, name)
);

-- "row" is a reserved word in SQL, hence seat_rows.
CREATE TABLE IF NOT EXISTS seat_rows (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id    uuid NOT NULL REFERENCES venue_sections(id) ON DELETE CASCADE,
    label         text NOT NULL,
    display_order integer NOT NULL DEFAULT 0,
    UNIQUE (section_id, label)
);

CREATE TABLE IF NOT EXISTS seats (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    row_id        uuid NOT NULL REFERENCES seat_rows(id) ON DELETE CASCADE,
    seat_number   text NOT NULL,
    is_accessible boolean NOT NULL DEFAULT false,
    is_available  boolean NOT NULL DEFAULT true,
    -- Coordinates for the SVG/Canvas seat map.
    map_x         numeric(8,2),
    map_y         numeric(8,2),
    UNIQUE (row_id, seat_number)
);

-- -----------------------------------------------------------------------------
-- 4. Events (SRS 4.2)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS events (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id           uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    venue_id               uuid REFERENCES venues(id) ON DELETE SET NULL,
    title                  text NOT NULL,
    slug                   citext NOT NULL UNIQUE,
    description            text,
    category               text,
    cover_image_url        text,
    -- Venue snapshot: an event keeps its address even if the venue row changes.
    venue_name             text,
    venue_address          text,
    starts_at              timestamptz NOT NULL,
    ends_at                timestamptz NOT NULL,
    -- SRS 4.11: calendar exports must preserve the configured IANA time zone.
    timezone               text NOT NULL DEFAULT 'Asia/Almaty',
    status                 event_status NOT NULL DEFAULT 'draft',
    visibility             event_visibility NOT NULL DEFAULT 'public',
    seating_mode           seating_mode NOT NULL DEFAULT 'general_admission',
    capacity               integer,
    registration_opens_at  timestamptz,
    registration_closes_at timestamptz,
    -- Flipped on only by a completed paid-sales activation (SRS 4.5).
    paid_sales_enabled     boolean NOT NULL DEFAULT false,
    refund_policy          text,
    published_at           timestamptz,
    cancelled_at           timestamptz,
    -- SRS 4.16: duplicating a past event into a new draft.
    duplicated_from_event_id uuid REFERENCES events(id) ON DELETE SET NULL,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT events_time_order_chk CHECK (ends_at > starts_at),
    CONSTRAINT events_registration_window_chk
        CHECK (registration_closes_at IS NULL
               OR registration_opens_at IS NULL
               OR registration_closes_at > registration_opens_at),
    CONSTRAINT events_capacity_chk CHECK (capacity IS NULL OR capacity > 0),
    CONSTRAINT events_title_not_blank_chk CHECK (btrim(title) <> ''),
    CONSTRAINT events_assigned_seating_needs_venue_chk
        CHECK (seating_mode = 'general_admission' OR venue_id IS NOT NULL)
);

-- SRS 4.8: an Event Admin may only see the events they are assigned to.
CREATE TABLE IF NOT EXISTS staff_assignments (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        staff_role NOT NULL DEFAULT 'event_admin',
    assigned_by uuid REFERENCES users(id) ON DELETE SET NULL,
    assigned_at timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz,
    UNIQUE (event_id, user_id, role)
);

CREATE TABLE IF NOT EXISTS ticket_types (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id           uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name               text NOT NULL,
    description        text,
    -- KZT, the settlement currency for the initial release (NFR section 7).
    price_kzt          numeric(14,2) NOT NULL DEFAULT 0,
    quantity_total     integer NOT NULL,
    quantity_sold      integer NOT NULL DEFAULT 0,
    quantity_reserved  integer NOT NULL DEFAULT 0,
    quantity_refunded  integer NOT NULL DEFAULT 0,
    max_per_order      integer NOT NULL DEFAULT 10,
    sales_start_at     timestamptz,
    sales_end_at       timestamptz,
    is_hidden          boolean NOT NULL DEFAULT false,
    price_category     text,
    display_order      integer NOT NULL DEFAULT 0,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    -- A ticket type is "free" exactly when its price is zero (SRS 1.3).
    is_free            boolean GENERATED ALWAYS AS (price_kzt = 0) STORED,
    CONSTRAINT ticket_types_price_chk     CHECK (price_kzt >= 0),
    CONSTRAINT ticket_types_quantity_chk  CHECK (quantity_total >= 0
                                                 AND quantity_sold >= 0
                                                 AND quantity_reserved >= 0
                                                 AND quantity_refunded >= 0),
    -- SRS 4.3: "prevent ticket sales when inventory has been exhausted".
    CONSTRAINT ticket_types_inventory_chk CHECK (quantity_sold + quantity_reserved <= quantity_total),
    CONSTRAINT ticket_types_max_per_order_chk CHECK (max_per_order > 0),
    CONSTRAINT ticket_types_sales_window_chk
        CHECK (sales_end_at IS NULL OR sales_start_at IS NULL OR sales_end_at > sales_start_at),
    UNIQUE (event_id, name)
);

-- -----------------------------------------------------------------------------
-- 5. Promotional campaigns (SRS 4.14) - declared before orders because an order
--    is attributed to the campaign it came from.
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS campaigns (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id          uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name              text NOT NULL,
    discount_type     discount_type NOT NULL,
    discount_value    numeric(14,2) NOT NULL,
    starts_at         timestamptz,
    ends_at           timestamptz,
    max_redemptions   integer,
    redemption_count  integer NOT NULL DEFAULT 0,
    status            campaign_status NOT NULL DEFAULT 'draft',
    -- SRS 4.14: the Campaign QR encodes an OPAQUE token, never a discount value,
    -- and must be functionally distinct from an admission ticket QR. The 'CMP_'
    -- prefix makes that distinction enforceable in the database itself.
    qr_token          text NOT NULL UNIQUE,
    created_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT campaigns_qr_token_prefix_chk CHECK (qr_token ~ '^CMP_[A-Za-z0-9_-]{8,}$'),
    CONSTRAINT campaigns_discount_value_chk  CHECK (discount_value > 0),
    CONSTRAINT campaigns_percentage_range_chk
        CHECK (discount_type <> 'percentage' OR discount_value <= 100),
    CONSTRAINT campaigns_window_chk CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at),
    -- SRS 4.14: "prevent redemption beyond the campaign limit".
    CONSTRAINT campaigns_redemption_limit_chk
        CHECK (max_redemptions IS NULL OR redemption_count <= max_redemptions),
    CONSTRAINT campaigns_max_redemptions_chk CHECK (max_redemptions IS NULL OR max_redemptions > 0)
);

-- Restricts a campaign to specific ticket types; no rows = applies to all.
CREATE TABLE IF NOT EXISTS campaign_ticket_types (
    campaign_id    uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    ticket_type_id uuid NOT NULL REFERENCES ticket_types(id) ON DELETE CASCADE,
    PRIMARY KEY (campaign_id, ticket_type_id)
);

CREATE TABLE IF NOT EXISTS promo_codes (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id  uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    code         citext NOT NULL UNIQUE,
    is_active    boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT promo_codes_code_shape_chk CHECK (code ~ '^[A-Za-z0-9_-]{3,32}$')
);

-- -----------------------------------------------------------------------------
-- 6. Orders, attendees and tickets (SRS 4.4, 4.6, 4.7)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS orders (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number      text NOT NULL UNIQUE,
    event_id          uuid NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    -- Nullable: guest checkout is still an open product decision (SRS 12).
    buyer_user_id     uuid REFERENCES users(id) ON DELETE SET NULL,
    buyer_email       citext NOT NULL,
    buyer_name        text NOT NULL,
    buyer_phone       text,
    status            order_status NOT NULL DEFAULT 'pending',
    currency          char(3) NOT NULL DEFAULT 'KZT' CHECK (currency = 'KZT'),
    subtotal_kzt      numeric(14,2) NOT NULL DEFAULT 0,
    discount_kzt      numeric(14,2) NOT NULL DEFAULT 0,
    processing_fee_kzt numeric(14,2) NOT NULL DEFAULT 0,
    total_kzt         numeric(14,2) NOT NULL DEFAULT 0,
    refunded_kzt      numeric(14,2) NOT NULL DEFAULT 0,
    -- Campaign attribution (SRS 4.14 revenue reporting).
    campaign_id       uuid REFERENCES campaigns(id) ON DELETE SET NULL,
    promo_code_id     uuid REFERENCES promo_codes(id) ON DELETE SET NULL,
    reserved_until    timestamptz,
    placed_at         timestamptz,
    completed_at      timestamptz,
    cancelled_at      timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT orders_amounts_non_negative_chk
        CHECK (subtotal_kzt >= 0 AND discount_kzt >= 0
               AND processing_fee_kzt >= 0 AND total_kzt >= 0 AND refunded_kzt >= 0),
    CONSTRAINT orders_discount_not_above_subtotal_chk CHECK (discount_kzt <= subtotal_kzt),
    -- The server, never the client, computes the total (SRS 4.14 NFR).
    CONSTRAINT orders_total_math_chk
        CHECK (total_kzt = subtotal_kzt - discount_kzt + processing_fee_kzt),
    CONSTRAINT orders_refund_not_above_total_chk CHECK (refunded_kzt <= total_kzt),
    CONSTRAINT orders_promo_needs_campaign_chk
        CHECK (promo_code_id IS NULL OR campaign_id IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS order_items (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    ticket_type_id  uuid NOT NULL REFERENCES ticket_types(id) ON DELETE RESTRICT,
    quantity        integer NOT NULL,
    unit_price_kzt  numeric(14,2) NOT NULL,
    discount_kzt    numeric(14,2) NOT NULL DEFAULT 0,
    line_total_kzt  numeric(14,2) NOT NULL,
    -- SRS 4.3.1: the assigned section/row/seat is stored on the order item.
    seat_id         uuid REFERENCES seats(id) ON DELETE RESTRICT,
    seat_section    text,
    seat_row        text,
    seat_number     text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT order_items_quantity_chk CHECK (quantity > 0),
    CONSTRAINT order_items_price_chk    CHECK (unit_price_kzt >= 0 AND discount_kzt >= 0 AND line_total_kzt >= 0),
    CONSTRAINT order_items_total_math_chk
        CHECK (line_total_kzt = (unit_price_kzt * quantity) - discount_kzt),
    -- An assigned seat is always exactly one ticket.
    CONSTRAINT order_items_seat_single_chk CHECK (seat_id IS NULL OR quantity = 1)
);

-- The person attending, who is not necessarily the account holder who paid.
CREATE TABLE IF NOT EXISTS attendees (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id     uuid REFERENCES users(id) ON DELETE SET NULL,
    full_name   text NOT NULL,
    email       citext NOT NULL,
    phone       text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT attendees_name_not_blank_chk CHECK (btrim(full_name) <> '')
);

CREATE TABLE IF NOT EXISTS tickets (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- SRS 4.7: one identifier shared by the digital and the printed copy.
    ticket_code    text NOT NULL UNIQUE,
    order_id       uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    order_item_id  uuid NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    event_id       uuid NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    ticket_type_id uuid NOT NULL REFERENCES ticket_types(id) ON DELETE RESTRICT,
    attendee_id    uuid REFERENCES attendees(id) ON DELETE SET NULL,
    -- SRS 4.14: an admission QR ('TKT_') and a campaign QR ('CMP_') live in
    -- disjoint token namespaces, so a campaign QR can never scan as admission.
    qr_token       text NOT NULL UNIQUE,
    status         ticket_status NOT NULL DEFAULT 'valid',
    seat_id        uuid REFERENCES seats(id) ON DELETE RESTRICT,
    seat_section   text,
    seat_row       text,
    seat_number    text,
    issued_at      timestamptz NOT NULL DEFAULT now(),
    checked_in_at  timestamptz,
    cancelled_at   timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tickets_qr_token_prefix_chk CHECK (qr_token ~ '^TKT_[A-Za-z0-9_-]{8,}$'),
    CONSTRAINT tickets_checked_in_consistency_chk
        CHECK ((status = 'checked_in') = (checked_in_at IS NOT NULL))
);

-- SRS 4.3.1 / NFR: "prevent two orders from purchasing the same seat, including
-- when multiple attendees check out concurrently". A live ticket (valid or
-- checked_in) takes the seat exclusively; cancelled/refunded tickets release it.
CREATE UNIQUE INDEX IF NOT EXISTS tickets_one_live_ticket_per_seat_uidx
    ON tickets (seat_id)
    WHERE seat_id IS NOT NULL AND status IN ('valid', 'checked_in');

-- SRS 4.3.1: a seat is held while checkout is in progress and the hold expires.
CREATE TABLE IF NOT EXISTS seat_holds (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seat_id     uuid NOT NULL REFERENCES seats(id) ON DELETE CASCADE,
    event_id    uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    order_id    uuid REFERENCES orders(id) ON DELETE CASCADE,
    session_token text,
    status      seat_hold_status NOT NULL DEFAULT 'active',
    held_at     timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    released_at timestamptz,
    CONSTRAINT seat_holds_expiry_chk CHECK (expires_at > held_at)
);

-- At most one ACTIVE hold per seat: the atomic half of the reservation process.
CREATE UNIQUE INDEX IF NOT EXISTS seat_holds_one_active_per_seat_uidx
    ON seat_holds (seat_id)
    WHERE status = 'active';

-- -----------------------------------------------------------------------------
-- 7. Payments, refunds and paid-sales activation (SRS 3.2, 4.5, 4.6, 4.9)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS payments (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose           payment_purpose NOT NULL,
    -- Exactly one of these is set, depending on the purpose.
    order_id          uuid REFERENCES orders(id) ON DELETE CASCADE,
    event_id          uuid REFERENCES events(id) ON DELETE CASCADE,
    payer_user_id     uuid REFERENCES users(id) ON DELETE SET NULL,
    amount_kzt        numeric(14,2) NOT NULL,
    currency          char(3) NOT NULL DEFAULT 'KZT' CHECK (currency = 'KZT'),
    status            payment_status NOT NULL DEFAULT 'pending',
    provider          text NOT NULL DEFAULT 'simulated',
    provider_payment_ref text,
    -- SRS 4.6: "Demonstration payment records shall never be presented as real
    -- financial transactions." Defaults to true; real money is out of MVP scope.
    is_simulated      boolean NOT NULL DEFAULT true,
    failure_reason    text,
    paid_at           timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payments_amount_chk CHECK (amount_kzt >= 0),
    CONSTRAINT payments_target_chk CHECK (
        (purpose = 'ticket_order'          AND order_id IS NOT NULL)
     OR (purpose = 'paid_sales_activation' AND event_id IS NOT NULL AND order_id IS NULL)
    ),
    CONSTRAINT payments_succeeded_needs_timestamp_chk
        CHECK (status <> 'succeeded' OR paid_at IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS refunds (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id        uuid NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
    order_id          uuid NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    amount_kzt        numeric(14,2) NOT NULL,
    status            refund_status NOT NULL DEFAULT 'pending',
    reason            text,
    initiated_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    provider_refund_ref text,
    is_simulated      boolean NOT NULL DEFAULT true,
    processed_at      timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT refunds_amount_chk CHECK (amount_kzt > 0)
);

-- SRS 4.5: activation is per event, and Platform Admins may suspend it.
CREATE TABLE IF NOT EXISTS paid_sales_activations (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id              uuid NOT NULL UNIQUE REFERENCES events(id) ON DELETE CASCADE,
    organizer_profile_id  uuid NOT NULL REFERENCES organizer_profiles(id) ON DELETE RESTRICT,
    payout_account_id     uuid REFERENCES payout_accounts(id) ON DELETE SET NULL,
    activation_fee_kzt    numeric(14,2) NOT NULL DEFAULT 0,
    activation_payment_id uuid REFERENCES payments(id) ON DELETE SET NULL,
    status                activation_status NOT NULL DEFAULT 'not_started',
    identity_verified_at  timestamptz,
    payout_verified_at    timestamptz,
    terms_accepted_at     timestamptz,
    activated_at          timestamptz,
    suspended_at          timestamptz,
    suspended_by          uuid REFERENCES users(id) ON DELETE SET NULL,
    suspension_reason     text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT activations_fee_chk CHECK (activation_fee_kzt >= 0),
    -- An event only becomes 'active' once every checklist item is satisfied.
    CONSTRAINT activations_checklist_chk CHECK (
        status <> 'active' OR (
            identity_verified_at IS NOT NULL
            AND payout_verified_at IS NOT NULL
            AND terms_accepted_at IS NOT NULL
            AND activation_payment_id IS NOT NULL
            AND activated_at IS NOT NULL
        )
    ),
    CONSTRAINT activations_suspended_chk
        CHECK (status <> 'suspended' OR suspended_at IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS promo_redemptions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id    uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    promo_code_id  uuid NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    order_id       uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id        uuid REFERENCES users(id) ON DELETE SET NULL,
    discount_kzt   numeric(14,2) NOT NULL,
    redeemed_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT promo_redemptions_discount_chk CHECK (discount_kzt >= 0),
    -- One redemption per order keeps campaign counters exact (SRS 4.14).
    UNIQUE (campaign_id, order_id)
);

-- -----------------------------------------------------------------------------
-- 8. Check-in (SRS 4.8)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS check_in_records (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id      uuid NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    event_id       uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    checked_in_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    checked_in_at  timestamptz NOT NULL DEFAULT now(),
    device_label   text,
    -- SRS 4.8: an accidental check-in may be reversed by authorized staff.
    reversed_at    timestamptz,
    reversed_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    reversal_reason text,
    CONSTRAINT check_in_reversal_chk
        CHECK ((reversed_at IS NULL AND reversed_by IS NULL)
               OR (reversed_at IS NOT NULL AND reversed_at >= checked_in_at))
);

-- SRS 4.8 / success criteria: "prevent the same ticket from being used twice".
CREATE UNIQUE INDEX IF NOT EXISTS check_in_one_active_per_ticket_uidx
    ON check_in_records (ticket_id)
    WHERE reversed_at IS NULL;

-- -----------------------------------------------------------------------------
-- 9. Support cases (SRS 4.13)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS support_cases (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    case_number       text NOT NULL UNIQUE,
    kind              support_case_kind NOT NULL,
    category          support_case_category NOT NULL,
    status            support_case_status NOT NULL DEFAULT 'open',
    subject           text NOT NULL,
    requester_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_to_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    -- Contextual links captured automatically when available (SRS 4.13).
    event_id          uuid REFERENCES events(id) ON DELETE SET NULL,
    order_id          uuid REFERENCES orders(id) ON DELETE SET NULL,
    ticket_id         uuid REFERENCES tickets(id) ON DELETE SET NULL,
    last_message_at   timestamptz,
    resolved_at       timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT support_cases_subject_not_blank_chk CHECK (btrim(subject) <> ''),
    CONSTRAINT support_cases_resolved_chk
        CHECK (status <> 'resolved' OR resolved_at IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS support_messages (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    support_case_id uuid NOT NULL REFERENCES support_cases(id) ON DELETE CASCADE,
    sender_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    body           text NOT NULL,
    -- Staff-only note: never shown to the requester.
    is_internal_note boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT support_messages_body_not_blank_chk CHECK (btrim(body) <> '')
);

-- -----------------------------------------------------------------------------
-- 10. Notifications and audit trail (SRS 4.10, 4.16)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid REFERENCES users(id) ON DELETE CASCADE,
    recipient_email citext,
    channel     notification_channel NOT NULL DEFAULT 'email',
    type        text NOT NULL,
    subject     text,
    body        text,
    status      notification_status NOT NULL DEFAULT 'pending',
    event_id    uuid REFERENCES events(id) ON DELETE SET NULL,
    order_id    uuid REFERENCES orders(id) ON DELETE SET NULL,
    ticket_id   uuid REFERENCES tickets(id) ON DELETE SET NULL,
    support_case_id uuid REFERENCES support_cases(id) ON DELETE SET NULL,
    sent_at     timestamptz,
    read_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notifications_recipient_chk
        CHECK (user_id IS NOT NULL OR recipient_email IS NOT NULL)
);

-- Append-only. Also serves as the per-event activity timeline (SRS 4.16).
CREATE TABLE IF NOT EXISTS audit_logs (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- Deliberately NO foreign keys: this table is append-only, so a cascading
    -- DELETE or a SET NULL would be blocked by trg_audit_logs_append_only and
    -- would make the referenced row undeletable. The ids are kept as plain
    -- values so the trail survives whatever happens to the operational rows.
    event_id       uuid,
    actor_user_id  uuid,
    action         text NOT NULL,
    entity_type    text NOT NULL,
    entity_id      text,
    description    text,
    metadata       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_logs_action_not_blank_chk CHECK (btrim(action) <> '')
);

DROP TRIGGER IF EXISTS trg_audit_logs_append_only ON audit_logs;
CREATE TRIGGER trg_audit_logs_append_only
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation();

-- -----------------------------------------------------------------------------
-- 11. Indexes for the access paths the SRS actually asks for
-- -----------------------------------------------------------------------------

CREATE INDEX IF NOT EXISTS users_status_idx              ON users (status);
CREATE INDEX IF NOT EXISTS organizer_profiles_user_idx   ON organizer_profiles (user_id);
CREATE INDEX IF NOT EXISTS payout_accounts_organizer_idx ON payout_accounts (organizer_profile_id);

CREATE INDEX IF NOT EXISTS venue_sections_venue_idx      ON venue_sections (venue_id);
CREATE INDEX IF NOT EXISTS seat_rows_section_idx         ON seat_rows (section_id);
CREATE INDEX IF NOT EXISTS seats_row_idx                 ON seats (row_id);

-- Event discovery: upcoming public events, newest first.
CREATE INDEX IF NOT EXISTS events_organizer_idx          ON events (organizer_id);
CREATE INDEX IF NOT EXISTS events_status_starts_at_idx   ON events (status, starts_at);
CREATE INDEX IF NOT EXISTS events_public_upcoming_idx    ON events (starts_at)
    WHERE status = 'published' AND visibility = 'public';

CREATE INDEX IF NOT EXISTS staff_assignments_user_idx    ON staff_assignments (user_id)
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS ticket_types_event_idx        ON ticket_types (event_id);

CREATE INDEX IF NOT EXISTS campaigns_event_idx           ON campaigns (event_id);
CREATE INDEX IF NOT EXISTS promo_codes_campaign_idx      ON promo_codes (campaign_id);
CREATE INDEX IF NOT EXISTS promo_redemptions_campaign_idx ON promo_redemptions (campaign_id);
CREATE INDEX IF NOT EXISTS promo_redemptions_order_idx   ON promo_redemptions (order_id);

CREATE INDEX IF NOT EXISTS orders_event_status_idx       ON orders (event_id, status);
CREATE INDEX IF NOT EXISTS orders_buyer_user_idx         ON orders (buyer_user_id);
CREATE INDEX IF NOT EXISTS orders_buyer_email_idx        ON orders (buyer_email);
-- Sales-over-time charts (SRS 4.15) read order timestamps per event.
CREATE INDEX IF NOT EXISTS orders_event_placed_at_idx    ON orders (event_id, placed_at);
CREATE INDEX IF NOT EXISTS orders_campaign_idx           ON orders (campaign_id)
    WHERE campaign_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS orders_reservation_expiry_idx ON orders (reserved_until)
    WHERE status IN ('pending', 'awaiting_payment');

CREATE INDEX IF NOT EXISTS order_items_order_idx         ON order_items (order_id);
CREATE INDEX IF NOT EXISTS order_items_ticket_type_idx   ON order_items (ticket_type_id);
CREATE INDEX IF NOT EXISTS attendees_order_idx           ON attendees (order_id);
CREATE INDEX IF NOT EXISTS attendees_email_idx           ON attendees (email);

CREATE INDEX IF NOT EXISTS tickets_event_status_idx      ON tickets (event_id, status);
CREATE INDEX IF NOT EXISTS tickets_order_idx             ON tickets (order_id);
CREATE INDEX IF NOT EXISTS tickets_ticket_type_idx       ON tickets (ticket_type_id);
CREATE INDEX IF NOT EXISTS tickets_attendee_idx          ON tickets (attendee_id);

CREATE INDEX IF NOT EXISTS seat_holds_event_idx          ON seat_holds (event_id);
CREATE INDEX IF NOT EXISTS seat_holds_expiry_idx         ON seat_holds (expires_at)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS payments_order_idx            ON payments (order_id);
CREATE INDEX IF NOT EXISTS payments_event_idx            ON payments (event_id);
CREATE INDEX IF NOT EXISTS payments_status_idx           ON payments (status);
CREATE INDEX IF NOT EXISTS refunds_order_idx             ON refunds (order_id);
CREATE INDEX IF NOT EXISTS refunds_payment_idx           ON refunds (payment_id);

CREATE INDEX IF NOT EXISTS check_in_records_event_idx    ON check_in_records (event_id);
CREATE INDEX IF NOT EXISTS check_in_records_ticket_idx   ON check_in_records (ticket_id);

CREATE INDEX IF NOT EXISTS support_cases_requester_idx   ON support_cases (requester_user_id);
CREATE INDEX IF NOT EXISTS support_cases_assignee_idx    ON support_cases (assigned_to_user_id);
CREATE INDEX IF NOT EXISTS support_cases_event_status_idx ON support_cases (event_id, status);
CREATE INDEX IF NOT EXISTS support_messages_case_idx     ON support_messages (support_case_id, created_at);

CREATE INDEX IF NOT EXISTS notifications_user_idx        ON notifications (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS notifications_pending_idx     ON notifications (created_at)
    WHERE status = 'pending';

-- The event activity timeline, filtered by date range and activity type.
CREATE INDEX IF NOT EXISTS audit_logs_event_created_idx  ON audit_logs (event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_actor_idx          ON audit_logs (actor_user_id);
CREATE INDEX IF NOT EXISTS audit_logs_entity_idx         ON audit_logs (entity_type, entity_id);

-- -----------------------------------------------------------------------------
-- 12. updated_at triggers
-- -----------------------------------------------------------------------------

DO $triggers$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY[
        'users', 'organizer_profiles', 'payout_accounts', 'venues', 'events',
        'ticket_types', 'campaigns', 'promo_codes', 'orders', 'attendees',
        'tickets', 'payments', 'refunds', 'paid_sales_activations',
        'support_cases'
    ] LOOP
        PERFORM attach_updated_at(tbl::regclass);
    END LOOP;
END;
$triggers$;

-- -----------------------------------------------------------------------------
-- 12b. Single-use account tokens (SRS 4.1)
-- -----------------------------------------------------------------------------
--
-- Email verification and password reset both need a secret that arrives out of
-- band, is good once, and expires. They share one table because they share
-- every one of those properties; only the purpose and the lifetime differ.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_token_purpose') THEN
        CREATE TYPE user_token_purpose AS ENUM ('email_verification', 'password_reset');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS user_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     user_token_purpose NOT NULL,
    -- The token itself is never stored. What arrives by email is a random
    -- string; this is its SHA-256, so a leaked database backup does not hand
    -- somebody a working password-reset link.
    token_hash  text NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT user_tokens_hash_shape_chk CHECK (char_length(token_hash) = 64),
    CONSTRAINT user_tokens_expiry_chk CHECK (expires_at > created_at)
);

-- Consumption looks the token up by hash, and nothing else ever does.
CREATE UNIQUE INDEX IF NOT EXISTS user_tokens_hash_key ON user_tokens (token_hash);
CREATE INDEX IF NOT EXISTS user_tokens_user_idx ON user_tokens (user_id, purpose);

COMMENT ON TABLE user_tokens IS
    'Single-use email-verification and password-reset tokens; only the hash is stored (SRS 4.1).';

-- -----------------------------------------------------------------------------
-- 12b. Moderation queue and platform settings (SRS 4.12)
-- -----------------------------------------------------------------------------
-- SRS 4.12 asks platform administrators to "review reported events" and to
-- "configure activation fees and platform settings". Both need somewhere to
-- put the data; neither existed.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'event_report_status') THEN
        CREATE TYPE event_report_status AS ENUM ('open', 'reviewing', 'upheld', 'dismissed');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'event_report_reason') THEN
        CREATE TYPE event_report_reason AS ENUM
            ('fraud', 'misleading', 'inappropriate', 'spam', 'copyright', 'other');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS event_reports (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    -- Nullable so a report survives the reporter deleting their account: the
    -- moderation record is about the event, not about who complained.
    reporter_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    reason       event_report_reason NOT NULL,
    details      text,
    status       event_report_status NOT NULL DEFAULT 'open',
    reviewed_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at  timestamptz,
    resolution   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT event_reports_details_length_chk
        CHECK (details IS NULL OR char_length(details) <= 2000),
    -- A decided report has to say who decided it and when, so the queue
    -- cannot contain a resolution nobody is accountable for.
    CONSTRAINT event_reports_reviewed_chk
        CHECK (status IN ('open', 'reviewing')
               OR (reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL))
);

-- One open report per person per event: a reporter clicking twice is the same
-- complaint, not two. A second report is allowed once the first is decided.
CREATE UNIQUE INDEX IF NOT EXISTS event_reports_one_open_per_reporter_uidx
    ON event_reports (event_id, reporter_user_id)
    WHERE status IN ('open', 'reviewing') AND reporter_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS event_reports_queue_idx
    ON event_reports (status, created_at DESC);

SELECT attach_updated_at('event_reports');

COMMENT ON TABLE event_reports IS
    'Attendee reports about events, for the platform moderation queue (SRS 4.12).';

-- Platform settings are a single row keyed by name rather than a wide config
-- table, so adding a setting needs no migration. The value is jsonb so a
-- number stays a number and a flag stays a boolean.
CREATE TABLE IF NOT EXISTS platform_settings (
    key         text PRIMARY KEY,
    value       jsonb NOT NULL,
    description text,
    updated_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT platform_settings_key_shape_chk CHECK (key ~ '^[a-z][a-z0-9_.]{2,63}$')
);

COMMENT ON TABLE platform_settings IS
    'Runtime platform configuration, including the paid-sales activation fee (SRS 4.12).';

-- The activation fee was a Go constant until now, so it could not be changed
-- without a rebuild. Seeded at the value that constant held.
INSERT INTO platform_settings (key, value, description)
VALUES ('activation_fee_kzt', '"5000.00"'::jsonb,
        'One-time paid-sales activation fee per event, in KZT (SRS 3.3).')
ON CONFLICT (key) DO NOTHING;

-- -----------------------------------------------------------------------------
-- 12c. Support attachments (bonus, SRS 4.13)
-- -----------------------------------------------------------------------------
--
-- An attendee reporting "my QR will not scan" is describing a picture. Columns
-- rather than a separate table: a message carries at most one attachment, and
-- a join table for a cardinality of one buys nothing.
--
-- The body is what a message is for, so it stays NOT NULL; a message that is
-- only an attachment carries a caption written by the client.

ALTER TABLE support_messages
    ADD COLUMN IF NOT EXISTS attachment_url       text,
    ADD COLUMN IF NOT EXISTS attachment_filename  text,
    ADD COLUMN IF NOT EXISTS attachment_mime_type text,
    ADD COLUMN IF NOT EXISTS attachment_bytes     bigint;

DO $$
BEGIN
    -- All four columns describe one file, so either all are present or none
    -- are. A half-populated attachment renders as a broken link, and the
    -- database is the right place to make that impossible.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'support_messages_attachment_chk'
    ) THEN
        ALTER TABLE support_messages ADD CONSTRAINT support_messages_attachment_chk CHECK (
            (attachment_url IS NULL
             AND attachment_filename IS NULL
             AND attachment_mime_type IS NULL
             AND attachment_bytes IS NULL)
            OR
            (attachment_url IS NOT NULL
             AND attachment_filename IS NOT NULL
             AND attachment_mime_type IS NOT NULL
             AND attachment_bytes IS NOT NULL
             AND attachment_bytes > 0)
        );
    END IF;
END
$$;

COMMENT ON COLUMN support_messages.attachment_url IS
    'Uploaded file backing this message; all four attachment columns move together.';

-- -----------------------------------------------------------------------------
-- 13. Documentation for the tables a reviewer opens first
-- -----------------------------------------------------------------------------

COMMENT ON TABLE users            IS 'Platform accounts; roles are granted through user_roles (SRS 4.1).';
COMMENT ON TABLE events           IS 'Physical, venue-based events owned by an organizer (SRS 4.2).';
COMMENT ON TABLE ticket_types     IS 'Free and paid ticket tiers with inventory counters (SRS 4.3).';
COMMENT ON TABLE orders           IS 'Checkout basket; total_kzt is computed server-side (SRS 4.6).';
COMMENT ON TABLE tickets          IS 'Issued admission tickets; qr_token always starts with TKT_ (SRS 4.7).';
COMMENT ON TABLE campaigns        IS 'Promotional campaigns; qr_token always starts with CMP_ and is never valid for admission (SRS 4.14).';
COMMENT ON TABLE support_cases    IS 'Asynchronous support conversations with event/order/ticket context (SRS 4.13).';
COMMENT ON TABLE check_in_records IS 'Check-in and reversal history; one active check-in per ticket (SRS 4.8).';
COMMENT ON TABLE audit_logs       IS 'Append-only activity timeline; UPDATE and DELETE are blocked by trigger (SRS 4.16).';

COMMENT ON COLUMN users.password_hash IS 'Hash only - plaintext passwords must never be stored.';
COMMENT ON COLUMN payments.is_simulated IS 'True for demonstration payments; the MVP moves no real money.';
