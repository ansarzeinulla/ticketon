import Link from "next/link";
import type { ReactNode } from "react";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center px-4 py-12">
      <div className="w-full max-w-md">
        <Link href="/" className="mb-8 flex items-center justify-center gap-2">
          <span className="grid h-9 w-9 place-items-center rounded-lg bg-brand text-lg font-bold text-white">
            B
          </span>
          <span className="text-xl font-semibold tracking-tight">BiletFlow</span>
        </Link>

        <div className="rounded-2xl border border-border-subtle bg-surface p-6 shadow-sm sm:p-8">
          {children}
        </div>
      </div>
    </div>
  );
}
