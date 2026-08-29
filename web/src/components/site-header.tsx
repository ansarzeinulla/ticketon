"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/auth-context";

const links = [
  { href: "/dashboard", label: "Events" },
  { href: "/events/new", label: "Create event" },
] as const;

/** Shown only to platform administrators (SRS 2.1). */
const adminLink = { href: "/admin", label: "Administration" } as const;

export function SiteHeader() {
  const pathname = usePathname();
  const router = useRouter();
  const { user, logout } = useAuth();

  function handleSignOut() {
    logout();
    router.replace("/login");
    router.refresh();
  }

  return (
    <header className="border-b border-border-subtle bg-surface">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-6 px-4 sm:px-6">
        <Link href="/dashboard" className="flex items-center gap-2">
          <span className="grid h-8 w-8 place-items-center rounded-lg bg-brand text-sm font-bold text-white">
            B
          </span>
          <span className="hidden font-semibold tracking-tight sm:inline">BiletFlow</span>
        </Link>

        <nav className="flex items-center gap-1" aria-label="Main">
          {[...links, ...(user?.roles.includes("platform_admin") ? [adminLink] : [])].map((link) => {
            const active = pathname === link.href;
            return (
              <Link
                key={link.href}
                href={link.href}
                aria-current={active ? "page" : undefined}
                className={`rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${
                  active
                    ? "bg-brand-soft text-brand-strong"
                    : "text-foreground-muted hover:bg-surface-muted hover:text-foreground"
                }`}
              >
                {link.label}
              </Link>
            );
          })}
        </nav>

        <div className="ml-auto flex items-center gap-3">
          {user && (
            // The name is the link to the profile: it is where somebody looks
            // for "my account" without being told to.
            <Link
              href="/dashboard/profile"
              className="hidden rounded-lg px-2 py-1 text-right hover:bg-surface-muted sm:block"
              data-testid="profile-link"
            >
              <span className="block text-sm font-medium leading-tight">{user.full_name}</span>
              <span className="block text-xs text-foreground-muted">{user.email}</span>
            </Link>
          )}
          <Button variant="secondary" size="sm" onClick={handleSignOut}>
            Sign out
          </Button>
        </div>
      </div>
    </header>
  );
}
