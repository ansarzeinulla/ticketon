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
