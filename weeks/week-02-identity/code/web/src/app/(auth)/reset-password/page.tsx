import { Suspense } from "react";
import type { Metadata } from "next";

import { ResetPasswordForm } from "./reset-password-form";

export const metadata: Metadata = { title: "Reset password" };

export default function ResetPasswordPage() {
  return (
    // The token arrives as ?token=, and useSearchParams needs a boundary.
    <Suspense fallback={<p className="text-sm text-foreground-muted">Loading…</p>}>
      <ResetPasswordForm />
    </Suspense>
  );
}
