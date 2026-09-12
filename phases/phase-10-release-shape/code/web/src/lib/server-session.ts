import "server-only";

import { cookies } from "next/headers";

import { TOKEN_COOKIE } from "@/lib/session";

/**
 * The token lives in an httpOnly cookie the browser never exposes to
 * JavaScript (SRS 7).
 *
 * It used to sit in a cookie readable by `document.cookie`, which meant any
 * injected script could read a session and use it anywhere. It now never
 * reaches the browser at all: the sign-in route handler puts it here, and the
 * proxy route handler is the only thing that reads it back. An XSS payload can
 * still act *as* the user through the app's own origin, but it can no longer
 * steal a bearer token and replay it elsewhere, from another device, after the
 * tab is closed.
 */
export { TOKEN_COOKIE };

/**
 * Cookie settings.
 *
 * `strict` rather than `lax`: every request the app makes is same-origin, so
 * nothing legitimate needs the cookie on a cross-site navigation, and strict is
 * what removes CSRF as a category rather than mitigating it. The cost is that
 * following a link from an email lands signed-out on the first click; for an
 * organizer's dashboard that is a fair trade.
 *
 * Secure is set only over HTTPS, because a Secure cookie is silently dropped on
 * plain http://localhost and development would appear to sign in and then
 * immediately sign out.
 */
function cookieOptions(maxAge: number, secure: boolean) {
  return {
    httpOnly: true,
    sameSite: "strict" as const,
    secure,
    path: "/",
    maxAge,
  };
}

/** Read the session token, server-side only. */
export async function readToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(TOKEN_COOKIE)?.value ?? null;
}

/**
 * Store the token until it expires.
 *
 * The lifetime comes from the API's own `expires_at`, so the cookie and the
 * token stop being valid at the same moment rather than leaving a cookie that
 * outlives the session it names.
 */
export async function writeToken(
  token: string,
  expiresAt: string | undefined,
  secure: boolean,
): Promise<void> {
  const expires = expiresAt ? new Date(expiresAt) : new Date(NaN);
  const maxAge = Number.isNaN(expires.getTime())
    ? 60 * 60 * 24
    : Math.max(0, Math.floor((expires.getTime() - Date.now()) / 1000));

  const store = await cookies();
  store.set(TOKEN_COOKIE, token, cookieOptions(maxAge, secure));
}

/** Remove the session, on sign-out or when the API rejects the token. */
export async function clearToken(): Promise<void> {
  const store = await cookies();
  store.set(TOKEN_COOKIE, "", cookieOptions(0, false));
}

/**
 * Where this process reaches the Go API.
 *
 * Server-side only and deliberately without a NEXT_PUBLIC_ prefix: the browser
 * no longer talks to the API directly, so its address does not belong in a
 * bundle the browser downloads.
 */
export function apiBaseURL(): string {
  return (
    process.env.API_INTERNAL_BASE_URL ||
    process.env.NEXT_PUBLIC_API_BASE_URL ||
    "http://localhost:8080/api/v1"
  );
}

/** Whether the incoming request arrived over HTTPS. */
export function isSecureRequest(request: Request): boolean {
  const url = new URL(request.url);
  if (url.protocol === "https:") return true;
  // Behind a reverse proxy the connection to Next.js is plain HTTP, and the
  // original scheme only survives in this header.
  return request.headers.get("x-forwarded-proto") === "https";
}
