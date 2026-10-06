import type { ReactNode } from "react";

import { SiteHeader } from "@/components/site-header";

/**
 * The shell every organizer page will sit in: header on top, content below.
 * The session check is added once sign-in exists.
 */
export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <SiteHeader />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-8 sm:px-6">{children}</main>
    </div>
  );
}
