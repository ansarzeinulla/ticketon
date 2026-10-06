"use client";

import { useCallback, useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { formatKZT } from "@/lib/money";
import type { Activation, ActivationStep, ActivationSubmission } from "@/lib/types";

/**
 * The checklist, in the order the organizer works through it.
 *
 * `key` matches the server's `outstanding` values, so what is ticked here is
 * decided by the API rather than by this file keeping its own tally.
 */
const STEPS: {
  key: ActivationStep;
  field: keyof ActivationSubmission;
  title: string;
  detail: string;
}[] = [
  {
    key: "identity",
    field: "confirm_identity",
    title: "Confirm your identity",
    detail:
      "A real platform would check a document or a company registration here. " +
      "For this MVP, confirming stands in for that check.",
  },
  {
    key: "payout",
    field: "confirm_payout",
    title: "Confirm where payouts go",
    detail:
      "The account that would receive ticket revenue after the event, minus fees.",
  },
  {
    key: "terms",
    field: "accept_terms",
    title: "Accept the seller terms",
    detail:
      "Refund obligations, what you may sell, and what happens if an event is cancelled.",
  },
  {
    key: "fee",
    field: "pay_activation_fee",
    title: "Pay the activation fee",
    detail: "Recorded as a simulated payment. No money moves anywhere.",
  },
];

/**
 * Paid-sales activation for one event (SRS 4.5).
 *
 * Renders nothing at all when the event has no paid tickets: activation exists
 * to clear an organizer to take money, and nagging about a checklist that
 * gates nothing is noise.
 */
export function ActivationChecklist({
  eventID,
  onActivated,
}: {
  eventID: string;
  onActivated?: () => void;
}) {
  const [activation, setActivation] = useState<Activation | null>(null);
  const [open, setOpen] = useState(false);
  const [checked, setChecked] = useState<Set<ActivationStep>>(new Set());
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const next = await api.eventActivation(eventID, signal);
        setActivation(next);
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load activation.");
      } finally {
        setLoading(false);
      }
    },
    [eventID],
  );

  useEffect(() => {
    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const toggle = (step: ActivationStep) => {
    setChecked((current) => {
      const next = new Set(current);
      if (next.has(step)) next.delete(step);
      else next.add(step);
      return next;
    });
  };

  async function submit() {
    if (!activation) return;

    const body: ActivationSubmission = {};
    for (const step of STEPS) {
      if (checked.has(step.key)) body[step.field] = true;
    }

    setSubmitting(true);
    try {
      const next = await api.advanceActivation(eventID, body);
      setActivation(next);
      setChecked(new Set());
      setError(null);
      if (next.is_active) {
        setOpen(false);
        onActivated?.();
      }
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "Could not activate paid sales.");
    } finally {
      setSubmitting(false);
    }
  }

  if (loading || !activation) return null;

  // A free event never needs activating.
  if (!activation.required_for_sales) return null;

  if (activation.status === "suspended") {
    return (
      <Alert tone="error" title="Paid sales suspended">
        BiletFlow has suspended paid ticket sales for this event
        {activation.suspension_reason ? `: ${activation.suspension_reason}` : "."} Free
        registration is unaffected. Contact support to have this reviewed.
      </Alert>
    );
  }

  if (activation.is_active) {
    return (
      <Alert tone="success" title="Paid sales are active">
        Activated on{" "}
        {activation.activated_at
          ? new Date(activation.activated_at).toLocaleDateString()
          : "this event"}
        . The simulated {formatKZT(activation.activation_fee_kzt)} activation fee was
        recorded.
      </Alert>
    );
  }

  const done = STEPS.length - activation.outstanding.length;

  return (
    <div className="rounded-lg border border-warning/40 bg-warning-soft/40 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="max-w-2xl">
          <p className="font-semibold text-foreground">
            Your paid tickets are not on sale yet
          </p>
          <p className="mt-1 text-sm text-foreground-muted">
            This event has paid ticket types, so BiletFlow needs you to complete a short
            activation checklist before the public can buy them. Free tickets, if you have
            any, are unaffected.
          </p>
          <p className="mt-2 text-xs text-foreground-muted">
            {done} of {STEPS.length} steps done
          </p>
        </div>

        {!open && (
          <Button onClick={() => setOpen(true)}>Activate paid sales</Button>
        )}
      </div>

      {open && (
        <div className="mt-4 space-y-3 border-t border-border-subtle pt-4">
          {STEPS.map((step) => {
            const complete = !activation.outstanding.includes(step.key);
            return (
              <label
                key={step.key}
                className={`flex gap-3 rounded-lg border p-3 ${
                  complete
                    ? "border-success/30 bg-success-soft/40"
                    : "border-border-subtle bg-surface"
                } ${complete ? "" : "cursor-pointer"}`}
              >
                <input
                  type="checkbox"
                  className="mt-1 size-4 shrink-0 accent-brand"
                  checked={complete || checked.has(step.key)}
                  disabled={complete || submitting}
                  onChange={() => toggle(step.key)}
                />
                <span className="min-w-0">
                  <span className="flex flex-wrap items-center gap-2 text-sm font-medium">
                    {step.title}
                    {step.key === "fee" && (
                      <span className="text-xs font-normal text-foreground-muted">
                        {formatKZT(activation.activation_fee_kzt)} · simulated
                      </span>
                    )}
                    {complete && (
                      <span className="text-xs font-normal text-success">done</span>
                    )}
                  </span>
                  <span className="mt-1 block text-xs text-foreground-muted">
                    {step.detail}
                  </span>
                </span>
              </label>
            );
          })}

          {error && <Alert>{error}</Alert>}

          <div className="flex flex-wrap items-center gap-2">
            <Button onClick={() => void submit()} loading={submitting} disabled={checked.size === 0}>
              {checked.size === activation.outstanding.length
                ? "Complete activation"
                : `Save ${checked.size || "no"} step${checked.size === 1 ? "" : "s"}`}
            </Button>
            <Button variant="ghost" onClick={() => setOpen(false)} disabled={submitting}>
              Close
            </Button>
            <span className="text-xs text-foreground-muted">
              Nothing is charged. This is a simulated activation for an academic MVP.
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
