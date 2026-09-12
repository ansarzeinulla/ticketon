"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

/** Matches the API's minimum; checked here so the error appears instantly. */
const MIN_PASSWORD_LENGTH = 8;

export function RegisterForm() {
  const router = useRouter();
  const { register } = useAuth();

  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    setFieldErrors({});

    // Confirmation is a UI-only concern; the API never sees it.
    if (password !== confirm) {
      setFieldErrors({ confirm: "The two passwords do not match." });
      return;
    }
    if (password.length < MIN_PASSWORD_LENGTH) {
      setFieldErrors({
        password: `Password must be at least ${MIN_PASSWORD_LENGTH} characters.`,
      });
      return;
    }

    setSubmitting(true);
    try {
      await register({
        email: email.trim(),
        password,
        // The API derives a name from the email when this is omitted.
        full_name: fullName.trim() || undefined,
      });
      router.replace("/dashboard");
      router.refresh();
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fields);
        setFormError(
          error.code === "conflict"
            ? "An account with this email already exists. Try signing in instead."
            : error.fields && Object.keys(error.fields).length > 0
              ? "Please correct the highlighted fields."
              : error.message,
        );
      } else {
        setFormError("Something went wrong. Please try again.");
      }
      setSubmitting(false);
    }
  }

  return (
    <>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight">Create your account</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          Publishing events and issuing free tickets costs nothing.
        </p>
      </div>

      {formError && (
        <div className="mb-4">
          <Alert tone="error">{formError}</Alert>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <TextField
          label="Full name"
          name="full_name"
          autoComplete="name"
          placeholder="Dana Amirova"
          hint="Optional — we'll derive one from your email if you skip it."
          value={fullName}
          error={fieldErrors.full_name}
          disabled={submitting}
          onChange={(event) => setFullName(event.target.value)}
        />

        <TextField
          label="Email"
          type="email"
          name="email"
          autoComplete="email"
          placeholder="dana@biletflow.kz"
          required
          value={email}
          error={fieldErrors.email}
          disabled={submitting}
          onChange={(event) => setEmail(event.target.value)}
        />

        <TextField
          label="Password"
          type="password"
          name="password"
          autoComplete="new-password"
          placeholder="At least 8 characters"
          required
          minLength={MIN_PASSWORD_LENGTH}
          value={password}
          error={fieldErrors.password}
          disabled={submitting}
          onChange={(event) => setPassword(event.target.value)}
        />

        <TextField
          label="Confirm password"
          type="password"
          name="confirm_password"
          autoComplete="new-password"
          required
          value={confirm}
          error={fieldErrors.confirm}
          disabled={submitting}
          onChange={(event) => setConfirm(event.target.value)}
        />

        <Button type="submit" loading={submitting} className="w-full">
          {submitting ? "Creating account…" : "Create account"}
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-foreground-muted">
        Already registered?{" "}
        <Link href="/login" className="font-medium text-brand hover:underline">
          Sign in
        </Link>
      </p>
    </>
  );
}
