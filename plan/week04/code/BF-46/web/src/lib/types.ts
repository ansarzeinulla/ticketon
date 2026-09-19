/**
 * Mirrors of the JSON the Go API returns. Field names match the Go struct tags
 * exactly, so a response can be used without any remapping.
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
  created_at: string;
  updated_at: string;
}

/** POST /auth/register and POST /auth/login both return this. */
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

/** 202 answers that start something asynchronous, such as an email. */
export interface AcceptedResponse {
  status: string;
  message: string;
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

export interface UploadedImage {
  url: string;
  filename: string;
  bytes: number;
  mime_type: string;
}

/** What the attendee-facing event page returns. */
export interface PublicEventResponse {
  event: BiletEvent;
  ticket_types: TicketType[];
  on_sale: boolean;
  sold_out: boolean;
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
}

export interface CheckoutInput {
  buyer_name: string;
  buyer_email: string;
  buyer_phone?: string;
  /** By tier, for general admission. */
  items?: { ticket_type_id: string; quantity: number }[];
  /** By seat, for assigned seating: the server prices each seat itself. */
  seat_ids?: string[];
}

/** One row of the organizer's order list. */
export interface EventOrder {
  id: string;
  order_number: string;
  buyer_name: string;
  buyer_email: string;
  status: string;
  total_kzt: Money;
  ticket_count: number;
  placed_at?: string;
  created_at: string;
}

/** One issued ticket with the person it was issued to. */
export interface Guest {
  ticket_id: string;
  ticket_code: string;
  full_name: string;
  email: string;
  ticket_type_name: string;
  status: string;
  order_number: string;
}

export type SeatStatus = "available" | "sold" | "unavailable";

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
