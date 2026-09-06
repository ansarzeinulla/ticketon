"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { formatKZT } from "@/lib/money";
import type { BuyerOrder } from "@/lib/types";

/**
 * The attendee's own orders (SRS 4.9, SRS 4.7).
 *
 * An order page is reachable by its unguessable id alone, which is what lets a
 * guest keep their tickets without an account. This is the other half of that
 * rule: once somebody has signed up, their tickets have to be findable from the
 * account itself rather than by hunting for the confirmation email. Guest
 * orders placed with the same address are claimed automatically by the API.
 */
export default function MyOrdersPage() {
  const [orders, setOrders] = useState<BuyerOrder[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (signal?: AbortSignal) => {
    try {
      const data = await api.listMyOrders(signal);
      setOrders(data.orders);
      setError(null);
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setError(cause instanceof ApiError ? cause.message : "Could not load your tickets.");
      setOrders([]);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  if (orders === null) {
    return (
      <div className="flex items-center gap-3 text-sm text-foreground-muted">
        <Spinner />
        Loading your tickets…
      </div>
    );
  }

  return (
    <section className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">My tickets</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          Every order placed with this account, newest first.
        </p>
      </div>

      {error && <Alert>{error}</Alert>}

      {orders.length === 0 && !error ? (
        <div className="rounded-xl border border-border-subtle bg-surface p-6 text-sm text-foreground-muted">
          You have no tickets yet.{" "}
          <Link href="/events" className="font-medium text-brand underline">
            Browse events
          </Link>
          .
        </div>
      ) : (
        <ul className="space-y-3">
          {orders.map((order) => (
            <li key={order.id}>
              <Link
                href={`/orders/${order.id}`}
                className="block rounded-xl border border-border-subtle bg-surface p-5 transition hover:border-brand"
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h2 className="font-semibold tracking-tight">{order.event_title}</h2>
                    <p className="mt-1 text-sm text-foreground-muted">
                      {new Date(order.event_starts_at).toLocaleString("en-GB", {
                        dateStyle: "medium",
                        timeStyle: "short",
                        timeZone: order.timezone,
                      })}{" "}
                      ({order.timezone})
                    </p>
                    <p className="mt-1 text-xs text-foreground-muted">
                      {order.order_number} · {order.ticket_count} ticket
                      {order.ticket_count === 1 ? "" : "s"}
                      {order.live_tickets !== order.ticket_count &&
                        ` · ${order.live_tickets} still valid`}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold tabular-nums">{formatKZT(order.total_kzt)}</p>
                    <p className="mt-1 text-xs uppercase tracking-wide text-foreground-muted">
                      {order.status}
                    </p>
                  </div>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
