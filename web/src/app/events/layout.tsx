import Link from "next/link";
import type { ReactNode } from "react";

/**
 * Shell for the attendee-facing pages. Deliberately has no auth gate: browsing
 * an event and buying a ticket must work without an account.
 */
export default function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-b border-border-subtle bg-surface">
        <div className="mx-auto flex h-16 max-w-3xl items-center justify-between px-4 sm:px-6">
          <Link href="/" className="flex items-center gap-2">
            <span className="grid h-8 w-8 place-items-center rounded-lg bg-brand text-sm font-bold text-white">
              B
            </span>
            <span className="font-semibold tracking-tight">BiletFlow</span>
          </Link>
          <Link
            href="/dashboard"
            className="text-sm font-medium text-foreground-muted hover:text-foreground"
          >
            Organizers
          </Link>
        </div>
      </header>
      {children}
    </div>
  );
}
