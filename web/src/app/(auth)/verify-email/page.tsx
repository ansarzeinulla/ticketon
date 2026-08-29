import { Suspense } from "react";
import type { Metadata } from "next";

import { VerifyEmailView } from "./verify-email-view";

export const metadata: Metadata = { title: "Confirm your email" };

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<p className="text-sm text-foreground-muted">Loading…</p>}>
      <VerifyEmailView />
    </Suspense>
  );
}
