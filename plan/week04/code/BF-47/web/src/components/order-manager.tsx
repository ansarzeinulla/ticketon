"use client";

import { useCallback, useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { formatKZT } from "@/lib/money";
import type { EventOrder } from "@/lib/types";

/** How an order's status reads, and how it looks. */
const STATUS_STYLES: Record<string, string> = {
  paid: "bg-success-soft text-success",
  completed: "bg-success-soft text-success",
  cancelled: "bg-surface-muted text-foreground-muted",
  pending: "bg-surface-muted text-foreground-muted",
};

function formatWhen(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** The organizer's order list: who bought what, newest first. */
export function OrderManager({ eventID }: { eventID: string }) {
  const [orders, setOrders] = useState<EventOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        setOrders(await api.eventOrders(eventID, signal));
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load the orders.");
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

  return (
    <section className="rounded-xl border border-border-subtle bg-surface">
      <div className="flex items-center justify-between gap-4 border-b border-border-subtle px-5 py-4">
        <div>
          <h2 className="text-base font-semibold">Orders</h2>
          <p className="text-xs text-foreground-muted">
            {orders.length === 1 ? "1 order" : `${orders.length} orders`}
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => {
            setLoading(true);
            void load();
          }}
          disabled={loading}
        >
          Refresh
        </Button>
      </div>

      {error && (
        <div className="p-5">
          <Alert tone="error">{error}</Alert>
        </div>
      )}

      {loading && orders.length === 0 ? (
        <p className="flex items-center gap-2 p-5 text-sm text-foreground-muted">
          <Spinner /> Loading orders…
        </p>
      ) : orders.length === 0 ? (
        <p className="p-5 text-sm text-foreground-muted">No orders yet.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-foreground-muted">
              <tr>
                <th className="px-4 py-2 font-medium">Buyer</th>
                <th className="px-4 py-2 font-medium">Order</th>
                <th className="px-4 py-2 text-right font-medium">Tickets</th>
                <th className="px-4 py-2 text-right font-medium">Total</th>
                <th className="px-4 py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((order) => (
                <tr key={order.id} className="border-t border-border-subtle align-top">
                  <td className="px-4 py-3">
                    <div className="font-medium">{order.buyer_name}</div>
                    <div className="text-xs text-foreground-muted">{order.buyer_email}</div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="font-mono text-xs">{order.order_number}</div>
                    <div className="text-xs text-foreground-muted">
                      {formatWhen(order.placed_at ?? order.created_at)}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div>{order.live_tickets}</div>
                    {order.live_tickets !== order.ticket_count && (
                      <div className="text-xs text-foreground-muted">
                        of {order.ticket_count} issued
                      </div>
                    )}
                    {order.checked_in > 0 && (
                      <div className="text-xs text-foreground-muted">
                        {order.checked_in} checked in
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right">{formatKZT(order.total_kzt)}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium capitalize ${
                        STATUS_STYLES[order.status] ?? "bg-surface-muted text-foreground-muted"
                      }`}
                    >
                      {order.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
