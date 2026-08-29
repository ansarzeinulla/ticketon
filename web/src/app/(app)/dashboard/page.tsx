"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { EventCard } from "@/components/event-card";
import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import type { BiletEvent, Lifecycle } from "@/lib/types";

/** The groupings SRS 4.16 asks the organizer's history to offer. */
const LIFECYCLE_TABS: { label: string; value: Lifecycle | "" }[] = [
  { label: "All", value: "" },
  { label: "Upcoming", value: "upcoming" },
  { label: "Active", value: "active" },
  { label: "Completed", value: "completed" },
  { label: "Cancelled", value: "cancelled" },
  { label: "Drafts", value: "draft" },
];

export default function DashboardPage() {
  const [events, setEvents] = useState<BiletEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lifecycle, setLifecycle] = useState<Lifecycle | "">("");

  /**
   * Fetches the organizer's own events. Every state update happens after the
   * await, so calling this from an effect cannot cascade renders.
   */
  const load = useCallback(async (signal?: AbortSignal) => {
    try {
      const data = await api.listMyEvents(
        { limit: 100, ...(lifecycle ? { lifecycle } : {}) },
        signal,
      );
      setEvents(data.events);
      setTotal(data.total);
      setError(null);
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setError(cause instanceof ApiError ? cause.message : "Could not load your events.");
    } finally {
      setLoading(false);
    }
  }, [lifecycle]);

  useEffect(() => {
    const controller = new AbortController();
    // The rule cannot see through the call into `load`, where every setState
    // now happens after the await. Nothing here updates state synchronously.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  /** The Refresh button owns its own spinner state, unlike the initial load. */
  const handleRefresh = useCallback(() => {
    setLoading(true);
    setError(null);
    void load();
  }, [load]);

  /** Swap one event in place after a publish or cancel. */
  const handleChanged = useCallback((updated: BiletEvent) => {
    setEvents((current) =>
      current.map((event) => (event.id === updated.id ? updated : event)),
    );
  }, []);

  const counts = events.reduce<Record<string, number>>((acc, event) => {
    acc[event.status] = (acc[event.status] ?? 0) + 1;
    return acc;
  }, {});

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Your events</h1>
          <p className="mt-1 text-sm text-foreground-muted">
            {loading
              ? "Loading…"
              : `${total} event${total === 1 ? "" : "s"}` +
                (counts.draft ? ` · ${counts.draft} draft` : "") +
                (counts.published ? ` · ${counts.published} published` : "")}
          </p>
        </div>

        <div className="flex gap-2">
          <Button variant="secondary" onClick={handleRefresh} disabled={loading}>
            Refresh
          </Button>
          <Link
            href="/events/new"
            className="inline-flex items-center justify-center rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-strong"
          >
            New event
          </Link>
        </div>
      </div>

      <div className="flex flex-wrap gap-1" role="group" aria-label="Filter events">
        {LIFECYCLE_TABS.map((tab) => (
          <Button
            key={tab.label}
            size="sm"
            variant={lifecycle === tab.value ? "primary" : "secondary"}
            onClick={() => {
              setLoading(true);
              setLifecycle(tab.value);
            }}
            data-testid={`filter-${tab.value || "all"}`}
          >
            {tab.label}
          </Button>
        ))}
      </div>

      {error && (
        <Alert tone="error" title="Could not load your events">
          {error}
        </Alert>
      )}

      {loading && events.length === 0 ? (
        <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface p-8 text-sm text-foreground-muted">
          <Spinner />
          Fetching your events from the API…
        </div>
      ) : events.length === 0 && !error && lifecycle ? (
        <p className="rounded-xl border border-dashed border-border-subtle bg-surface p-10 text-center text-sm text-foreground-muted">
          No {LIFECYCLE_TABS.find((tab) => tab.value === lifecycle)?.label.toLowerCase()} events.
        </p>
      ) : events.length === 0 && !error ? (
        <div className="rounded-xl border border-dashed border-border-subtle bg-surface p-12 text-center">
          <h2 className="text-base font-semibold">No events yet</h2>
          <p className="mx-auto mt-1 max-w-sm text-sm text-foreground-muted">
            Create your first event to start issuing tickets. Free events cost nothing to
            publish.
          </p>
          <Link
            href="/events/new"
            className="mt-5 inline-flex items-center justify-center rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-strong"
          >
            Create your first event
          </Link>
        </div>
      ) : (
        <ul className="space-y-3">
          {events.map((event) => (
            <EventCard key={event.id} event={event} onChanged={handleChanged} />
          ))}
        </ul>
      )}
    </div>
  );
}
