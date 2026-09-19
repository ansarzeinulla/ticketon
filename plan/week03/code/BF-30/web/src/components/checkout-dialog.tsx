"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";
import { formatKZT, formatTiyn, lineTotal } from "@/lib/money";
import type { TicketType } from "@/lib/types";

interface Line {
  type: TicketType;
  quantity: number;
}

/**
 * The simulated checkout.
 *
 * A native <dialog> gives the focus trap, Escape handling and backdrop for
 * free, so none of that has to be reimplemented.
 */
export function CheckoutDialog({
  eventID,
  eventTitle,
  lines,
  totalTiyn,
  onClose,
  onSoldOut,
}: {
  eventID: string;
  eventTitle: string;
  lines: Line[];
  totalTiyn: number;
  onClose: () => void;
  onSoldOut: () => void;
}) {
  const router = useRouter();
  const dialogRef = useRef<HTMLDialogElement>(null);

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // showModal() cannot be called during render, and the dialog only becomes
  // modal - backdrop, focus trap, inert background - when opened this way.
  useEffect(() => {
    const dialog = dialogRef.current;
    if (dialog && !dialog.open) dialog.showModal();
  }, []);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    setFieldErrors({});

    const errors: Record<string, string> = {};
    if (!name.trim()) errors.buyer_name = "Your name is required.";
    if (!email.trim()) errors.buyer_email = "Your email is required.";

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setSubmitting(true);
    try {
      const result = await api.checkout(eventID, {
        buyer_name: name.trim(),
        buyer_email: email.trim(),
        items: lines.map((line) => ({
          ticket_type_id: line.type.id,
          quantity: line.quantity,
        })),
      });

      // The confirmation lives at its own URL, so it survives a refresh and can
      // be shared or bookmarked.
      router.push(`/orders/${result.order.id}`);
    } catch (error) {
      if (error instanceof ApiError && error.isSoldOut) {
        setFormError(
          error.remaining === 0
            ? "These tickets sold out while you were choosing. The page has been refreshed."
            : `${error.message} The page has been refreshed with what is left.`,
        );
        onSoldOut();
      } else if (error instanceof ApiError) {
        setFieldErrors(error.fields);
        setFormError(
          Object.keys(error.fields).length > 0
            ? "Please correct the highlighted fields."
            : error.message,
        );
      } else {
        setFormError("The payment simulation failed. Please try again.");
      }
      setSubmitting(false);
    }
  }

  return (
    <dialog
      ref={dialogRef}
      onClose={onClose}
      onCancel={onClose}
      aria-labelledby="checkout-title"
      className="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-2xl border border-border-subtle bg-surface p-0 text-foreground backdrop:bg-black/50"
    >
      <div className="space-y-5 p-6">
        <div>
          <h2 id="checkout-title" className="text-lg font-semibold tracking-tight">
            Checkout
          </h2>
          <p className="mt-1 text-sm text-foreground-muted">{eventTitle}</p>
        </div>

        <ul className="divide-y divide-border-subtle rounded-lg border border-border-subtle">
          {lines.map((line) => (
            <li key={line.type.id} className="flex items-center justify-between gap-4 px-4 py-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{line.type.name}</p>
                <p className="text-xs text-foreground-muted">
                  {line.quantity} × {formatKZT(line.type.price_kzt)}
                </p>
              </div>
              <p className="shrink-0 text-sm font-medium tabular-nums">
                {lineTotal(line.type.price_kzt, line.quantity)}
              </p>
            </li>
          ))}
          <li className="flex items-center justify-between gap-4 bg-surface-muted/50 px-4 py-3">
            <p className="text-sm font-semibold">Total</p>
            <p className="text-base font-semibold tabular-nums">{formatTiyn(totalTiyn)}</p>
          </li>
        </ul>

        {formError && <Alert tone="error">{formError}</Alert>}

        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <TextField
            label="Full name"
            name="buyer_name"
            autoComplete="name"
            placeholder="Nurlan Amanov"
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
            placeholder="nurlan@example.kz"
            hint="Your tickets are issued to this address."
            required
            value={email}
            error={fieldErrors.buyer_email}
            disabled={submitting}
            onChange={(event) => setEmail(event.target.value)}
          />

          {/* SRS 4.6: a demonstration payment must never look like a real one. */}
          <p className="rounded-lg border border-warning/30 bg-warning-soft px-3 py-2 text-xs text-warning">
            Simulated payment — no card is charged and no money moves. This is a demonstration checkout.
          </p>

          <div className="flex flex-wrap gap-3">
            <Button type="submit" loading={submitting} className="flex-1">
              {submitting
                ? "Processing…"
                : `Pay ${formatTiyn(totalTiyn)} (simulated)`}
            </Button>
            <Button
              type="button"
              variant="secondary"
              disabled={submitting}
              onClick={() => dialogRef.current?.close()}
            >
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </dialog>
  );
}
