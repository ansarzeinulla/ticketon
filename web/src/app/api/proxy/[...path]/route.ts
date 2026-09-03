import { NextResponse } from "next/server";

import { apiBaseURL, clearToken, readToken } from "@/lib/server-session";

/**
 * The authenticated proxy to the Go API (SRS 7).
 *
 * Every call the browser makes goes through here. The session token lives in an
 * httpOnly cookie the browser cannot read, so this handler is the only thing
 * that can turn it into an `Authorization: Bearer` header.
 *
 * It is deliberately a dumb forwarder: it does not know what any endpoint
 * means, so a new API route needs no change here. What it does know is which
 * headers may cross - see below.
 */

/**
 * Request headers worth forwarding.
 *
 * An allowlist, not a blocklist: the browser can set arbitrary headers, and
 * forwarding them wholesale would let a page smuggle its own Authorization,
 * X-Forwarded-For or Cookie upstream. Everything that matters is here.
 */
const FORWARD_REQUEST_HEADERS = ["content-type", "accept", "accept-language"];

/**
 * Response headers worth returning.
 *
 * Content-Disposition matters: it is what makes a PDF ticket, a CSV report or
 * an .ics file download rather than render.
 */
const FORWARD_RESPONSE_HEADERS = [
  "content-type",
  "content-disposition",
  "content-length",
  "cache-control",
  "location",
];

async function forward(
  request: Request,
  segments: string[],
): Promise<NextResponse> {
  const token = await readToken();

  const search = new URL(request.url).search;
  const target = `${apiBaseURL()}/${segments.map(encodeURIComponent).join("/")}${search}`;

  const headers = new Headers();
  for (const name of FORWARD_REQUEST_HEADERS) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  if (token) headers.set("Authorization", `Bearer ${token}`);

  // The body is read whole rather than streamed. Uploads are capped at 5 MB by
  // the API, so buffering costs little, and streaming a request body needs
  // duplex negotiation that varies between runtimes - a source of failures
  // that only appear in production.
  const body =
    request.method === "GET" || request.method === "HEAD"
      ? undefined
      : await request.arrayBuffer();

  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method: request.method,
      headers,
      body,
      cache: "no-store",
      redirect: "manual",
    });
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "network_error",
          message: "Could not reach the API. Is it running?",
        },
      },
      { status: 502 },
    );
  }

  // A rejected token means the cookie is stale. Clearing it turns the next
  // request into a clean anonymous one instead of a second identical failure.
  if (upstream.status === 401) {
    await clearToken();
  }

  const responseHeaders = new Headers();
  for (const name of FORWARD_RESPONSE_HEADERS) {
    const value = upstream.headers.get(name);
    if (value) responseHeaders.set(name, value);
  }

  // The body is passed through as bytes, so this works for JSON, a PDF ticket,
  // a QR PNG, a CSV report and an .ics file without knowing which is which.
  return new NextResponse(upstream.body, {
    status: upstream.status,
    headers: responseHeaders,
  });
}

type Context = { params: Promise<{ path: string[] }> };

export async function GET(request: Request, context: Context) {
  return forward(request, (await context.params).path);
}

export async function POST(request: Request, context: Context) {
  return forward(request, (await context.params).path);
}

export async function PATCH(request: Request, context: Context) {
  return forward(request, (await context.params).path);
}

export async function PUT(request: Request, context: Context) {
  return forward(request, (await context.params).path);
}

export async function DELETE(request: Request, context: Context) {
  return forward(request, (await context.params).path);
}
