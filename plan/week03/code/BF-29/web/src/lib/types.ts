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
}
