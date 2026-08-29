import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { OrderSupport } from "@/components/order-support";
import { TicketCard } from "@/components/ticket-card";
import { ApiError, api } from "@/lib/api";
import { formatKZT } from "@/lib/money";

export const metadata: Metadata = { title: "Order confirmation" };

/**
 * The order confirmation, at its own URL so it survives a refresh and can be
 * shared. The order id is a UUID, which is the unguessable capability that
 * makes this safe to serve without a login - a guest checkout has no account
 * to authenticate against.
 */
export default async function OrderConfirmationPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let data;
  try {
    data = await api.getOrder(id);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) notFound();
    throw error;
  }

  const { order, items, tickets } = data;

  return (
    <main className="mx-auto w-full max-w-2xl px-4 py-10 sm:px-6">
      <div className="rounded-2xl border border-border-subtle bg-surface p-6 sm:p-8">
        <div className="flex items-start gap-4">
          <span
            aria-hidden="true"
            className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-success-soft text-success"
          >
            ✓
          </span>
          <div>
            <h1 className="text-xl font-semibold tracking-tight">Order confirmed</h1>
            <p className="mt-1 text-sm text-foreground-muted">
              {tickets.length} ticket{tickets.length === 1 ? "" : "s"} issued to{" "}
              {order.buyer_email}.
            </p>
          </div>
        </div>

        <dl className="mt-6 grid gap-4 sm:grid-cols-2">
          <div>
            <dt className="text-xs text-foreground-muted">Order number</dt>
            <dd className="mt-0.5 font-mono text-sm font-medium">{order.order_number}</dd>
          </div>
          <div>
            <dt className="text-xs text-foreground-muted">Order ID</dt>
            <dd className="mt-0.5 break-all font-mono text-xs">{order.id}</dd>
          </div>
          <div>
            <dt className="text-xs text-foreground-muted">Status</dt>
            <dd className="mt-0.5">
              <span className="inline-flex rounded-full bg-success-soft px-2.5 py-0.5 text-xs font-medium capitalize text-success">
                {order.status}
              </span>
            </dd>
          </div>
          <div>
            <dt className="text-xs text-foreground-muted">Total paid</dt>
            <dd className="mt-0.5 text-sm font-semibold tabular-nums">
              {formatKZT(order.total_kzt)}
            </dd>
          </div>
        </dl>

        <section className="mt-8">
          <h2 className="text-sm font-semibold">Order summary</h2>
          <ul className="mt-2 divide-y divide-border-subtle rounded-lg border border-border-subtle">
            {items.map((item) => (
              <li key={item.id} className="flex items-center justify-between gap-4 px-4 py-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{item.ticket_type_name}</p>
                  <p className="text-xs text-foreground-muted">
                    {item.quantity} × {formatKZT(item.unit_price_kzt)}
                  </p>
                </div>
                <p className="shrink-0 text-sm tabular-nums">
                  {formatKZT(item.line_total_kzt)}
                </p>
              </li>
            ))}
          </ul>
        </section>

        <section className="mt-8">
          <h2 className="text-sm font-semibold">Your tickets</h2>
          <p className="mt-1 text-xs text-foreground-muted">
            Show the QR code at the entrance, or download the A4 PDF to print.
            Each ticket admits one person once.
          </p>
          <ul className="mt-3 space-y-3">
            {tickets.map((ticket) => (
              <TicketCard key={ticket.id} ticket={ticket} />
            ))}
          </ul>
        </section>

        <OrderSupport orderID={order.id} buyerEmail={order.buyer_email} />

        <p className="mt-8 rounded-lg border border-warning/30 bg-warning-soft px-3 py-2 text-xs text-warning">
          This was a simulated payment. No card was charged and no money moved.
        </p>

        <Link
          href="/"
          className="mt-6 inline-flex rounded-lg border border-border-subtle px-4 py-2.5 text-sm font-medium hover:bg-surface-muted"
        >
          Back to BiletFlow
        </Link>
      </div>
    </main>
  );
}
