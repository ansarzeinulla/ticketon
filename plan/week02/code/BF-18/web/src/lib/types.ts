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

/** POST /auth/register and POST /auth/login both return this. */
export interface AuthResponse {
  user: User;
  access_token: string;
  token_type: string;
  expires_at: string;
  expires_in: number;
}

/** The API's error envelope: { "error": { code, message, fields? } }. */
export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
    fields?: Record<string, string>;
  };
}

