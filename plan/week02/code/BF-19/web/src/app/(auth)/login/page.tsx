import { Suspense } from "react";
import type { Metadata } from "next";

import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Sign in" };

export default function LoginPage() {
  return (
    // useSearchParams inside the form needs a Suspense boundary, otherwise the
    // whole route falls back to client-side rendering.
    <Suspense fallback={<p className="text-sm text-foreground-muted">Loading…</p>}>
      <LoginForm />
    </Suspense>
  );
}
