"use client";

import { useState } from "react";

import { SeatCheckout } from "@/components/seat-checkout";
import { SeatMapPicker } from "@/components/seat-map";

/**
 * The assigned-seating purchase flow (SRS 4.3.1, 4.6).
 *
 * Two steps, one at a time: pick seats, then pay for the reservation those
 * seats produced. They are separate components because they are separate
 * states of the world - before the hold nothing is reserved, after it a
 * timer is running - and pretending otherwise is how a checkout ends up
 * showing a countdown for seats nobody has claimed.
 */
export function SeatedPurchase({
  eventID,
  eventSlug,
}: {
  eventID: string;
  eventSlug: string;
}) {
  const [orderID, setOrderID] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  if (orderID) {
    return (
      <SeatCheckout
        orderID={orderID}
        eventSlug={eventSlug}
        onExpired={() => {
          setOrderID(null);
          setNotice(
            "Your reservation ran out and the seats are back on sale. Please choose again.",
          );
        }}
        onCancelled={() => {
          setOrderID(null);
          setNotice("Those seats are back on sale.");
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
      {/* Remounted whenever a hold ends, so the map is re-read rather than
          showing the seats as they were before somebody else took one. */}
      <SeatMapPicker
        key={notice ?? "initial"}
        eventID={eventID}
        onHeld={(id) => {
          setNotice(null);
          setOrderID(id);
        }}
      />
    </div>
  );
}
