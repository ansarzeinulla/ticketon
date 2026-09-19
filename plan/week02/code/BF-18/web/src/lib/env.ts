/**
 * Settings the browser needs, read in one place.
 *
 * Next inlines NEXT_PUBLIC_* variables at build time, so a missing value would
 * otherwise surface as a request to "undefined/auth/login". Falling back to the
 * local API keeps `npm run dev` working with no .env.local at all.
 */

const DEFAULT_API_BASE_URL = "http://localhost:8080/api/v1";

function readApiBaseUrl(): string {
  const raw = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  if (!raw) return DEFAULT_API_BASE_URL;
  // A trailing slash would turn "/auth/login" into "//auth/login".
  return raw.replace(/\/+$/, "");
}

export const env = {
  apiBaseUrl: readApiBaseUrl(),
} as const;
