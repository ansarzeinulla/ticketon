/** Mirrors of the JSON the Go API returns, limited to what the scanner uses. */

export interface User {
  id: string;
  email: string;
  full_name: string;
  roles: string[];
  status: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  expires_at: string;
}

export interface CheckInStats {
  issued: number;
  checked_in: number;
}

export interface ScannableEvent {
  id: string;
  title: string;
  slug: string;
  starts_at: string;
  ends_at: string;
  timezone: string;
  venue_name?: string;
  status: string;
  /** "organizer", "event_admin" or "manager". */
  access_via: string;
  stats: CheckInStats;
}

export interface CheckInSuccess {
  ticket_id: string;
  ticket_code: string;
  ticket_type_name: string;
  attendee_name: string;
  attendee_email: string;
  seat_label?: string;
  checked_in_at: string;
  stats: CheckInStats;
}

/** The scanner result codes the API returns; the UI switches on these. */
export type ScanErrorCode =
  | "already_checked_in"
  | "campaign_token"
  | "wrong_event"
  | "ticket_not_valid"
  | "unknown_ticket"
  | "forbidden"
  | "unauthorized"
  | "network_error"
  | string;

/**
 * One ticket in the manual attendee search (SRS 4.8).
 *
 * Deliberately carries no QR token: staff searching by name are standing next
 * to the person, and handing every device a working admission credential for
 * every attendee would make the QR code pointless. Check-in goes by ticket id.
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

/**
 * One ticket as the offline scanner stores it (SRS 4.8).
 *
 * `token_hash` is the SHA-256 of the QR token, never the token itself: the
 * roster is a whole event's worth of admission credentials sitting on a device
 * that lives on a table at a venue door, and a plaintext copy would let anyone
 * who lifted the phone forge every ticket in the house. The device hashes what
 * it scans and compares hashes.
 */
export interface RosterEntry {
  ticket_id: string;
  token_hash: string;
  ticket_code: string;
  attendee_name: string;
  ticket_type_name: string;
  seat_label?: string;
  status: "valid" | "checked_in" | "cancelled" | "refunded" | string;
}

/** A downloaded snapshot of an event's tickets. */
export interface Roster {
  event_id: string;
  event_title: string;
  generated_at: string;
  tickets: RosterEntry[];
}

/** What the server made of one queued admission on reconciliation. */
export interface SyncResult {
  ticket_id: string;
  outcome: "recorded" | "already_checked_in" | "not_valid" | "unknown_ticket" | string;
  attendee_name?: string;
  checked_in_at?: string;
}

/** The tallied result of one sync, plus the per-ticket detail. */
export interface SyncSummary {
  results: SyncResult[];
  recorded: number;
  already_checked_in: number;
  rejected: number;
}
