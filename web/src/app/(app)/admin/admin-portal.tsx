"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { ModerationAction } from "@/components/moderation-action";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatKZT } from "@/lib/money";
import type { AdminSearchResponse } from "@/lib/types";

/** How long to wait after a keystroke before searching. */
const DEBOUNCE_MS = 300;

function formatDate(iso?: string): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/**
 * The administrative portal (SRS 2.1, 4.12).
 *
 * What is rendered here is a convenience: every one of these endpoints is
 * enforced server-side by the platform_admin role, so an account that reaches
 * this page without the role sees nothing but errors from the API. The check
 * below exists to say so plainly rather than to be the security boundary.
 */
export function AdminPortal() {
  const { user } = useAuth();
  const isAdmin = user?.roles.includes("platform_admin") ?? false;

  const [query, setQuery] = useState("");
  const [data, setData] = useState<AdminSearchResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [downloading, setDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState<string | null>(null);

  const search = useCallback(
    async (term: string, signal?: AbortSignal) => {
      try {
        setData(await api.adminSearch(term, signal));
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not search.");
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  // Debounced: an admin typing an email should not fire eight queries.
  //
  // Nothing is fetched for a non-admin. `loading` is never read on that path -
  // the refusal below returns before it matters - so the effect simply does
  // not run rather than setting state to say so.
  useEffect(() => {
    if (!isAdmin) return;

    const controller = new AbortController();
    const timer = setTimeout(() => {
      void search(query.trim(), controller.signal);
    }, query === "" ? 0 : DEBOUNCE_MS);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [query, search, isAdmin]);

  /**
   * Re-run the current search after a moderation action.
   *
   * A suspension changes the row that was just acted on - and the stat tiles
   * above it - so the table has to be re-read rather than patched in place,
   * which would leave the counts stale.
   */
  const rerun = useCallback(() => {
    void search(query.trim());
  }, [query, search]);

  /**
   * Download the CSV report (SRS 4.12).
   *
   * A plain <a href> cannot carry the bearer token, so the file is fetched with
   * the header attached and handed to the browser as a blob. The object URL is
   * revoked afterwards rather than leaking a reference to the whole file.
   */
  async function downloadReport() {
    setDownloading(true);
    setDownloadError(null);
    try {
      const blob = await api.adminReportBlob();
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `biletflow-events-${new Date().toISOString().slice(0, 10)}.csv`;
      document.body.append(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
    } catch {
      setDownloadError("Could not build the report. Try again.");
    } finally {
      setDownloading(false);
    }
  }

  if (!isAdmin) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-semibold tracking-tight">Administration</h1>
        <Alert tone="error" title="Not available to this account">
          The administrative portal is restricted to BiletFlow platform
          administrators. If you organize events, your own dashboard is{" "}
          <Link href="/dashboard" className="font-medium underline">
            here
          </Link>
          .
        </Alert>
      </div>
    );
  }

  const results = data?.results;
  const stats = data?.stats;

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Administration</h1>
          <p className="mt-1 text-sm text-foreground-muted">
            Search users, events, orders and payments across the platform.
          </p>
        </div>
        <div className="flex flex-col items-end gap-2">
          <div className="flex gap-2">
            <Link
              href="/admin/reports"
              className="inline-flex items-center rounded-lg border border-border-subtle px-3 py-2 text-sm font-medium hover:bg-surface-muted"
              data-testid="reports-link"
            >
              Reported events
            </Link>
            <Button onClick={() => void downloadReport()} loading={downloading}>
              Export CSV report
            </Button>
          </div>
          {downloadError && <span className="text-xs text-danger">{downloadError}</span>}
        </div>
      </header>

      {stats && (
        <section className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          {[
            { label: "Users", value: String(stats.users) },
            { label: "Events", value: String(stats.events) },
            { label: "Published", value: String(stats.published_events) },
            { label: "Tickets sold", value: String(stats.tickets_sold) },
            { label: "Gross", value: formatKZT(stats.gross_revenue_kzt) },
            { label: "Refunded", value: formatKZT(stats.refunded_kzt) },
          ].map((tile) => (
            <div
              key={tile.label}
              className="rounded-lg border border-border-subtle bg-surface p-3"
            >
              <p className="text-xs text-foreground-muted">{tile.label}</p>
              <p className="mt-1 text-lg font-semibold tabular-nums">{tile.value}</p>
            </div>
          ))}
        </section>
      )}

      {stats && (stats.suspended_events > 0 || stats.failed_payments > 0 ||
        stats.open_support_cases > 0) && (
        <Alert tone="info" title="Needs attention">
          {[
            stats.suspended_events > 0 && `${stats.suspended_events} suspended event(s)`,
            stats.failed_payments > 0 && `${stats.failed_payments} failed payment(s)`,
            stats.open_support_cases > 0 && `${stats.open_support_cases} open support case(s)`,
          ]
            .filter(Boolean)
            .join(" · ")}
        </Alert>
      )}

      <div>
        <label htmlFor="admin-search" className="block text-sm font-medium">
          Search
        </label>
        <input
          id="admin-search"
          type="search"
          value={query}
          placeholder="Email, name, event title, order number…"
          onChange={(event) => setQuery(event.target.value)}
          className="mt-1.5 w-full rounded-lg border border-border-subtle bg-surface px-3 py-2 text-sm"
        />
        <p className="mt-1 text-xs text-foreground-muted">
          {loading ? "Searching…" : "One query, every kind of record."}
        </p>
      </div>

      {error && <Alert>{error}</Alert>}

      {results && (
        <div className="space-y-8">
          <Section title="Users" count={results.users.length}>
            <Table columns={["Account", "Status", "Roles", "Events", "Orders", "Joined", "Action"]}>
              {results.users.map((row) => (
                <tr key={row.id} className="border-t border-border-subtle">
                  <td className="px-4 py-2">
                    <div className="font-medium">{row.full_name}</div>
                    <div className="text-xs text-foreground-muted">{row.email}</div>
                  </td>
                  <td className="px-4 py-2">
                    <Pill value={row.status} />
                    {!row.email_verified && (
                      <span className="ml-2 text-xs text-warning">unverified</span>
                    )}
                  </td>
                  <td className="px-4 py-2 text-xs">
                    {row.roles.length > 0 ? row.roles.join(", ") : "—"}
                  </td>
                  <td className="px-4 py-2 tabular-nums">{row.event_count}</td>
                  <td className="px-4 py-2 tabular-nums">{row.order_count}</td>
                  <td className="px-4 py-2 text-xs">{formatDate(row.created_at)}</td>
                  <td className="px-4 py-2">
                    {row.id === user?.id ? (
                      <span className="text-xs text-foreground-muted">You</span>
                    ) : row.status === "suspended" ? (
                      <ModerationAction
                        label="Restore"
                        confirmLabel="Restore this account"
                        consequence="They will be able to sign in again straight away. An account that never confirmed its address goes back to unverified rather than active."
                        tone="secondary"
                        withReason={false}
                        onConfirm={() => api.unsuspendUser(row.id)}
                        onDone={rerun}
                      />
                    ) : (
                      <ModerationAction
                        label="Suspend"
                        confirmLabel="Suspend this account"
                        consequence="They are signed out on their very next request, and their events stop selling tickets. Tickets already sold stay valid."
                        onConfirm={(reason) => api.suspendUser(row.id, reason)}
                        onDone={rerun}
                      />
                    )}
                  </td>
                </tr>
              ))}
            </Table>
          </Section>

          <Section title="Events" count={results.events.length}>
            <Table columns={["Event", "Organizer", "Stage", "Activation", "Sold", "Revenue", "Action"]}>
              {results.events.map((row) => (
                <tr key={row.id} className="border-t border-border-subtle">
                  <td className="px-4 py-2">
                    <Link
                      href={`/events/${row.slug}`}
                      className="font-medium text-brand hover:underline"
                    >
                      {row.title}
                    </Link>
                    <div className="text-xs text-foreground-muted">{formatDate(row.starts_at)}</div>
                  </td>
                  <td className="px-4 py-2 text-xs">{row.organizer_email}</td>
                  <td className="px-4 py-2">
                    <Pill value={row.lifecycle} />
                  </td>
                  <td className="px-4 py-2 text-xs">{row.activation_status}</td>
                  <td className="px-4 py-2 tabular-nums">{row.tickets_sold}</td>
                  <td className="px-4 py-2 tabular-nums">{formatKZT(row.revenue_kzt)}</td>
                  <td className="px-4 py-2">
                    <div className="space-y-2">
                      {row.lifecycle === "suspended" || row.status === "suspended" ? (
                        <ModerationAction
                          label="Lift suspension"
                          confirmLabel="Lift the suspension"
                          consequence="The event returns to unpublished, so the organizer has to publish it again deliberately before it can sell."
                          tone="secondary"
                          withReason={false}
                          onConfirm={() => api.unsuspendEvent(row.id)}
                          onDone={rerun}
                        />
                      ) : (
                        <ModerationAction
                          label="Suspend"
                          confirmLabel="Suspend this event"
                          consequence="Checkout is refused immediately and the public page shows a notice. Tickets already sold stay valid, so paying attendees are not stranded."
                          onConfirm={(reason) => api.suspendEvent(row.id, reason)}
                          onDone={rerun}
                        />
                      )}

                      {row.activation_status === "suspended" ? (
                        <ModerationAction
                          label="Restore paid sales"
                          confirmLabel="Restore paid sales"
                          consequence="The event can take money again. Free registration was never affected."
                          tone="secondary"
                          withReason={false}
                          onConfirm={() => api.restorePaidSales(row.id)}
                          onDone={rerun}
                        />
                      ) : row.activation_status === "active" ? (
                        <ModerationAction
                          label="Stop paid sales"
                          confirmLabel="Stop paid sales"
                          consequence="Paid tickets stop selling; free registration carries on. Use this when the money is the problem and the event is not."
                          onConfirm={(reason) => api.suspendPaidSales(row.id, reason)}
                          onDone={rerun}
                        />
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
            </Table>
          </Section>

          <Section title="Orders" count={results.orders.length}>
            <Table columns={["Order", "Buyer", "Event", "Status", "Total", "Refunded"]}>
              {results.orders.map((row) => (
                <tr key={row.id} className="border-t border-border-subtle">
                  <td className="px-4 py-2 font-mono text-xs">{row.order_number}</td>
                  <td className="px-4 py-2">
                    <div>{row.buyer_name}</div>
                    <div className="text-xs text-foreground-muted">{row.buyer_email}</div>
                  </td>
                  <td className="px-4 py-2 text-xs">{row.event_title}</td>
                  <td className="px-4 py-2">
                    <Pill value={row.status} />
                  </td>
                  <td className="px-4 py-2 tabular-nums">{formatKZT(row.total_kzt)}</td>
                  <td className="px-4 py-2 tabular-nums">
                    {row.refunded_kzt === "0.00" ? "—" : formatKZT(row.refunded_kzt)}
                  </td>
                </tr>
              ))}
            </Table>
          </Section>

          <Section title="Payments" count={results.payments.length}>
            <Table columns={["Purpose", "Reference", "Status", "Amount", "When"]}>
              {results.payments.map((row) => (
                <tr key={row.id} className="border-t border-border-subtle">
                  <td className="px-4 py-2 text-xs">{row.purpose}</td>
                  <td className="px-4 py-2 text-xs">
                    {row.order_number ?? row.event_title ?? "—"}
                  </td>
                  <td className="px-4 py-2">
                    <Pill value={row.status} />
                    {row.is_simulated && (
                      <span className="ml-2 text-xs text-foreground-muted">simulated</span>
                    )}
                  </td>
                  <td className="px-4 py-2 tabular-nums">{formatKZT(row.amount_kzt)}</td>
                  <td className="px-4 py-2 text-xs">{formatDate(row.created_at)}</td>
                </tr>
              ))}
            </Table>
          </Section>
        </div>
      )}
    </div>
  );
}

/**
 * A neutral status pill.
 *
 * The portal shows four different kinds of status - account, lifecycle, order,
 * payment - and StatusBadge is deliberately typed to event statuses alone.
 * Widening it to `string` to serve this page would throw away the type safety
 * it gives everywhere else, so the portal carries its own.
 */
const PILL_TONES: Record<string, string> = {
  active: "bg-success-soft text-success",
  published: "bg-success-soft text-success",
  paid: "bg-success-soft text-success",
  completed: "bg-success-soft text-success",
  succeeded: "bg-success-soft text-success",
  upcoming: "bg-brand/10 text-brand",
  in_progress: "bg-brand/10 text-brand",
  pending_verification: "bg-warning-soft text-warning",
  pending: "bg-warning-soft text-warning",
  partially_refunded: "bg-warning-soft text-warning",
  suspended: "bg-danger-soft text-danger",
  cancelled: "bg-danger-soft text-danger",
  refunded: "bg-danger-soft text-danger",
  failed: "bg-danger-soft text-danger",
};

function Pill({ value }: { value: string }) {
  const tone = PILL_TONES[value] ?? "bg-surface-muted text-foreground-muted";
  return (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${tone}`}>
      {value.replace(/_/g, " ")}
    </span>
  );
}

function Section({
  title,
  count,
  children,
}: {
  title: string;
  count: number;
  children: React.ReactNode;
}) {
  return (
    <section>
      <h2 className="text-sm font-semibold uppercase tracking-wide text-foreground-muted">
        {title} <span className="ml-1 font-normal">({count})</span>
      </h2>
      <div className="mt-2">
        {count === 0 ? (
          <p className="rounded-lg border border-dashed border-border-subtle px-4 py-6 text-center text-sm text-foreground-muted">
            Nothing matched.
          </p>
        ) : (
          children
        )}
      </div>
    </section>
  );
}

function Table({
  columns,
  children,
}: {
  columns: string[];
  children: React.ReactNode;
}) {
  return (
    <div className="overflow-x-auto rounded-lg border border-border-subtle">
      <table className="w-full min-w-[44rem] text-sm">
        <thead className="bg-surface-muted text-left text-xs uppercase tracking-wide text-foreground-muted">
          <tr>
            {columns.map((column) => (
              <th key={column} className="px-4 py-2 font-medium">
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  );
}
