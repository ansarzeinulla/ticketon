import Link from "next/link";
import type { ReactNode } from "react";

import { LanguageSwitcher } from "@/components/language-switcher";
import { getTranslations } from "@/lib/i18n/server";

/**
 * Shell for the attendee-facing pages. Deliberately has no auth gate: browsing
 * an event and buying a ticket must work without an account.
 *
 * This header carries the language switcher, because the pages inside it are
 * the customer-facing interface the SRS requires in Kazakh and Russian (§685).
 */
export default async function PublicLayout({ children }: { children: ReactNode }) {
  const { t } = await getTranslations();

  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-b border-border-subtle bg-surface">
        <div className="mx-auto flex h-16 max-w-3xl items-center justify-between gap-4 px-4 sm:px-6">
          <Link href="/" className="flex items-center gap-2">
            <span className="grid h-8 w-8 place-items-center rounded-lg bg-brand text-sm font-bold text-white">
              B
            </span>
            <span className="font-semibold tracking-tight">BiletFlow</span>
          </Link>
          <div className="flex items-center gap-3">
            <LanguageSwitcher />
            <Link
              href="/dashboard"
              className="text-sm font-medium text-foreground-muted hover:text-foreground"
            >
              {t("header.organizers")}
            </Link>
          </div>
        </div>
      </header>
      {children}
    </div>
  );
}
