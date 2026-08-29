"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { CheckoutDialog } from "@/components/checkout-dialog";
import { PromoBox } from "@/components/promo-box";
import { formatKZT, formatTiyn, toTiyn } from "@/lib/money";
import type { PromoPreview, TicketType } from "@/lib/types";

export function TicketSelector({
  eventID,
  eventTitle,
  ticketTypes,
  onSale,
  soldOut,
  campaignToken,
}: {
  eventID: string;
  eventTitle: string;
  ticketTypes: TicketType[];
  onSale: boolean;
  soldOut: boolean;
  /** Present when the attendee arrived through a campaign QR or link. */
  campaignToken?: string;
}) {
  const router = useRouter();
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [checkoutOpen, setCheckoutOpen] = useState(false);
  const [promo, setPromo] = useState<PromoPreview | null>(null);

  const selected = useMemo(
    () =>
      ticketTypes
        .map((type) => ({ type, quantity: quantities[type.id] ?? 0 }))
        .filter((line) => line.quantity > 0),
    [ticketTypes, quantities],
  );

  const totalTiyn = selected.reduce(
    (sum, line) => sum + toTiyn(line.type.price_kzt) * line.quantity,
    0,
  );
  const totalTickets = selected.reduce((sum, line) => sum + line.quantity, 0);

  /** A type can never be selected beyond what is left, or past its order cap. */
  function limitFor(type: TicketType): number {
    return Math.min(type.quantity_remaining, type.max_per_order);
  }

  function setQuantity(type: TicketType, next: number) {
    const clamped = Math.max(0, Math.min(next, limitFor(type)));

    setQuantities((current) => {
      const updated = { ...current, [type.id]: clamped };

      // A discount priced against a basket means nothing once the basket is
      // empty, so it is dropped here - in the handler that emptied it.
      const anyLeft = Object.values(updated).some((quantity) => quantity > 0);
      if (!anyLeft) setPromo(null);

      return updated;
    });
  }

  if (ticketTypes.length === 0) {
    return (
      <section className="rounded-xl border border-border-subtle bg-surface p-6">
        <h2 className="text-lg font-semibold tracking-tight">Tickets</h2>
        <p className="mt-2 text-sm text-foreground-muted">
          The organizer has not published any ticket types for this event yet.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold tracking-tight">Tickets</h2>

      {soldOut && (
        <Alert tone="info" title="Sold out">
          Every ticket for this event has been taken.
        </Alert>
      )}
      {!onSale && !soldOut && (
        <Alert tone="info" title="Not on sale">
          Tickets for this event are not currently available.
        </Alert>
      )}

      <ul className="space-y-2">
        {ticketTypes.map((type) => {
          const limit = limitFor(type);
          const quantity = quantities[type.id] ?? 0;
          const exhausted = type.quantity_remaining === 0;

          return (
            <li
              key={type.id}
              className="flex flex-wrap items-center gap-4 rounded-xl border border-border-subtle bg-surface p-5"
            >
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h3 className="font-medium">{type.name}</h3>
                  {type.is_free && (
                    <span className="rounded-full bg-brand-soft px-2 py-0.5 text-xs font-medium text-brand-strong">
                      Free
                    </span>
                  )}
                </div>
                {type.description && (
                  <p className="mt-1 text-sm text-foreground-muted">{type.description}</p>
                )}
                <p className="mt-1 text-sm">
                  <span className="font-semibold">{formatKZT(type.price_kzt)}</span>
                  <span
                    className={`ml-3 text-xs ${
                      exhausted ? "text-danger" : "text-foreground-muted"
                    }`}
                  >
                    {exhausted
                      ? "Sold out"
                      : `${type.quantity_remaining} of ${type.quantity_total} left`}
                  </span>
                </p>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="secondary"
                  aria-label={`Remove one ${type.name}`}
                  disabled={!onSale || quantity === 0}
                  onClick={() => setQuantity(type, quantity - 1)}
                >
                  −
                </Button>
                <span
                  aria-live="polite"
                  className="w-8 text-center font-mono text-sm tabular-nums"
                >
                  {quantity}
                </span>
                <Button
                  size="sm"
                  variant="secondary"
                  aria-label={`Add one ${type.name}`}
                  disabled={!onSale || quantity >= limit}
                  onClick={() => setQuantity(type, quantity + 1)}
                >
                  +
                </Button>
              </div>
            </li>
          );
        })}
      </ul>

      <div className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-border-subtle bg-surface-muted/50 p-5">
        <div>
          <p className="text-xs text-foreground-muted">
            {totalTickets === 0
              ? "No tickets selected"
              : `${totalTickets} ticket${totalTickets === 1 ? "" : "s"}`}
          </p>
          <p className="text-xl font-semibold tabular-nums">{formatTiyn(totalTiyn)}</p>
        </div>

        <Button disabled={!onSale || totalTickets === 0} onClick={() => setCheckoutOpen(true)}>
          Get tickets
        </Button>
      </div>

      <PromoBox
        eventID={eventID}
        campaignToken={campaignToken}
        lines={selected}
        promo={promo}
        onChange={setPromo}
      />

      {checkoutOpen && (
        <CheckoutDialog
          eventID={eventID}
          eventTitle={eventTitle}
          lines={selected}
          totalTiyn={totalTiyn}
          promo={promo}
          campaignToken={campaignToken}
          onClose={() => setCheckoutOpen(false)}
          onSoldOut={() => {
            // Stock moved under us: pull fresh counts from the server.
            setQuantities({});
            router.refresh();
          }}
          onPromoRejected={() => setPromo(null)}
        />
      )}
    </section>
  );
}
