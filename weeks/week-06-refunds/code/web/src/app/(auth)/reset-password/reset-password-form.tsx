"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";

/**
 * "Reset password" (SRS 4.1).
 *
 * The token usually arrives in the link as ?token=, but the field is editable
 * and shown either way: in this MVP the code is printed to the API console, so
 * somebody will be pasting it by hand.
 *
 * Resetting does not sign anybody in. The API deliberately returns no session -
 * proving control of an inbox is not the same as intending to sign in on this
 * device - so this redirects to the login page instead.
 */
export function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [token, setToken] = useState(searchParams.get("token") ?? "");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    // Checked here rather than at the API, which has no business knowing the
    // user typed it twice.
    if (password !== confirmation) {
      setFieldErrors({ confirmation: "The two passwords do not match." });
      return;
    }

    setSubmitting(true);
    setError(null);
    setFieldErrors({});

    try {
      await api.resetPassword(token.trim(), password);
      setDone(true);
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        setError(
          cause.code === "token_invalid"
            ? "That code is invalid, expired or already used. Ask for a new one."
            : Object.keys(cause.fields).length > 0
              ? null
              : cause.message,
        );
      } else {
        setError("Something went wrong. Please try again.");
      }
      setSubmitting(false);
    }
  }

  if (done) {
    return (
      <>
        <div className="mb-6">
          <h1 className="text-2xl font-semibold tracking-tight">Password changed</h1>
        </div>
        <Alert tone="success">
          Your new password is in place. Sign in with it now.
        </Alert>
        <div className="mt-6">
          <Button className="w-full" onClick={() => router.push("/login")}>
            Go to sign in
          </Button>
        </div>
      </>
    );
  }

  return (
    <>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight">Set a new password</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          Paste the code from your email, then choose a new password.
        </p>
      </div>

      {error && (
        <div className="mb-4">
          <Alert tone="error">{error}</Alert>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <TextField
          label="Reset code"
          name="token"
          required
          value={token}
          error={fieldErrors.token}
          hint="From the email. In this MVP it is printed to the API console."
          onChange={(event) => setToken(event.target.value)}
        />

        <TextField
          label="New password"
          type="password"
          name="password"
          autoComplete="new-password"
          required
          value={password}
          error={fieldErrors.password}
          hint="At least 8 characters."
          onChange={(event) => setPassword(event.target.value)}
        />

        <TextField
          label="Confirm new password"
          type="password"
          name="confirmation"
          autoComplete="new-password"
          required
          value={confirmation}
          error={fieldErrors.confirmation}
          onChange={(event) => setConfirmation(event.target.value)}
        />

        <Button type="submit" loading={submitting} className="w-full">
          Change password
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-foreground-muted">
        Need a new code?{" "}
        <Link href="/forgot-password" className="font-medium text-brand hover:underline">
          Ask for one
        </Link>
      </p>
    </>
  );
}
