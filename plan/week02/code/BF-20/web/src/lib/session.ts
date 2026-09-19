/** The session cookie, read and written from the browser. */
export const TOKEN_COOKIE = "biletflow_token";

/** Read the access token, or null when nobody is signed in. */
export function getToken(): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)biletflow_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : null;
}

/** Store the access token so later requests can attach it. */
export function setToken(token: string, expiresAt: string): void {
  const expires = new Date(expiresAt).toUTCString();
  document.cookie = `${TOKEN_COOKIE}=${encodeURIComponent(token)}; path=/; expires=${expires}; SameSite=Lax`;
}

/** Forget the session. */
export function clearToken(): void {
  document.cookie = `${TOKEN_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
}
