"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";

/**
 * Email confirmation (SRS 4.1).
 *
 * A token in the URL is redeemed on arrival, because that is what clicking the
 * link is meant to do. Without one the page offers a field, since in this MVP
 * the code is read off the API console.
 */
export function VerifyEmailView() {
  const searchParams = useSearchParams();
  const linkToken = searchParams.get("token");

  const [token, setToken] = useState(linkToken ?? "");
  const [state, setState] = useState<"idle" | "working" | "done" | "failed">(
    linkToken ? "working" : "idle",
  );
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!linkToken) return;

    let cancelled = false;
    void (async () => {
      try {
        await api.verifyEmail(linkToken);
        if (!cancelled) setState("done");
      } catch (cause) {
        if (cancelled) return;
        setError(
          cause instanceof ApiError
            ? "That confirmation code is invalid, expired or already used."
            : "Something went wrong. Please try again.",
        );
        setState("failed");
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [linkToken]);

  async function submitTyped() {
    setState("working");
    setError(null);
    try {
      await api.verifyEmail(token.trim());
      setState("done");
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? "That confirmation code is invalid, expired or already used."
          : "Something went wrong. Please try again.",
      );
      setState("failed");
    }
  }

  if (state === "done") {
    return (
      <>
        <div className="mb-6">
          <h1 className="text-2xl font-semibold tracking-tight">Email confirmed</h1>
        </div>
        <Alert tone="success">Your address is verified and your account is active.</Alert>
        <p className="mt-6 text-center text-sm text-foreground-muted">
          <Link href="/login" className="font-medium text-brand hover:underline">
            Continue to sign in
          </Link>
        </p>
      </>
    );
  }

  return (
    <>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight">Confirm your email</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          {state === "working"
            ? "Checking your code…"
            : "Paste the code from your confirmation email."}
        </p>
      </div>

      {error && (
        <div className="mb-4">
          <Alert tone="error">{error}</Alert>
        </div>
      )}

      <div className="space-y-4">
        <TextField
          label="Confirmation code"
          name="token"
          value={token}
          disabled={state === "working"}
          hint="In this MVP it is printed to the API console."
          onChange={(event) => setToken(event.target.value)}
        />
        <Button
          className="w-full"
          loading={state === "working"}
          disabled={token.trim() === ""}
          onClick={() => void submitTyped()}
        >
          Confirm email
        </Button>
      </div>
    </>
  );
}
