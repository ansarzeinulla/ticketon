/**
 * Mirrors of the JSON the Phase 2 Go API returns. Field names match the Go
 * struct tags exactly, so a response can be used without any remapping.
 */

export type UserRole =
  | "attendee"
  | "organizer"
  | "event_admin"
  | "support_staff"
  | "platform_admin";

export type UserStatus =
  | "pending_verification"
  | "active"
  | "suspended"
  | "deactivated";

export interface User {
  id: string;
  email: string;
  full_name: string;
  phone?: string;
  locale: string;
  status: UserStatus;
  roles: UserRole[];
  email_verified_at?: string;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export type EventStatus =
  | "draft"
  | "published"
  | "unpublished"
  | "cancelled"
  | "completed"
  | "suspended";

export type EventVisibility = "public" | "unlisted" | "private";
export type SeatingMode = "general_admission" | "assigned_seating";

export interface BiletEvent {
  id: string;
  organizer_id: string;
  venue_id?: string;
  title: string;
  slug: string;
  description?: string;
  category?: string;
  cover_image_url?: string;
  venue_name?: string;
  venue_address?: string;
  starts_at: string;
  ends_at: string;
  timezone: string;
  status: EventStatus;
  visibility: EventVisibility;
  seating_mode: SeatingMode;
  capacity?: number;
  registration_opens_at?: string;
  registration_closes_at?: string;
  paid_sales_enabled: boolean;
  refund_policy?: string;
  published_at?: string;
  cancelled_at?: string;
  /** Set when this event was created by duplicating another (SRS 4.16). */
  duplicated_from_event_id?: string;
  created_at: string;
  updated_at: string;
  /** Derived server-side: Upcoming / Active / Completed / Cancelled (SRS 4.16). */
  lifecycle: Lifecycle;
}

/** POST /auth/register and POST /auth/login both return this. */
/**
 * What the Go API returns on sign-in.
 *
 * Server-side only now: the route handlers unwrap it, keep the token in an
 * httpOnly cookie and return just the user, so no browser code ever sees an
 * access_token (SRS 7).
 */
export interface AuthResponse {
  user: User;
  access_token: string;
  token_type: string;
  expires_at: string;
  expires_in: number;
}

export interface EventResponse {
  event: BiletEvent;
}

export interface EventListResponse {
  events: BiletEvent[];
  total: number;
  limit: number;
  offset: number;
}

/** The API's error envelope: { "error": { code, message, fields? } }. */
export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
    fields?: Record<string, string>;
  };
}

export interface CreateEventInput {
  title: string;
  slug?: string;
  description?: string;
  category?: string;
  venue_name?: string;
  venue_address?: string;
  starts_at: string;
  ends_at: string;
  timezone: string;
  visibility?: EventVisibility;
  seating_mode?: SeatingMode;
  capacity?: number;
  refund_policy?: string;
  /** A URL returned by POST /uploads/images (SRS 4.2). */
  cover_image_url?: string;
}

/**
 * Money arrives from the API as a decimal string ("5000.00"), never a number,
 * so a numeric(14,2) is never rounded by JavaScript's float arithmetic.
 */
export type Money = string;

export interface TicketType {
  id: string;
  event_id: string;
  name: string;
  description?: string;
  price_kzt: Money;
  quantity_total: number;
  quantity_sold: number;
  quantity_reserved: number;
  quantity_refunded: number;
  quantity_remaining: number;
  /** How many of this tier walked through the door (SRS 4.3). */
  quantity_checked_in: number;
  max_per_order: number;
  sales_start_at?: string;
  sales_end_at?: string;
  is_hidden: boolean;
  is_free: boolean;
  display_order: number;
  created_at: string;
  updated_at: string;
}

export interface TicketTypeListResponse {
  ticket_types: TicketType[];
}

export interface TicketTypeResponse {
  ticket_type: TicketType;
}

