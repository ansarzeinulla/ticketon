import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

import { TOKEN_COOKIE } from "@/lib/session";

/**
 * Route gate.
 *
 * This is an optimistic check only: it can see that a token cookie exists, not
 * that the token is still valid. AuthProvider does the real check with
 * GET /auth/me.
 */
export function middleware(request: NextRequest) {
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
  matcher: ["/dashboard/:path*", "/events/new", "/login", "/register"],
};
