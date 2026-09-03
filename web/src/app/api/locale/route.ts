import { NextResponse } from "next/server";

import { asLocale, LOCALE_COOKIE } from "@/lib/i18n/config";

/**
 * Persist the visitor's language choice (SRS §685).
 *
 * A year-long cookie, so the choice survives the visit. It carries no personal
 * data and is not a session credential - just "kk", "ru" or "en" - so it is
 * deliberately readable by client script and not httpOnly; every server
 * component reads it to render in the chosen language.
 */
export async function POST(request: Request) {
  let locale: string | undefined;
  try {
    locale = (await request.json())?.locale;
  } catch {
    // Falls through to the validation below.
  }

  const chosen = asLocale(locale);
  if (!chosen) {
    return NextResponse.json({ error: "unsupported locale" }, { status: 400 });
  }

  const response = NextResponse.json({ locale: chosen });
  response.cookies.set(LOCALE_COOKIE, chosen, {
    path: "/",
    maxAge: 60 * 60 * 24 * 365,
    sameSite: "lax",
  });
  return response;
}
