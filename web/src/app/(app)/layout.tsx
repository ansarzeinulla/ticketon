"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { SiteHeader } from "@/components/site-header";
import { Spinner } from "@/components/ui/button";
import { useAuth } from "@/lib/auth-context";

/**
 * The signed-in shell.
 *
 * proxy.ts already turned away requests with no token cookie. This second gate
 * covers the case the proxy cannot see: a cookie that exists but the API has
 * rejected, which AuthProvider discovers on its /auth/me call.
 */
export default function AppLayout({ children }: { children: ReactNode }) {
  const { status } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (status === "unauthenticated") router.replace("/login");
  }, [status, router]);

  if (status !== "authenticated") {
    return (
      <div className="grid min-h-dvh place-items-center">
        <div className="flex items-center gap-3 text-sm text-foreground-muted">
          <Spinner />
          {status === "loading" ? "Checking your session…" : "Redirecting to sign in…"}
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh flex-col">
      <SiteHeader />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-8 sm:px-6">{children}</main>
    </div>
  );
}