/** What the attendee-facing event page returns. */
export interface PublicEventResponse {
  event: BiletEvent;
  ticket_types: TicketType[];
  on_sale: boolean;
  sold_out: boolean;
  /** True when a platform admin has suspended the event (SRS 4.12). */
  suspended: boolean;
  /** Whether this event is cleared to take money yet (SRS 4.5). */
  paid_sales_active: boolean;
  /** Whether activation gates anything here - false for a free event. */
  paid_sales_required: boolean;
}

export interface Order {
  id: string;
  order_number: string;
  event_id: string;
  buyer_user_id?: string;
  buyer_email: string;
  buyer_name: string;
  status: string;
  currency: string;
  subtotal_kzt: Money;
  discount_kzt: Money;
  processing_fee_kzt: Money;
  total_kzt: Money;
  placed_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface OrderItem {
  id: string;
  order_id: string;
  ticket_type_id: string;
  ticket_type_name: string;
  quantity: number;
  unit_price_kzt: Money;
  line_total_kzt: Money;
}

export interface IssuedTicket {
  id: string;
  ticket_code: string;
  qr_token: string;
  ticket_type_id: string;
  ticket_type_name: string;
  status: string;
  issued_at: string;
}

export interface OrderAttendee {
  id: string;
  order_id: string;
  full_name: string;
  email: string;
}

export interface OrderPayment {
  id: string;
  amount_kzt: Money;
  status: string;
  provider: string;
  is_simulated: boolean;
  paid_at: string;
}

/** The checkout response, and what GET /orders/{id} returns. */
export interface CheckoutResult {
  order: Order;
  items: OrderItem[];
  attendee?: OrderAttendee;
  tickets: IssuedTicket[];
  payment?: OrderPayment;
  promo?: AppliedPromo;
}

export interface CreateTicketTypeInput {
  name: string;
  description?: string;
  price_kzt?: Money;
  quantity_total: number;
  max_per_order?: number;
  sales_start_at?: string;
  sales_end_at?: string;
  is_hidden?: boolean;
}

export interface CheckoutInput {
  buyer_name: string;
  buyer_email: string;
  buyer_phone?: string;
  items: { ticket_type_id: string; quantity: number }[];
  /** A typed code, or the opaque token from a scanned campaign QR. */
  promo_code?: string;
  campaign_token?: string;
}

export type DiscountType = "percentage" | "fixed_amount";
export type CampaignStatus = "draft" | "active" | "disabled" | "exhausted" | "expired";

export interface Campaign {
  id: string;
  event_id: string;
  name: string;
  code: string;
  promo_code_id: string;
  code_is_active: boolean;
  discount_type: DiscountType;
  discount_value: Money;
  starts_at?: string;
  ends_at?: string;
  max_redemptions?: number;
  redemption_count: number;
  status: CampaignStatus;
  /** Always prefixed CMP_. Opaque: it never encodes the discount itself. */
  qr_token: string;
  ticket_type_ids: string[];
  /** The trackable event link the QR encodes. */
  campaign_url: string;
  qr_image_url: string;
  remaining: number;
  orders_count: number;
  tickets_sold: number;
  gross_revenue_kzt: Money;
  discount_given_kzt: Money;
  created_at: string;
}

export interface CreateCampaignInput {
  name: string;
  code: string;
  discount_type: DiscountType;
  discount_value: Money;
  max_redemptions?: number;
  starts_at?: string;
  ends_at?: string;
  ticket_type_ids?: string[];
}

/** What the checkout shows once a promo code has been priced by the server. */
export interface PromoPreview {
  code: string;
  campaign_id: string;
  campaign_name: string;
  discount_type: DiscountType;
  discount_value: Money;
  subtotal_kzt: Money;
  discount_kzt: Money;
  total_kzt: Money;
  remaining: number;
  applies_to_all: boolean;
}

export interface AppliedPromo {
  campaign_id: string;
  campaign_name: string;
  code: string;
  discount_kzt: Money;
}

export type SupportStatus = "open" | "in_progress" | "waiting_for_customer" | "resolved";

export interface SupportCase {
  id: string;
  case_number: string;
  kind: "attendee" | "organizer";
  category: string;
  status: SupportStatus;
  subject: string;
  requester_user_id: string;
  requester_name: string;
  assigned_to_user_id?: string;
  assigned_to_name?: string;
  event_id?: string;
  event_title?: string;
  order_id?: string;
  order_number?: string;
  ticket_id?: string;
  ticket_code?: string;
  last_message_at?: string;
  resolved_at?: string;
  created_at: string;
  message_count: number;
}

export interface SupportMessage {
  id: string;
  support_case_id: string;
  sender_user_id?: string;
  sender_name: string;
  /** "requester" is the attendee who opened the case; "staff" is the organizer. */
  sender_role: "requester" | "staff";
  body: string;
  is_internal_note: boolean;
  created_at: string;
}

export interface SupportThread {
  case: SupportCase;
  messages: SupportMessage[];
  can_reply: boolean;
  can_moderate: boolean;
}

export interface OpenCaseInput {
  category: string;
  subject: string;
  message: string;
  order_id?: string;
  ticket_id?: string;
  event_id?: string;
}

export type Lifecycle =
  | "upcoming"
  | "active"
  | "completed"
  | "cancelled"
  | "draft"
  | "suspended";

export interface TicketTypeSales {
  ticket_type_id: string;
  name: string;
  price_kzt: Money;
  quantity_total: number;
  sold: number;
  remaining: number;
  checked_in: number;
  revenue_kzt: Money;
}

export interface CampaignSales {
  campaign_id: string;
  name: string;
  code: string;
  redemptions: number;
  tickets_sold: number;
  revenue_kzt: Money;
  discount_kzt: Money;
}

export interface SalesPoint {
  day: string;
  orders: number;
  tickets: number;
  revenue_kzt: Money;
}

/** Every figure here is computed by PostgreSQL from the authoritative rows. */
export interface EventAnalytics {
  event_id: string;
  total_capacity: number;
  tickets_sold: number;
  tickets_remaining: number;
  tickets_refunded: number;
  percentage_sold: number;
  gross_revenue_kzt: Money;
  discounts_kzt: Money;
  refunds_kzt: Money;
  net_revenue_kzt: Money;
  orders_count: number;
  checked_in: number;
  absent: number;
  check_in_percentage: number;
  by_ticket_type: TicketTypeSales[];
  by_campaign: CampaignSales[];
  sales_over_time: SalesPoint[];
}

export interface AnalyticsResponse {
  analytics: EventAnalytics;
  event: BiletEvent;
}

export interface TimelineEntry {
  id: number;
  event_id?: string;
  actor_user_id?: string;
  actor_name?: string;
  action: string;
  entity_type: string;
  entity_id?: string;
  description?: string;
  created_at: string;
}

/** One step of the paid-sales activation checklist (SRS 4.5). */
export type ActivationStep = "identity" | "payout" | "terms" | "fee";

export type ActivationStatus =
  | "not_started"
  | "in_progress"
  | "active"
  | "suspended";

/**
 * Paid-sales activation state for one event.
 *
 * `outstanding` is the server's list of steps still to do, so the UI never has
 * to decide for itself what "complete" means.
 */
export interface Activation {
  event_id: string;
  status: ActivationStatus;
  identity_verified_at?: string;
  payout_verified_at?: string;
  terms_accepted_at?: string;
  fee_paid_at?: string;
  activation_fee_kzt: Money;
  activated_at?: string;
  suspended_at?: string;
  suspension_reason?: string;
  is_active: boolean;
  required_for_sales: boolean;
  outstanding: ActivationStep[];
}

/** The checklist submission. Each flag completes one step. */
export interface ActivationSubmission {
  confirm_identity?: boolean;
  confirm_payout?: boolean;
  accept_terms?: boolean;
  pay_activation_fee?: boolean;
}

/** One row of the organizer's attendee view (SRS 4.9). */
export interface EventOrder {
  id: string;
  order_number: string;
  buyer_name: string;
  buyer_email: string;
  status: string;
  total_kzt: Money;
  discount_kzt: Money;
  refunded_kzt: Money;
  ticket_count: number;
  /** Tickets that still admit somebody - zero once an order is refunded. */
  live_tickets: number;
  checked_in: number;
  placed_at?: string;
  created_at: string;
  /**
   * Mirrors the rule the refund endpoint enforces, so the button can be
   * disabled rather than offering an action certain to fail.
   */
  refundable: boolean;
  /**
   * A free registration is cancelled rather than refunded: there is no money
   * to reverse, and the refunds table rejects a zero amount (SRS 4.9). The two
   * flags are mutually exclusive.
   */
  cancellable: boolean;
}

export interface Refund {
  id: string;
  order_id: string;
  payment_id: string;
  amount_kzt: Money;
  status: string;
  reason?: string;
  is_simulated: boolean;
  processed_at?: string;
  created_at: string;
}

export interface RefundResponse {
  refund: Refund;
  order: Order;
  voided_tickets: number;
}

// --- Phase 12: the admin portal, account recovery and uploads ---------------

export interface AdminUserRow {
  id: string;
  email: string;
  full_name: string;
  status: string;
  email_verified: boolean;
  roles: string[];
  event_count: number;
  order_count: number;
  last_login_at?: string;
  created_at: string;
}

export interface AdminEventRow {
  id: string;
  title: string;
  slug: string;
  status: string;
  lifecycle: Lifecycle;
  organizer_email: string;
  organizer_name: string;
  starts_at: string;
  tickets_sold: number;
  revenue_kzt: Money;
  activation_status: ActivationStatus;
}

export interface AdminOrderRow {
  id: string;
  order_number: string;
  buyer_name: string;
  buyer_email: string;
  event_title: string;
  event_id: string;
  status: string;
  total_kzt: Money;
  refunded_kzt: Money;
  ticket_count: number;
  created_at: string;
}

export interface AdminPaymentRow {
  id: string;
  purpose: string;
  status: string;
  amount_kzt: Money;
  order_number?: string;
  event_title?: string;
  is_simulated: boolean;
  created_at: string;
}

export interface AdminSearchResults {
  query: string;
  users: AdminUserRow[];
  events: AdminEventRow[];
  orders: AdminOrderRow[];
  payments: AdminPaymentRow[];
}

export interface PlatformStats {
  users: number;
  organizers: number;
  events: number;
  published_events: number;
  suspended_events: number;
  orders: number;
  tickets_sold: number;
  gross_revenue_kzt: Money;
  refunded_kzt: Money;
  failed_payments: number;
  open_support_cases: number;
}

export interface AdminSearchResponse {
  results: AdminSearchResults;
  stats: PlatformStats;
}

/** The uninformative answer to "send me a reset link" (SRS 4.1). */
export interface AcceptedResponse {
  status: string;
  message: string;
}

export interface UploadedImage {
  url: string;
  filename: string;
  bytes: number;
  mime_type: string;
}


// --- Phase 13: moderation, platform settings and organizer profiles ---------

export type ReportReason =
  | "fraud"
  | "misleading"
  | "inappropriate"
  | "spam"
  | "copyright"
  | "other";

export type ReportStatus = "open" | "reviewing" | "upheld" | "dismissed";

/** One complaint about an event, with the context a moderator needs (SRS 4.12). */
export interface EventReport {
  id: string;
  event_id: string;
  event_title: string;
  event_slug: string;
  event_status: EventStatus;
  organizer_id: string;
  organizer_email: string;
  reporter_user_id?: string;
  reporter_email?: string;
  reason: ReportReason;
  details?: string;
  status: ReportStatus;
  reviewed_by?: string;
  reviewed_at?: string;
  resolution?: string;
  created_at: string;
}

export interface EventReportsResponse {
  reports: EventReport[];
  total: number;
}

/** One configurable platform value (SRS 4.12). */
export interface PlatformSetting {
  key: string;
  value: unknown;
  description?: string;
  updated_by?: string;
  updated_at: string;
}

export interface PlatformSettingsResponse {
  settings: PlatformSetting[];
}

/**
 * A payout destination as the profile exposes it.
 *
 * Masked only: NFR section 7 forbids the platform storing card data, so there
 * is nothing here but a provider name and a display value.
 */
export interface PayoutAccount {
  id: string;
  provider: string;
  masked_account?: string;
  currency: string;
  status: string;
  is_simulated: boolean;
  verified_at?: string;
  created_at: string;
}

/** An organizer's contact and payout information (SRS 4.1). */
export interface OrganizerProfile {
  id?: string;
  user_id: string;
  display_name?: string;
  legal_name?: string;
  contact_email?: string;
  contact_phone?: string;
  description?: string;
  website_url?: string;
  identity_verified_at?: string;
  created_at?: string;
  updated_at?: string;
  payout_accounts: PayoutAccount[];
}

/**
 * A profile PATCH. An absent key leaves the field alone; an explicit null
 * clears it - the same tri-state the event PATCH uses.
 */
export interface ProfilePatch {
  display_name?: string | null;
  legal_name?: string | null;
  contact_email?: string | null;
  contact_phone?: string | null;
  description?: string | null;
  website_url?: string | null;
}

export interface CancelOrderResponse {
  order: Order;
  cancelled_tickets: number;
}

export interface SupportCaseResponse {
  case: SupportCase;
  messages: SupportMessage[];
  can_reply: boolean;
  can_moderate: boolean;
}

export type StaffRole = "event_admin" | "support_staff" | "manager";

/** Somebody an organizer has authorised to work an event (SRS 4.8). */
export interface StaffMember {
  id: string;
  event_id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  role: StaffRole;
  assigned_at: string;
  revoked_at?: string;
}

export interface StaffResponse {
  staff: StaffMember[];
}

/**
 * One row of the door-side attendee search (SRS 4.8).
 *
 * There is deliberately no QR token here: the list is a way to find somebody
 * whose code will not scan, not a way to mint an admission credential. Check-in
 * goes by ticket_id and runs the same transaction as a camera scan.
 */
export interface AttendeeTicket {
  ticket_id: string;
  ticket_code: string;
  attendee_name: string;
  attendee_email: string;
  ticket_type_name: string;
  status: "valid" | "checked_in" | "cancelled" | "refunded";
  order_number: string;
  checked_in_at?: string;
  /** Mirrors the gate's rule, so a row can be disabled rather than refused. */
  admissible: boolean;
}

// --- assigned seating (SRS 4.3.1) -------------------------------------------

/**
 * What the server knows about a seat.
 *
 * "Selected" is deliberately absent: that is this browser's own state, and no
 * other client can know it.
 */
export type SeatStatus = "available" | "held" | "sold" | "unavailable";

export interface Seat {
  id: string;
  number: string;
  x: number;
  y: number;
  accessible: boolean;
  status: SeatStatus;
}

export interface SeatRow {
  id: string;
  label: string;
  seats: Seat[];
}

export interface SeatSection {
  id: string;
  name: string;
  price_category: string;
  ticket_type_id?: string;
  ticket_type_name?: string;
  price_kzt?: Money;
  rows: SeatRow[];
}

export interface SeatMap {
  event_id: string;
  venue_id: string;
  venue_name: string;
  sections: SeatSection[];
  min_x: number;
  min_y: number;
  max_x: number;
  max_y: number;
  total_seats: number;
  available_seats: number;
}

/** A reserved basket (SRS 4.6). */
export interface Hold {
  order_id: string;
  order_number: string;
  event_id: string;
  status: string;
  items: OrderItem[];
  subtotal_kzt: Money;
  reserved_until: string;
  seconds_remaining: number;
  estimated_processing_fee_kzt: Money;
  estimated_total_kzt: Money;
}
