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
