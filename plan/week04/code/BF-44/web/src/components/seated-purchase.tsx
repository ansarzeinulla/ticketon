"use client";

import { useState } from "react";

import { SeatCheckout } from "@/components/seat-checkout";
import { SeatMapPicker, type SeatChoice } from "@/components/seat-map";

/**
 * The assigned-seating purchase flow (SRS 4.3.1): pick seats, then pay for
 * them. Two components, one at a time, so the payment form never shows seats
 * the map no longer has.
 */
export function SeatedPurchase({ eventID }: { eventID: string }) {
  const [choice, setChoice] = useState<SeatChoice | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  if (choice) {
    return (
      <SeatCheckout
        eventID={eventID}
        choice={choice}
        onBack={() => setChoice(null)}
        onSeatGone={() => {
          setChoice(null);
          setNotice("Somebody else has just bought one of those seats. Please choose again.");
        }}
      />
    );
  }

  return (
    <div className="space-y-3">
      {notice && (
        <p
          className="rounded-lg border border-border-subtle bg-surface-muted px-4 py-3 text-sm text-foreground-muted"
          role="status"
        >
          {notice}
        </p>
      )}
      {/* Remounted after a lost seat, so the map is re-read rather than showing
          the seats as they were before somebody else took one. */}
      <SeatMapPicker
        key={notice ?? "initial"}
        eventID={eventID}
        onChosen={(picked) => {
          setNotice(null);
          setChoice(picked);
        }}
      />
    </div>
  );
}
