import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { TOKEN_COOKIE } from "@/lib/session";

/**
 * The root is just a signpost. Sending signed-in organizers straight to the
 * dashboard is decided on the server, so there is no flash of the wrong page.
 */
export default async function HomePage() {
  // cookies() is async in Next.js 15+.
  const cookieStore = await cookies();
  const hasToken = Boolean(cookieStore.get(TOKEN_COOKIE)?.value);

  redirect(hasToken ? "/dashboard" : "/login");
}
