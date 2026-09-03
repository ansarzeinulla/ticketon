import { NextResponse } from "next/server";

import { apiBaseURL, isSecureRequest, writeToken } from "@/lib/server-session";

/** Create an account and start a session, the same way login does (SRS 4.1). */
export async function POST(request: Request) {
  let details: unknown;
  try {
    details = await request.json();
  } catch {
    return NextResponse.json(
      { error: { code: "invalid_json", message: "Expected a JSON body." } },
      { status: 400 },
    );
  }

  const upstream = await fetch(`${apiBaseURL()}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(details),
    cache: "no-store",
  });

  const payload = await upstream.json().catch(() => null);
  if (!upstream.ok) {
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
  return NextResponse.json({ user }, { status: 201 });
}
