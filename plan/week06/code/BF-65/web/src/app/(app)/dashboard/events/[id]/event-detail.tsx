"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { ActivationChecklist } from "@/components/activation-checklist";
import { AttendeeList } from "@/components/attendee-list";
import { OrderManager } from "@/components/order-manager";
import { StaffManager } from "@/components/staff-manager";
import { SupportInbox } from "@/components/support-inbox";
import { TicketTypeManager } from "@/components/ticket-type-manager";
import { Alert } from "@/components/ui/alert";
import { Spinner } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { ApiError, api } from "@/lib/api";
import { formatInTimezone } from "@/lib/datetime";
import type { BiletEvent } from "@/lib/types";

export function EventDetail({ eventID }: { eventID: string }) {
  // Bumped when something elsewhere on the page changes the figures - a
  // refund, or paid sales opening - so the panels below re-read rather than
  // showing numbers the organizer just invalidated.
  const [refreshKey, setRefreshKey] = useState(0);

  const [event, setEvent] = useState<BiletEvent | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (signal?: AbortSignal) => {
    try {
      setEvent(await api.getEvent(eventID, signal));
      setError(null);
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setError(cause instanceof ApiError ? cause.message : "Could not load this event.");
    } finally {
      setLoading(false);
    }
  }, [eventID]);

  useEffect(() => {
    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  if (loading) {
    return (
      <div className="flex items-center gap-3 text-sm text-foreground-muted">
        <Spinner />
        Loading event…
      </div>
    );
  }

  if (error || !event) {
    return <Alert tone="error">{error ?? "Event not found."}</Alert>;
  }

  return (
    <div className="space-y-8">
      <div>
        <Link href="/dashboard" className="text-sm text-foreground-muted hover:underline">
          ← Back to events
        </Link>

        <div className="mt-2 flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">{event.title}</h1>
          <StatusBadge status={event.status} />
        </div>

        <p className="mt-1 font-mono text-xs text-foreground-muted">/{event.slug}</p>

        <dl className="mt-4 grid grid-cols-2 gap-x-6 gap-y-3 text-sm sm:grid-cols-4">
          <div>
            <dt className="text-xs text-foreground-muted">Starts</dt>
            <dd className="mt-0.5">{formatInTimezone(event.starts_at, event.timezone)}</dd>
          </div>
          <div>
            <dt className="text-xs text-foreground-muted">Ends</dt>
            <dd className="mt-0.5">{formatInTimezone(event.ends_at, event.timezone)}</dd>
          </div>
          <div>
            <dt className="text-xs text-foreground-muted">Timezone</dt>
            <dd className="mt-0.5">{event.timezone}</dd>
          </div>
          <div>
            <dt className="text-xs text-foreground-muted">Capacity</dt>
            <dd className="mt-0.5">{event.capacity ?? "Unlimited"}</dd>
          </div>
        </dl>

        {event.status === "suspended" ? (
          <Alert tone="error" title="Suspended by BiletFlow">
            A platform administrator has suspended this event pending review. Ticket
            sales are blocked. Existing ticket holders can still be checked in.
          </Alert>
        ) : event.status === "published" ? (
          <p className="mt-4 text-sm">
            <Link
              href={`/events/${event.slug}`}
              className="font-medium text-brand hover:underline"
            >
              View the public page →
            </Link>
          </p>
        ) : (
          <Alert tone="info">
            This event is a {event.status}. Publish it from the dashboard before
            attendees can find and buy tickets.
          </Alert>
        )}
      </div>

      {/*
        Activation sits above the numbers: an organizer whose paid tickets are
        not on sale needs to know that before they wonder why nothing has sold.
        It renders nothing at all for a free event.
      */}
      <ActivationChecklist eventID={event.id} onActivated={() => setRefreshKey((n) => n + 1)} />

      <OrderManager
        eventID={event.id}
        onRefunded={() => setRefreshKey((n) => n + 1)}
      />

      {/*
        Keyed on the refresh counter: a refund returns inventory, so leaving
        this panel showing the pre-refund "sold" would contradict the order
        list directly above it.
      */}
      <TicketTypeManager key={`types-${refreshKey}`} eventID={event.id} />

      <AttendeeList key={`attendees-${refreshKey}`} eventID={event.id} />

      <StaffManager eventID={event.id} />

      <SupportInbox eventID={event.id} />

    </div>
  );
}
