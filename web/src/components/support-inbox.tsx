"use client";

import { useCallback, useEffect, useState } from "react";

import { SupportThreadView } from "@/components/support-thread";
import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { STATUS_LABELS, STATUS_STYLES, categoryLabel, formatMessageTime } from "@/lib/support";
import type { SupportCase } from "@/lib/types";

/** How often the inbox re-checks for new cases while it is on screen. */
const POLL_INTERVAL_MS = 15_000;

/** The organizer's support inbox for one event. */
export function SupportInbox({ eventID }: { eventID: string }) {
  const [cases, setCases] = useState<SupportCase[]>([]);
  const [openCaseID, setOpenCaseID] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        setCases(await api.eventSupportCases(eventID, signal));
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load support cases.");
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

  // Keep the list current while an organizer has the page open.
  useEffect(() => {
    const timer = setInterval(() => void load(), POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [load]);

  const openCount = cases.filter((item) => item.status !== "resolved").length;

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Support</h2>
          <p className="mt-1 text-sm text-foreground-muted">
            {loading
              ? "Loading…"
              : cases.length === 0
                ? "No attendee has asked anything yet."
                : `${cases.length} case${cases.length === 1 ? "" : "s"} · ${openCount} needing a reply`}
          </p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void load()} disabled={loading}>
          Refresh
        </Button>
      </div>

      {error && <Alert tone="error">{error}</Alert>}

      {loading && cases.length === 0 ? (
        <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface p-6 text-sm text-foreground-muted">
          <Spinner />
          Loading support cases…
        </div>
      ) : cases.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border-subtle bg-surface p-6 text-center text-sm text-foreground-muted">
          Questions attendees ask from their order page arrive here.
        </p>
      ) : (
        <ul className="space-y-2">
          {cases.map((item) => (
            <li key={item.id} className="space-y-2">
              <button
                type="button"
                onClick={() => setOpenCaseID(openCaseID === item.id ? null : item.id)}
                aria-expanded={openCaseID === item.id}
                className="flex w-full flex-wrap items-center justify-between gap-3 rounded-xl border border-border-subtle bg-surface px-5 py-4 text-left hover:bg-surface-muted"
                data-testid="inbox-case"
              >
                <span className="min-w-0">
                  <span className="block truncate font-medium">{item.subject}</span>
                  <span className="text-xs text-foreground-muted">
                    {item.case_number} · {categoryLabel(item.category)} · from{" "}
                    {item.requester_name}
                    {item.order_number ? ` · order ${item.order_number}` : ""}
                    {item.last_message_at
                      ? ` · ${formatMessageTime(item.last_message_at)}`
                      : ""}
                  </span>
                </span>
                <span
                  className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLES[item.status]}`}
                >
                  {STATUS_LABELS[item.status]}
                </span>
              </button>

              {openCaseID === item.id && (
                <SupportThreadView
                  caseID={item.id}
                  onClose={() => setOpenCaseID(null)}
                  onChanged={(thread) =>
                    setCases((current) =>
                      current.map((existing) =>
                        existing.id === thread.case.id ? thread.case : existing,
                      ),
                    )
                  }
                />
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
