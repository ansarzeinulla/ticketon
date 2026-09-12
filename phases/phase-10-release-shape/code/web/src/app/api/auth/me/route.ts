import { NextResponse } from "next/server";

import { apiBaseURL, clearToken, readToken } from "@/lib/server-session";

/**
 * Who is signed in.
 *
 * This is how the browser learns about its own session now that it cannot read
 * the token: it asks, and the server answers from the cookie.
 */
export async function GET() {
  const token = await readToken();
  if (!token) {
    return NextResponse.json(
      { error: { code: "unauthorized", message: "Not signed in." } },
      { status: 401 },
    );
  }

  const upstream = await fetch(`${apiBaseURL()}/auth/me`, {
    headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
    cache: "no-store",
  });

  if (upstream.status === 401) {
    // The cookie outlived the token it names - expired, or revoked. Clearing
    // it here means the next request is a clean anonymous one rather than
    // another round trip that fails the same way.
    await clearToken();
  }

  const payload = await upstream.json().catch(() => null);
  return NextResponse.json(payload, { status: upstream.status });
}
