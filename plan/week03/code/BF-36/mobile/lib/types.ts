/** Mirrors of the JSON the Go API returns, limited to what the app uses. */

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

/** An event as the device lists it. */
export interface MobileEvent {
  id: string;
  title: string;
  slug: string;
  starts_at: string;
  ends_at: string;
  timezone: string;
  venue_name?: string;
  status: string;
}
