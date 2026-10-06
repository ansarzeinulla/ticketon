"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";

/**
 * "Forgot password" (SRS 4.1).
 *
 * The confirmation deliberately does not say whether the address had an
 * account. The API answers the same either way, and a page that said "no
 * account with that email" would undo that by turning this form into a way of
 * testing which addresses are registered.
 */
export function ForgotPasswordForm() {
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    setFieldErrors({});

    try {
      await api.requestPasswordReset(email.trim());
      setSent(true);
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        setError(Object.keys(cause.fields).length > 0 ? null : cause.message);
      } else {
        setError("Something went wrong. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (sent) {
    return (
      <>
        <div className="mb-6">
          <h1 className="text-2xl font-semibold tracking-tight">Check your inbox</h1>
        </div>
        <Alert tone="success" title="Reset link sent">
          If <span className="font-medium">{email.trim()}</span> has an account, a reset
          code is on its way. It works once, and stops working after an hour.
        </Alert>
        <p className="mt-4 text-sm text-foreground-muted">
          This MVP has no mail server: the message is printed to the API&apos;s console.
          Copy the code from there and{" "}
          <Link href="/reset-password" className="font-medium text-brand hover:underline">
            enter it here
          </Link>
          .
        </p>
        <p className="mt-6 text-center text-sm text-foreground-muted">
          <Link href="/login" className="font-medium text-brand hover:underline">
            Back to sign in
          </Link>
        </p>
      </>
    );
  }

  return (
    <>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight">Forgot your password?</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          Enter your email and we will send a code to set a new one.
        </p>
      </div>

      {error && (
        <div className="mb-4">
          <Alert tone="error">{error}</Alert>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <TextField
          label="Email"
          type="email"
          name="email"
          autoComplete="email"
          placeholder="dana@biletflow.kz"
          required
          value={email}
          error={fieldErrors.email}
          onChange={(event) => setEmail(event.target.value)}
        />

        <Button type="submit" loading={submitting} className="w-full">
          Send reset code
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-foreground-muted">
        Remembered it?{" "}
        <Link href="/login" className="font-medium text-brand hover:underline">
          Sign in
        </Link>
      </p>
    </>
  );
}
