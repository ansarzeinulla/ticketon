import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { EventAnalyticsTracker } from "@/components/event-analytics-tracker";
import { SeatedPurchase } from "@/components/seated-purchase";
import { ApiError, api } from "@/lib/api";
import { formatInTimezone } from "@/lib/datetime";
import { getTranslations } from "@/lib/i18n/server";

import { TicketSelector } from "./ticket-selector";

/**
 * The attendee-facing event page.
 *
 * A Server Component: the public endpoint needs no token, so the event and its
 * remaining stock are fetched on the server. That gives a real HTML page for
 * sharing and search, with no loading flash. Only the ticket selector and its
 * checkout are interactive, and they are a Client Component below.
 */
export default async function PublicEventPage({
  params,
  searchParams,
}: {
  params: Promise<{ slug: string }>;
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const [{ slug }, query] = await Promise.all([params, searchParams]);
  const { t } = await getTranslations();

  const data = await loadEvent(slug);
  if (!data) notFound();

  // A scanned campaign QR lands here as ?c=CMP_<uuid>. Only the opaque token
  // travels in the link; what it is worth is decided by the server.
  const campaignToken = typeof query.c === "string" && query.c.startsWith("CMP_")
    ? query.c
    : undefined;

  const {
    event,
    ticket_types: ticketTypes,
    on_sale: onSale,
    sold_out: soldOut,
    suspended,
    paid_sales_active: paidSalesActive,
    paid_sales_required: paidSalesRequired,
  } = data;

  // Paid tickets exist but the organizer has not finished activation, so the
  // checkout would refuse them (SRS 4.5). Saying so beats letting somebody
  // pick tickets and fill in a form that cannot succeed.
  const paidSalesPending = paidSalesRequired && !paidSalesActive && !suspended;
  const hasFreeTier = ticketTypes.some((type) => type.is_free);

  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-10 sm:px-6">
      {/*
        Reports the view, and separately whether it arrived through a campaign
        QR (bonus). No token is sent - only the fact that one was present.
      */}
      <EventAnalyticsTracker
        slug={event.slug}
        category={event.category}
        viaCampaignQR={Boolean(campaignToken)}
      />

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
              <dt className="text-xs text-foreground-muted">{t("eventPage.when")}</dt>
              <dd className="mt-0.5 font-medium">
                {formatInTimezone(event.starts_at, event.timezone)}
              </dd>
              <dd className="text-xs text-foreground-muted">
                {t("eventPage.until", {
                  time: formatInTimezone(event.ends_at, event.timezone),
                  tz: event.timezone,
                })}
              </dd>
            </div>
            {event.venue_name && (
              <div>
                <dt className="text-xs text-foreground-muted">{t("eventPage.where")}</dt>
                <dd className="mt-0.5 font-medium">{event.venue_name}</dd>
                {event.venue_address && (
                  <dd className="text-xs text-foreground-muted">{event.venue_address}</dd>
                )}
              </div>
            )}
          </dl>
        </header>

        {suspended && (
          <div
            role="alert"
            className="rounded-xl border border-danger/40 bg-danger-soft p-5"
            data-testid="suspended-banner"
          >
            <h2 className="text-base font-semibold text-danger">
              {t("eventPage.suspendedTitle")}
            </h2>
            <p className="mt-1 text-sm text-danger/90">
              {t("eventPage.suspendedBody")}
            </p>
          </div>
        )}

        {paidSalesPending && (
          <div
            role="status"
            className="rounded-xl border border-warning/40 bg-warning-soft p-5"
            data-testid="paid-sales-pending-banner"
          >
            <h2 className="text-base font-semibold text-warning">
              {t("eventPage.paidPendingTitle")}
            </h2>
            <p className="mt-1 text-sm text-warning/90">
              {t("eventPage.paidPendingBody")}
              {hasFreeTier
                ? t("eventPage.paidPendingFree")
                : t("eventPage.paidPendingCheckBack")}
            </p>
          </div>
        )}

        {event.description && (
          <p className="whitespace-pre-line text-sm leading-relaxed text-foreground-muted">
            {event.description}
          </p>
        )}

        {/*
          With activation outstanding and no free tier there is nothing to
          offer, and the selector's empty state would claim the organizer had
          published no ticket types - which is untrue and would send them
          looking for a problem that does not exist. The banner above has
          already explained the real reason.
        */}
        {/*
          An assigned-seating event is bought by picking a seat, not by
          stepping a quantity: the attendee is choosing *where* to sit, and a
          "+/-" cannot express that (SRS 4.3.1).
        */}
        {!suspended && !(paidSalesPending && !hasFreeTier) &&
          event.seating_mode === "assigned_seating" && (
            <SeatedPurchase eventID={event.id} eventSlug={event.slug} />
          )}

        {!suspended && !(paidSalesPending && !hasFreeTier) &&
          event.seating_mode !== "assigned_seating" && (
          <TicketSelector
            eventID={event.id}
            eventSlug={event.slug}
            eventTitle={event.title}
            /*
              While activation is outstanding the paid tiers are not offered.
              The checkout would refuse them anyway, and a selector that lets
              somebody build a basket it will reject is worse than one that
              shows only what can actually be bought. The banner above explains
              where the rest went.
            */
            ticketTypes={
              paidSalesPending ? ticketTypes.filter((type) => type.is_free) : ticketTypes
            }
            onSale={onSale}
            soldOut={soldOut}
            campaignToken={campaignToken}
          />
        )}

        {event.refund_policy && (
          <section className="rounded-xl border border-border-subtle bg-surface p-5">
            <h2 className="text-sm font-semibold">{t("eventPage.refundPolicy")}</h2>
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
