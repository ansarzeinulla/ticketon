"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import type { SeatChoice } from "@/components/seat-map";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";
import { formatKZT, formatTiyn } from "@/lib/money";

/**
 * Paying for the seats chosen on the map (SRS 4.3.1, 4.6).
 *
 * Only the seat ids travel: what each seat costs is decided by the server from
 * where it is, so the total shown here is a preview and the order is the truth.
 * Nothing is reserved while the attendee types; if somebody else buys a seat
 * first, the checkout says so and the map is shown again.
 */
export function SeatCheckout({
  eventID,
  choice,
  onBack,
  onSeatGone,
}: {
  eventID: string;
  choice: SeatChoice;
  onBack: () => void;
  /** A seat was sold to somebody else while this attendee was paying. */
  onSeatGone: () => void;
}) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setFieldErrors({});

    if (!name.trim() || !email.trim()) {
      setFieldErrors({
        ...(name.trim() ? {} : { buyer_name: "Your name is required." }),
        ...(email.trim() ? {} : { buyer_email: "Your email is required." }),
      });
      return;
    }

    setSubmitting(true);
    try {
      const result = await api.checkout(eventID, {
        buyer_name: name.trim(),
        buyer_email: email.trim(),
        seat_ids: choice.seatIDs,
      });
      router.push(`/orders/${result.order.id}`);
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "seat_unavailable") {
        onSeatGone();
        return;
      }
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        setError(
          Object.keys(cause.fields).length > 0
            ? "Please correct the highlighted fields."
            : cause.message,
        );
      } else {
        setError("The payment simulation failed. Please try again.");
      }
      setSubmitting(false);
    }
  }

  return (
    <section className="space-y-4 rounded-xl border border-border-subtle bg-surface p-5">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">Your seats</h2>
        <p className="mt-1 text-sm text-foreground-muted">
          {choice.lines.length} seat{choice.lines.length === 1 ? "" : "s"}
        </p>
      </div>

      <ul className="divide-y divide-border-subtle rounded-lg border border-border-subtle text-sm">
        {choice.lines.map((line) => (
          <li key={line.id} className="flex items-center justify-between px-4 py-2">
            <span>{line.section}</span>
            <span className="tabular-nums">{formatKZT(line.price)}</span>
          </li>
        ))}
        <li className="flex items-center justify-between bg-surface-muted/50 px-4 py-2 font-semibold">
          <span>Total</span>
          <span className="tabular-nums">{formatTiyn(choice.totalTiyn)}</span>
        </li>
      </ul>

      {error && <Alert tone="error">{error}</Alert>}

      <form onSubmit={submit} noValidate className="space-y-4">
        <TextField
          label="Full name"
          name="buyer_name"
          autoComplete="name"
          required
          value={name}
          error={fieldErrors.buyer_name}
          disabled={submitting}
          onChange={(event) => setName(event.target.value)}
        />
        <TextField
          label="Email"
          type="email"
          name="buyer_email"
          autoComplete="email"
          hint="Your tickets are issued to this address."
          required
          value={email}
          error={fieldErrors.buyer_email}
          disabled={submitting}
          onChange={(event) => setEmail(event.target.value)}
        />

        {/* SRS 4.6: a demonstration payment must never look like a real one. */}
        <p className="rounded-lg border border-warning/30 bg-warning-soft px-3 py-2 text-xs text-warning">
          Simulated payment — no card is charged and no money moves.
        </p>

        <div className="flex flex-wrap gap-3">
          <Button type="submit" loading={submitting} className="flex-1">
            {submitting ? "Processing…" : `Pay ${formatTiyn(choice.totalTiyn)} (simulated)`}
          </Button>
          <Button type="button" variant="secondary" disabled={submitting} onClick={onBack}>
            Back to the map
          </Button>
        </div>
      </form>
    </section>
  );
}
