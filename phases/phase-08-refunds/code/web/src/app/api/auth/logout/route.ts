import { NextResponse } from "next/server";

import { clearToken } from "@/lib/server-session";

/**
 * Sign out.
 *
 * A POST rather than a GET: signing out changes state, and a GET would let any
 * page sign somebody out by embedding an image.
 */
export async function POST() {
  await clearToken();
  return NextResponse.json({ status: "signed_out" });
}
