"use client";

import { useCallback, useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { formatKZT } from "@/lib/money";
import type { EventOrder } from "@/lib/types";

/** How an order's status reads, and how it looks. */
const STATUS_STYLES: Record<string, string> = {
  paid: "bg-success-soft text-success",
  completed: "bg-success-soft text-success",
  refunded: "bg-danger-soft text-danger",
  partially_refunded: "bg-warning-soft text-warning",
  cancelled: "bg-surface-muted text-foreground-muted",
  pending: "bg-surface-muted text-foreground-muted",
};

const STATUS_LABELS: Record<string, string> = {
  paid: "Paid",
  completed: "Completed",
  refunded: "Refunded",
  partially_refunded: "Partly refunded",
  cancelled: "Cancelled",
  pending: "Pending",
};

function statusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status;
}

function formatWhen(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/**
 * The organizer's attendee view: who bought what, and the Refund button
 * (SRS 4.9).
 *
 * Refunding is a full refund and cannot be undone, so it asks for confirmation
 * and an optional reason first rather than firing on a single click.
 */
export function OrderManager({
  eventID,
  onRefunded,
}: {
  eventID: string;
  onRefunded?: () => void;
}) {
  const [orders, setOrders] = useState<EventOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  // Which order is mid-confirmation, and what reason has been typed for it.
  const [confirming, setConfirming] = useState<string | null>(null);
  const [reason, setReason] = useState("");
  const [refunding, setRefunding] = useState<string | null>(null);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        setOrders(await api.eventOrders(eventID, signal));
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load orders.");
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

  async function refund(order: EventOrder) {
    setRefunding(order.id);
    try {
      const result = await api.refundOrder(order.id, reason.trim() || undefined);
      setNotice(
        `Refunded ${formatKZT(result.refund.amount_kzt)} to ${order.buyer_name}. ` +
          `${result.voided_tickets} ticket${result.voided_tickets === 1 ? "" : "s"} ` +
          `${result.voided_tickets === 1 ? "is" : "are"} now void.`,
      );
      setError(null);
      setConfirming(null);
      setReason("");
      await load();
      onRefunded?.();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "Could not refund this order.");
    } finally {
      setRefunding(null);
    }
  }

  /**
   * Withdraw a free registration (SRS 4.9).
   *
   * Deliberately its own action rather than a branch inside refund(): a free
   * order has no money to give back, and sending one to the refund endpoint
   * used to trip refunds_amount_chk and surface as a 500.
   */
  async function cancel(order: EventOrder) {
    setRefunding(order.id);
    try {
      const result = await api.cancelOrder(order.id, reason.trim() || undefined);
      setNotice(
        `Cancelled ${order.buyer_name}'s registration. ` +
          `${result.cancelled_tickets} ticket${result.cancelled_tickets === 1 ? "" : "s"} ` +
          `${result.cancelled_tickets === 1 ? "is" : "are"} now void.`,
      );
      setError(null);
      setConfirming(null);
      setReason("");
      await load();
      onRefunded?.();
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.message : "Could not cancel this registration.",
      );
    } finally {
      setRefunding(null);
    }
  }

  const refundedCount = orders.filter((o) => o.status === "refunded").length;

  return (
    <section className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">Orders &amp; attendees</h2>
        <p className="mt-1 text-sm text-foreground-muted">
          {loading
            ? "Loading…"
            : orders.length === 0
              ? "Nobody has bought a ticket yet."
              : `${orders.length} order${orders.length === 1 ? "" : "s"}` +
                (refundedCount > 0 ? ` · ${refundedCount} refunded` : "")}
        </p>
      </div>

      {error && <Alert>{error}</Alert>}
      {notice && (
        <Alert tone="success" title="Refund complete">
          {notice}
        </Alert>
      )}

      {orders.length > 0 && (
        <div className="overflow-x-auto rounded-lg border border-border-subtle">
          <table className="w-full min-w-[42rem] text-sm">
            <thead className="bg-surface-muted text-left text-xs uppercase tracking-wide text-foreground-muted">
              <tr>
                <th className="px-4 py-2 font-medium">Attendee</th>
                <th className="px-4 py-2 font-medium">Order</th>
                <th className="px-4 py-2 text-right font-medium">Tickets</th>
                <th className="px-4 py-2 text-right font-medium">Total</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 text-right font-medium">Action</th>
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
                  <td className="px-4 py-3 text-right">
                    <div>{formatKZT(order.total_kzt)}</div>
                    {order.status === "refunded" && (
                      <div className="text-xs text-danger">
                        −{formatKZT(order.refunded_kzt)} refunded
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${
                        STATUS_STYLES[order.status] ?? "bg-surface-muted text-foreground-muted"
                      }`}
                    >
                      {statusLabel(order.status)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    {order.refundable || order.cancellable ? (
                      <Button
                        variant="danger"
                        size="sm"
                        onClick={() => {
                          setConfirming(order.id);
                          setReason("");
                          setNotice(null);
                        }}
                        disabled={refunding !== null}
                        data-testid={`order-action-${order.id}`}
                      >
                        {order.refundable ? "Refund order" : "Cancel registration"}
                      </Button>
                    ) : (
                      <span className="text-xs text-foreground-muted">—</span>
                    )}

                    {confirming === order.id && (
                      <div className="mt-3 space-y-2 rounded-lg border border-danger/30 bg-danger-soft/30 p-3 text-left">
                        <p className="text-xs text-foreground">
                          {order.refundable ? (
                            <>
                              Refund {formatKZT(order.total_kzt)} to {order.buyer_name} and
                              void {order.live_tickets} ticket
                              {order.live_tickets === 1 ? "" : "s"}?
                            </>
                          ) : (
                            <>
                              Cancel {order.buyer_name}&rsquo;s registration and void{" "}
                              {order.live_tickets} ticket
                              {order.live_tickets === 1 ? "" : "s"}? Nothing was charged,
                              so there is no refund to make.
                            </>
                          )}{" "}
                          The QR codes stop working immediately and this cannot be undone.
                        </p>
                        <input
                          type="text"
                          value={reason}
                          onChange={(e) => setReason(e.target.value)}
                          placeholder="Reason (optional, shown to the attendee)"
                          maxLength={500}
                          className="w-full rounded-md border border-border-subtle bg-surface px-2 py-1.5 text-xs"
                        />
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setConfirming(null)}
                            disabled={refunding === order.id}
                          >
                            Cancel
                          </Button>
                          <Button
                            variant="danger"
                            size="sm"
                            loading={refunding === order.id}
                            onClick={() =>
                              void (order.refundable ? refund(order) : cancel(order))
                            }
                            data-testid={`confirm-${order.id}`}
                          >
                            {order.refundable ? "Confirm refund" : "Cancel this registration"}
                          </Button>
                        </div>
                      </div>
                    )}
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
