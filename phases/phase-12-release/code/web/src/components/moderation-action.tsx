"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";

/**
 * A moderation control that asks before it acts (SRS 4.12).
 *
 * Every action behind this button is visible to somebody else - an organizer
 * whose event stops selling, a person locked out of their account - so none of
 * them fire on a single click. The confirmation names the consequence rather
 * than asking "are you sure", and offers a reason the audit log keeps.
 *
 * Errors are shown in place instead of thrown away: a 409 saying "already
 * suspended" is the answer to the administrator's question, not a failure.
 */
export function ModerationAction({
  label,
  confirmLabel,
  consequence,
  tone = "danger",
  withReason = true,
  onConfirm,
  onDone,
}: {
  /** The resting button, e.g. "Suspend". */
  label: string;
  /** The button inside the confirmation, e.g. "Suspend this event". */
  confirmLabel: string;
  /** One sentence saying what will happen, in the second person. */
  consequence: string;
  tone?: "danger" | "secondary";
  withReason?: boolean;
  onConfirm: (reason: string) => Promise<unknown>;
  onDone?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function confirm() {
    setBusy(true);
    setError(null);
    try {
      await onConfirm(reason.trim());
      setOpen(false);
      setReason("");
      onDone?.();
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.message : "That did not work. Try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  if (!open) {
    return (
      <Button variant={tone === "danger" ? "danger" : "secondary"} size="sm" onClick={() => setOpen(true)}>
        {label}
      </Button>
    );
  }

  return (
    <div className="space-y-2 rounded-lg border border-border-subtle bg-surface-muted p-3">
      <p className="text-xs text-foreground-muted">{consequence}</p>

      {withReason && (
        <label className="block text-xs">
          <span className="text-foreground-muted">Reason (kept in the audit log)</span>
          <input
            type="text"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            maxLength={500}
            className="mt-1 w-full rounded-md border border-border-subtle bg-surface px-2 py-1 text-sm"
            placeholder="Optional"
          />
        </label>
      )}

      {error && <Alert>{error}</Alert>}

      <div className="flex gap-2">
        <Button
          variant={tone === "danger" ? "danger" : "primary"}
          size="sm"
          loading={busy}
          onClick={confirm}
        >
          {confirmLabel}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setOpen(false);
            setError(null);
          }}
        >
          Cancel
        </Button>
      </div>
    </div>
  );
}
