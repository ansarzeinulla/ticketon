import Link from "next/link";

const links = [
  { href: "/", label: "Events" },
] as const;

/**
 * The header of the signed-in area. Sign-in does not exist yet, so for now it
 * only carries the brand and the main navigation.
 */
export function SiteHeader() {
  return (
    <header className="border-b border-border-subtle bg-surface">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-6 px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2">
          <span className="grid h-8 w-8 place-items-center rounded-lg bg-brand text-sm font-bold text-white">
            B
          </span>
          <span className="hidden font-semibold tracking-tight sm:inline">BiletFlow</span>
        </Link>

        <nav className="flex items-center gap-1" aria-label="Main">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="rounded-lg px-3 py-1.5 text-sm font-medium text-foreground-muted transition-colors hover:bg-surface-muted hover:text-foreground"
            >
              {link.label}
            </Link>
          ))}
        </nav>
      </div>
    </header>
  );
}
