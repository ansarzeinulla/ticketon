import Link from "next/link";
import type { Metadata } from "next";

import { api } from "@/lib/api";
import { formatInTimezone } from "@/lib/datetime";
import type { BiletEvent } from "@/lib/types";

export const metadata: Metadata = {
  title: "Events",
  description: "Public events on BiletFlow.",
};

/** Events per page. The API caps a single request at 100. */
const PAGE_SIZE = 24;

/**
 * The public event catalogue (SRS 1.2: "Attendees will be able to discover
 * events").
 *
 * Until now /events was a 404 and the only route to an event was a direct slug
 * link, which made discovery impossible for anybody who had not been sent one.
 *
 * It is paginated rather than capped. The API's own limit is 100, so a single
 * unpaginated request silently hides everything past the hundredth soonest
 * event — which looks identical to those events not existing.
 *
 * The API decides what is listed: GET /events forces published and public, so
 * unlisted events stay reachable by link only and private ones stay invisible.
 * Nothing here filters on the client, because a client-side filter over a
 * server that returned too much would be a leak rather than a view.
 */
export const dynamic = "force-dynamic";

export default async function EventCataloguePage({
  searchParams,
}: {
  searchParams: Promise<{ page?: string }>;
}) {
  const { page: rawPage } = await searchParams;
  const page = Math.max(1, Number.parseInt(rawPage ?? "1", 10) || 1);

  let events: BiletEvent[] = [];
  let total = 0;
  let failed = false;

  try {
    const data = await api.listPublicEvents({
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
    });
    events = data.events;
    total = data.total ?? data.events.length;
  } catch {
    failed = true;
  }

  const lastPage = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="mx-auto max-w-4xl space-y-8">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">What&rsquo;s on</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          Public events with tickets available on BiletFlow, soonest first.
        </p>
      </header>

      {failed ? (
        <p className="rounded-xl border border-dashed border-border-subtle px-6 py-12 text-center text-sm text-foreground-muted">
          The events list is unavailable right now. Please try again shortly.
        </p>
      ) : events.length === 0 ? (
        <p
          className="rounded-xl border border-dashed border-border-subtle px-6 py-12 text-center text-sm text-foreground-muted"
          data-testid="catalogue-empty"
        >
          {page > 1
            ? "Nothing on this page."
            : "No public events are on sale at the moment."}
        </p>
      ) : (
        <>
          <ul className="grid gap-4 sm:grid-cols-2" data-testid="catalogue">
            {events.map((event) => (
              <li key={event.id}>
                <Link
                  href={`/events/${event.slug}`}
                  className="flex h-full flex-col overflow-hidden rounded-xl border border-border-subtle bg-surface transition hover:border-brand"
                >
                  {event.cover_image_url && (
                    // Not next/image: the banner is served by the Go API from
                    // local disk, which the optimizer is not configured for.
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={event.cover_image_url}
                      alt=""
                      className="h-36 w-full object-cover"
                    />
                  )}
                  <div className="flex flex-1 flex-col p-4">
                    {event.category && (
                      <span className="text-xs uppercase tracking-wide text-brand">
                        {event.category}
                      </span>
                    )}
                    <h2 className="mt-1 font-semibold">{event.title}</h2>
                    <p className="mt-2 text-sm text-foreground-muted">
                      {formatInTimezone(event.starts_at, event.timezone)}
                    </p>
                    {event.venue_name && (
                      <p className="text-sm text-foreground-muted">{event.venue_name}</p>
                    )}
                  </div>
                </Link>
              </li>
            ))}
          </ul>

          {lastPage > 1 && (
            <nav
              className="flex items-center justify-between gap-4 border-t border-border-subtle pt-6"
              aria-label="Pagination"
              data-testid="catalogue-pagination"
            >
              {page > 1 ? (
                <Link
                  href={`/events?page=${page - 1}`}
                  className="rounded-lg border border-border-subtle px-4 py-2 text-sm font-medium hover:bg-surface-muted"
                  rel="prev"
                >
                  ← Earlier
                </Link>
              ) : (
                <span />
              )}

              <p className="text-sm text-foreground-muted">
                Page {page} of {lastPage} · {total} event{total === 1 ? "" : "s"}
              </p>

              {page < lastPage ? (
                <Link
                  href={`/events?page=${page + 1}`}
                  className="rounded-lg border border-border-subtle px-4 py-2 text-sm font-medium hover:bg-surface-muted"
                  rel="next"
                >
                  Later →
                </Link>
              ) : (
                <span />
              )}
            </nav>
          )}
        </>
      )}
    </div>
  );
}
