/**
 * The name of the session cookie (SRS 7).
 *
 * Only the name lives here, and deliberately nothing else. This module used to
 * expose getToken/setToken/clearToken against a cookie readable by
 * `document.cookie`, which meant any injected script could lift a session and
 * replay it anywhere. The token is now httpOnly: it is written by the sign-in
 * route handler and read by the proxy route handler, and no browser code can
 * touch it.
 *
 * The name is shared because two places that cannot import each other need it:
 * `proxy.ts`, which runs in the routing layer and only checks that a cookie is
 * present, and `server-session.ts`, which is server-only and does the real
 * reading and writing.
 */
export const TOKEN_COOKIE = "biletflow_session";
