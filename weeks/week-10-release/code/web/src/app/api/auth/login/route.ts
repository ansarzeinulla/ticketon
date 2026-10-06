import { NextResponse } from "next/server";

import { apiBaseURL, isSecureRequest, writeToken } from "@/lib/server-session";

/**
 * Sign in (SRS 4.1, 7).
 *
 * The browser posts credentials here rather than to the Go API. This handler
 * forwards them, keeps the access token in an httpOnly cookie, and returns
 * only the user. The token never reaches JavaScript, so a script injected into
 * the page cannot read a session and replay it somewhere else.
 */
export async function POST(request: Request) {
  let credentials: unknown;
  try {
    credentials = await request.json();
  } catch {
    return NextResponse.json(
      { error: { code: "invalid_json", message: "Expected a JSON body." } },
      { status: 400 },
    );
  }

  const upstream = await fetch(`${apiBaseURL()}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(credentials),
    cache: "no-store",
  });

  const payload = await upstream.json().catch(() => null);

  if (!upstream.ok) {
    // The API's error envelope is passed through untouched, so the form can
    // still highlight the exact fields it names.
    return NextResponse.json(payload, { status: upstream.status });
  }

  const { access_token: token, expires_at: expiresAt, user } = payload as {
    access_token?: string;
    expires_at?: string;
    user?: unknown;
  };

  if (!token) {
    return NextResponse.json(
      { error: { code: "invalid_response", message: "The API returned no session." } },
      { status: 502 },
    );
  }

  await writeToken(token, expiresAt, isSecureRequest(request));

  // Deliberately no access_token in the response body: handing it back would
  // undo the entire point of storing it out of JavaScript's reach.
  return NextResponse.json({ user });
}
