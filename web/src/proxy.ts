import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

import { TOKEN_COOKIE } from "@/lib/session";

/**
 * Route gate. In Next.js 16 this file replaces `middleware.ts`.
 *
 * This is an optimistic check only: it can see that a token cookie exists, not
 * that the token is still valid. The real verification happens in AuthProvider,
 * which calls GET /auth/me and signs the user out when the API rejects it.
 * The value here is avoiding a visible flash of the dashboard before that
 * round-trip finishes.
 */
export function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  const hasToken = Boolean(request.cookies.get(TOKEN_COOKIE)?.value);

  const isAuthPage = pathname === "/login" || pathname === "/register";

  if (!hasToken && !isAuthPage) {
    const login = new URL("/login", request.url);
    // Remember where they were headed so login can send them back.
    login.searchParams.set("next", `${pathname}${search}`);
    return NextResponse.redirect(login);
  }

  if (hasToken && isAuthPage) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  return NextResponse.next();
}

export const config = {
  // Deliberately NOT "/events/:path*": /events/[slug] is the attendee-facing
  // page and has to stay reachable without an account. Only the organizer's
  // create form under /events/new is gated, alongside everything in /dashboard.
  // /orders/[id] is public too - the order id is the unguessable capability.
  // /admin is gated here only for the token cookie. Whether the account is
  // actually a platform admin is decided by the API on every request, and by
  // the portal itself for what it renders - a route matcher cannot read a role
  // out of a cookie, and pretending otherwise would be security theatre.
  matcher: ["/dashboard/:path*", "/admin/:path*", "/events/new", "/login", "/register"],
};
