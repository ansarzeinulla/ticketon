"use client";

import { useCallback, useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import type { TimelineEntry } from "@/lib/types";

/** Activity groups the filter offers, keyed by the action prefix. */
const ACTIVITY_FILTERS = [
  { label: "Everything", prefix: "" },
  { label: "Event", prefix: "event." },
  { label: "Tickets", prefix: "ticket." },
  { label: "Orders", prefix: "order." },
  { label: "Campaigns", prefix: "campaign." },
  { label: "Support", prefix: "support_case." },
] as const;

/** Human wording for the actions the system records. */
const ACTION_LABELS: Record<string, string> = {
  "event.created": "Event created",
  "event.updated": "Event details changed",
  "event.published": "Published",
  "event.unpublished": "Unpublished",
  "event.cancelled": "Cancelled",
  "event.suspended": "Suspended by BiletFlow",
  "event.unsuspended": "Suspension lifted",
  "event.deleted": "Deleted",
  "event.duplicated": "Created by duplication",
  "event.duplicated_from": "Duplicated into a new draft",
  "ticket_type.created": "Ticket type added",
  "ticket_type.updated": "Ticket type changed",
  "ticket_type.deleted": "Ticket type removed",
  "order.created": "Ticket sold",
  "ticket.checked_in": "Attendee checked in",
  "ticket.check_in_reversed": "Check-in reversed",
  "campaign.created": "Promo code created",
  "campaign.active": "Promo code enabled",
  "campaign.disabled": "Promo code disabled",
  "campaign.deleted": "Promo code deleted",
  "staff.assigned": "Event Admin assigned",
  "staff.revoked": "Event Admin removed",
  "support_case.opened": "Support case opened",
  "support_case.resolved": "Support case resolved",
  "support_case.in_progress": "Support case in progress",
  "support_case.waiting_for_customer": "Support case waiting on the attendee",
  "support_case.open": "Support case reopened",
};

function actionLabel(action: string): string {
  return ACTION_LABELS[action] ?? action.replace(/[._]/g, " ");
}

function formatWhen(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

/** The event's chronological activity history (SRS 4.16). */
export function EventTimeline({ eventID }: { eventID: string }) {
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [prefix, setPrefix] = useState("");
  const [from, setFrom] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        setEntries(
          await api.eventTimeline(
            eventID,
            { type: prefix || undefined, from: from || undefined, limit: 200 },
            signal,
          ),
        );
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load the history.");
      } finally {
        setLoading(false);
      }
    },
    [eventID, prefix, from],
  );

  useEffect(() => {
    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  return (
    <section className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">Activity history</h2>
        <p className="mt-1 text-sm text-foreground-muted">
          Everything that has happened to this event, newest first. Entries are
          append-only — they cannot be edited or removed.
        </p>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="flex flex-wrap gap-1">
          {ACTIVITY_FILTERS.map((filter) => (
            <Button
              key={filter.label}
              size="sm"
              variant={prefix === filter.prefix ? "primary" : "secondary"}
              onClick={() => setPrefix(filter.prefix)}
            >
              {filter.label}
            </Button>
          ))}
        </div>
        <div className="space-y-1">
          <label htmlFor="timeline-from" className="block text-xs text-foreground-muted">
            Since
          </label>
          <input
            id="timeline-from"
            type="date"
            value={from}
            onChange={(event) => setFrom(event.target.value)}
            className="rounded-lg border border-border-subtle bg-surface px-3 py-2 text-sm"
          />
        </div>
      </div>

      {error && <Alert tone="error">{error}</Alert>}

      {loading && entries.length === 0 ? (
        <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface p-6 text-sm text-foreground-muted">
          <Spinner />
          Loading the history…
        </div>
      ) : entries.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border-subtle bg-surface p-6 text-center text-sm text-foreground-muted">
          Nothing recorded for this filter.
        </p>
      ) : (
        <ol className="space-y-1 rounded-xl border border-border-subtle bg-surface p-2" data-testid="timeline">
          {entries.map((entry) => (
            <li
              key={entry.id}
              className="flex flex-wrap items-baseline gap-x-3 gap-y-1 rounded-lg px-3 py-2 hover:bg-surface-muted/60"
            >
              <span className="w-40 shrink-0 text-xs text-foreground-muted tabular-nums">
                {formatWhen(entry.created_at)}
              </span>
              <span className="font-medium">{actionLabel(entry.action)}</span>
              {entry.description && (
                <span className="text-sm text-foreground-muted">{entry.description}</span>
              )}
              {entry.actor_name && (
                <span className="ml-auto text-xs text-foreground-muted">
                  by {entry.actor_name}
                </span>
              )}
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
