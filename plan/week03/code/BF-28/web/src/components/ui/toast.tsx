"use client";

import { useEffect } from "react";

type Tone = "success" | "error";

const tones: Record<Tone, string> = {
  success: "border-success/30 bg-success-soft text-success",
  error: "border-danger/30 bg-danger-soft text-danger",
};

/**
 * A short confirmation that floats over the page and goes away on its own -
 * "Event published", "Could not cancel". Render it only while there is a
 * message; `onDismiss` clears that message.
 */
export function Toast({
  message,
  tone = "success",
  onDismiss,
  durationMs = 4000,
}: {
  message: string;
  tone?: Tone;
  onDismiss: () => void;
  durationMs?: number;
}) {
  useEffect(() => {
    const timer = setTimeout(onDismiss, durationMs);
    return () => clearTimeout(timer);
  }, [message, onDismiss, durationMs]);

  return (
    <div
      role="status"
      aria-live="polite"
      className={`fixed bottom-6 right-6 z-50 flex max-w-sm items-start gap-3 rounded-xl border px-4 py-3 text-sm shadow-lg ${tones[tone]}`}
    >
      <span className="flex-1">{message}</span>
      <button
        type="button"
        onClick={onDismiss}
        className="font-medium opacity-70 hover:opacity-100"
        aria-label="Dismiss"
      >
        ×
      </button>
    </div>
  );
}
