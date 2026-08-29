/**
 * Where the access token lives in the browser.
 *
 * A cookie rather than localStorage, for one concrete reason: proxy.ts runs on
 * the server and can only read cookies, so route protection happens before the
 * page renders instead of flashing a dashboard and then bouncing to /login.
 *
 * The cookie is intentionally readable by JavaScript, because the Go API
 * authenticates with an `Authorization: Bearer` header rather than cookies.
 * That means it carries the same XSS exposure as localStorage would. Moving to
 * an httpOnly cookie needs a Next.js route handler proxying every API call, and
 * a refresh token to rotate against - neither exists yet, so this stays a
 * documented Phase 3 trade-off rather than a silent one.
 */

export const TOKEN_COOKIE = "biletflow_token";

/** Read the token, or null when it is missing or we are on the server. */
export function getToken(): string | null {
  if (typeof document === "undefined") return null;

  const match = document.cookie
    .split("; ")
    .find((entry) => entry.startsWith(`${TOKEN_COOKIE}=`));

  if (!match) return null;

  const value = decodeURIComponent(match.slice(TOKEN_COOKIE.length + 1));
  return value === "" ? null : value;
}

/**
 * Store the token until it expires.
 *
 * `expiresAt` comes from the API response, so the cookie and the token stop
 * being valid at the same moment.
 */
export function setToken(token: string, expiresAt: string): void {
  if (typeof document === "undefined") return;

  const expires = new Date(expiresAt);
  const maxAge = Number.isNaN(expires.getTime())
    ? 60 * 60 * 24
    : Math.max(0, Math.floor((expires.getTime() - Date.now()) / 1000));

  const secure = window.location.protocol === "https:" ? "; Secure" : "";
  document.cookie =
    `${TOKEN_COOKIE}=${encodeURIComponent(token)}` +
    `; Path=/; Max-Age=${maxAge}; SameSite=Lax${secure}`;
}

/** Remove the token on sign-out, or when the API rejects it. */
export function clearToken(): void {
  if (typeof document === "undefined") return;
  document.cookie = `${TOKEN_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax`;
}
