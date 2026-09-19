import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ApiError, api } from "@/lib/api";
import { formatInTimezone } from "@/lib/datetime";
import { formatKZT } from "@/lib/money";

/**
 * The attendee-facing event page.
 *
 * A Server Component: the public endpoint needs no token, so the event and its
 * remaining stock are fetched on the server. That gives a real HTML page for
 * sharing and search, with no loading flash.
 */
export default async function PublicEventPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;

  const data = await loadEvent(slug);
  if (!data) notFound();

  const { event, ticket_types: ticketTypes, on_sale: onSale, sold_out: soldOut } = data;

  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-10 sm:px-6">
      <article className="space-y-8">
        {event.cover_image_url && (
          <figure>
            {/*
              A plain <img>, not next/image: the banner lives on the API's
              upload route, which is not a configured next/image domain, and
              the file is already a modest photograph.
            */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={event.cover_image_url}
              alt={`Banner for ${event.title}`}
              className="max-h-80 w-full rounded-xl border border-border-subtle object-cover"
            />
          </figure>
        )}

        <header className="space-y-3">
          {event.category && (
            <p className="text-xs font-medium uppercase tracking-wider text-brand">
              {event.category}
            </p>
          )}
          <h1 className="text-3xl font-semibold tracking-tight">{event.title}</h1>

          <dl className="flex flex-wrap gap-x-8 gap-y-3 text-sm">
            <div>
              <dt className="text-xs text-foreground-muted">When</dt>
              <dd className="mt-0.5 font-medium">
                {formatInTimezone(event.starts_at, event.timezone)}
              </dd>
              <dd className="text-xs text-foreground-muted">
                until {formatInTimezone(event.ends_at, event.timezone)} ({event.timezone})
              </dd>
            </div>
            {event.venue_name && (
              <div>
                <dt className="text-xs text-foreground-muted">Where</dt>
                <dd className="mt-0.5 font-medium">{event.venue_name}</dd>
                {event.venue_address && (
                  <dd className="text-xs text-foreground-muted">{event.venue_address}</dd>
                )}
              </div>
            )}
          </dl>
        </header>

        {event.description && (
          <p className="whitespace-pre-line text-sm leading-relaxed text-foreground-muted">
            {event.description}
          </p>
        )}

        <section className="space-y-3">
          <h2 className="text-lg font-semibold">Tickets</h2>

          {ticketTypes.length === 0 ? (
            <p className="text-sm text-foreground-muted">
              The organizer has not published any ticket types for this event yet.
            </p>
          ) : (
            <ul className="divide-y divide-border-subtle rounded-xl border border-border-subtle bg-surface">
              {ticketTypes.map((type) => (
                <li key={type.id} className="flex items-center justify-between gap-4 p-4">
                  <div>
                    <p className="font-medium">{type.name}</p>
                    {type.description && (
                      <p className="text-sm text-foreground-muted">{type.description}</p>
                    )}
                  </div>
                  <div className="text-right">
                    <p className="font-medium">{type.is_free ? "Free" : formatKZT(type.price_kzt)}</p>
                    <p className="text-xs text-foreground-muted">
                      {type.quantity_remaining > 0
                        ? `${type.quantity_remaining} of ${type.quantity_total} left`
                        : "Sold out"}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          )}

          {soldOut ? (
            <p className="text-sm font-medium text-danger">Every ticket for this event has been taken.</p>
          ) : !onSale && ticketTypes.length > 0 ? (
            <p className="text-sm text-foreground-muted">
              Tickets for this event are not currently available.
            </p>
          ) : null}
        </section>

        {event.refund_policy && (
          <section className="rounded-xl border border-border-subtle bg-surface p-5">
            <h2 className="text-sm font-semibold">Refund policy</h2>
            <p className="mt-1 whitespace-pre-line text-sm text-foreground-muted">
              {event.refund_policy}
            </p>
          </section>
        )}
      </article>
    </main>
  );
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const data = await loadEvent(slug);
  if (!data) return { title: "Event not found" };

  return {
    title: data.event.title,
    description: data.event.description ?? `Tickets for ${data.event.title} on BiletFlow.`,
  };
}

/** Returns null for a 404 so the caller can render notFound(). */
async function loadEvent(slug: string) {
  try {
    return await api.getPublicEvent(slug);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}
