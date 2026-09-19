"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import type { Guest } from "@/lib/types";

/** How long to wait after a keystroke before searching. */
const DEBOUNCE_MS = 300;

/**
 * The organizer's guest list (SRS 4.4: "The organizer shall see the
 * registration in the attendee list").
 *
 * The orders table answers "who bought what"; this answers "is this person on
 * the list". A sequence number guards the results: a slow response for "an"
 * must not overwrite a fast one for "anna" typed after it.
 */
export function AttendeeList({ eventID }: { eventID: string }) {
  const [query, setQuery] = useState("");
  const [guests, setGuests] = useState<Guest[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const sequence = useRef(0);

  const search = useCallback(
    async (term: string, signal?: AbortSignal) => {
      const mine = ++sequence.current;
      setLoading(true);
      try {
        const found = await api.eventGuests(eventID, term.trim(), signal);
        if (mine !== sequence.current) return;
        setGuests(found);
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        if (mine !== sequence.current) return;
        setError(cause instanceof ApiError ? cause.message : "Could not load the guest list.");
      } finally {
        if (mine === sequence.current) setLoading(false);
      }
    },
    [eventID],
  );

  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(() => {
      void search(query, controller.signal);
    }, DEBOUNCE_MS);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [query, search]);

  return (
    <section className="rounded-xl border border-border-subtle bg-surface p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-base font-semibold">Attendees</h2>
        <p className="text-xs text-foreground-muted">
          Find somebody by name, email, ticket or order number.
        </p>
      </div>

      <label className="mt-3 block">
        <span className="sr-only">Search attendees</span>
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Name, email, ticket or order number…"
          className="w-full rounded-lg border border-border-subtle bg-surface px-3 py-2 text-sm"
          data-testid="attendee-search"
        />
      </label>

      {error && (
        <div className="mt-3">
          <Alert tone="error">{error}</Alert>
        </div>
      )}

      <div className="mt-3" aria-live="polite">
        {loading && !guests && (
          <p className="flex items-center gap-2 text-sm text-foreground-muted">
            <Spinner /> Loading…
          </p>
        )}

        {guests?.length === 0 && (
          <p className="text-sm text-foreground-muted" data-testid="attendee-empty">
            {query.trim() ? "Nobody on this event’s list matches that." : "No tickets issued yet."}
          </p>
        )}

        {guests && guests.length > 0 && (
          <ul className="divide-y divide-border-subtle" data-testid="attendee-results">
            {guests.map((guest) => (
              <li key={guest.ticket_id} className="py-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{guest.full_name}</span>
                  <span className="rounded-full bg-surface-muted px-2 py-0.5 text-xs font-medium capitalize text-foreground-muted">
                    {guest.status.replace("_", " ")}
                  </span>
                </div>
                <p className="text-xs text-foreground-muted">{guest.email}</p>
                <p className="font-mono text-xs text-foreground-muted">
                  {guest.ticket_type_name} · {guest.ticket_code} · {guest.order_number}
                </p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
