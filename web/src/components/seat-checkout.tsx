"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { toAnalyticsValue, trackPurchase } from "@/lib/analytics";
import { ApiError, api } from "@/lib/api";
import { formatKZT } from "@/lib/money";
import type { Hold } from "@/lib/types";

/**
 * Paying for held seats (SRS 4.6, 4.3.1).
 *
 * The countdown is the point: an attendee holding the last two seats in the
 * house needs to know how long they have, and a reservation that expires
 * silently while somebody types their name is worse than one that never
 * existed.
 */
export function SeatCheckout({
  orderID,
  eventSlug,
  onExpired,
  onCancelled,
}: {
  orderID: string;
  eventSlug: string;
  /** The hold ran out; the caller puts the seat map back. */
  onExpired: () => void;
  onCancelled: () => void;
}) {
  const router = useRouter();

  const [hold, setHold] = useState<Hold | null>(null);
  const [remaining, setRemaining] = useState<number | null>(null);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  /** True once the reservation has run out. Derived, never stored. */
  const expired = remaining !== null && remaining <= 0;

  useEffect(() => {
    const controller = new AbortController();

    void (async () => {
      try {
        const basket = await api.getHold(orderID, controller.signal);
        setHold(basket);
        setRemaining(basket.seconds_remaining);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        // The basket is already gone - expired, paid or released. Setting
        // state here is safe: it happens after an await, not during the
        // effect body.
        setRemaining(0);
      }
    })();

    return () => controller.abort();
  }, [orderID]);

  // The countdown runs off the server's own number of seconds, not off a
  // deadline compared against the browser's clock: a device an hour fast would
  // otherwise show a basket as expired the moment it was created.
  useEffect(() => {
    if (remaining === null || remaining <= 0) return;

    const timer = setTimeout(() => setRemaining((value) => (value ?? 1) - 1), 1000);
    return () => clearTimeout(timer);
  }, [remaining]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    setFieldErrors({});

    try {
      const result = await api.confirmHold(orderID, {
        buyer_name: name.trim(),
        buyer_email: email.trim(),
      });

      trackPurchase({
        slug: eventSlug,
        valueKZT: toAnalyticsValue(result.order.total_kzt),
        tickets: result.tickets.length,
        discounted: toAnalyticsValue(result.order.discount_kzt) > 0,
      });

      router.push(`/orders/${result.order.id}`);
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        if (cause.code === "hold_expired") {
          setRemaining(0);
          return;
        }
        setError(Object.keys(cause.fields).length > 0 ? null : cause.message);
      } else {
        setError("Something went wrong. Please try again.");
      }
      setSubmitting(false);
    }
  }

  async function cancel() {
    try {
      await api.releaseHold(orderID);
    } catch {
      // The hold expires on its own; nothing useful to say if the release
      // request itself fails.
    }
    onCancelled();
  }

  // Rendered rather than acted on: telling the attendee what happened and
  // letting them choose again beats a page that rearranges itself under them.
  if (expired) {
    return (
      <div className="space-y-3 rounded-xl border border-border-subtle bg-surface p-5">
        <Alert tone="info" title="Your reservation ran out">
          The seats are back on sale. Pick them again to carry on.
        </Alert>
        <Button onClick={onExpired}>Choose seats again</Button>
      </div>
    );
  }

  if (!hold) {
    return <p className="text-sm text-foreground-muted">Checking your reservation…</p>;
  }

  const minutes = Math.floor((remaining ?? 0) / 60);
  const seconds = String((remaining ?? 0) % 60).padStart(2, "0");

  return (
    <section className="space-y-4 rounded-xl border border-border-subtle bg-surface p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Your seats are held</h2>
          <p className="mt-1 text-sm text-foreground-muted">
            {hold.items.length} seat{hold.items.length === 1 ? "" : "s"} ·{" "}
            {hold.order_number}
          </p>
        </div>
        <p
          className={`text-sm font-semibold tabular-nums ${
            (remaining ?? 0) < 120 ? "text-danger" : "text-foreground-muted"
          }`}
          role="timer"
          aria-live="off"
        >
          {minutes}:{seconds} left
        </p>
      </div>

      <dl className="space-y-1 text-sm">
        <div className="flex justify-between">
          <dt className="text-foreground-muted">Tickets</dt>
          <dd className="tabular-nums">{formatKZT(hold.subtotal_kzt)}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-foreground-muted">Processing fee</dt>
          <dd className="tabular-nums">
            {formatKZT(hold.estimated_processing_fee_kzt)}
          </dd>
        </div>
        <div className="flex justify-between border-t border-border-subtle pt-1 font-semibold">
          <dt>Total</dt>
          <dd className="tabular-nums">{formatKZT(hold.estimated_total_kzt)}</dd>
        </div>
      </dl>

      {error && <Alert>{error}</Alert>}

      <form onSubmit={submit} className="space-y-3" noValidate>
        <TextField
          label="Your name"
          name="buyer_name"
          required
          value={name}
          error={fieldErrors.buyer_name}
          onChange={(event) => setName(event.target.value)}
        />
        <TextField
          label="Email"
          type="email"
          name="buyer_email"
          required
          hint="Your tickets are sent here."
          value={email}
          error={fieldErrors.buyer_email}
          onChange={(event) => setEmail(event.target.value)}
        />

        <div className="flex flex-wrap gap-2">
          <Button type="submit" loading={submitting}>
            Pay {formatKZT(hold.estimated_total_kzt)} (simulated)
          </Button>
          <Button
            type="button"
            variant="ghost"
            disabled={submitting}
            onClick={() => void cancel()}
          >
            Release the seats
          </Button>
        </div>
      </form>
    </section>
  );
}
