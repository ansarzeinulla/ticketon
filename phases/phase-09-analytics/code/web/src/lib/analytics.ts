"use client";

/**
 * Google Analytics 4 (bonus).
 *
 * Two rules run through everything here:
 *
 *   1. **No PII.** GA4's terms forbid it and so does SRS 7's stance on personal
 *      data. Nothing sent from this file contains a name, an email, an order
 *      number or a ticket code. What goes out is the event slug, a price, and
 *      where the visitor came from.
 *   2. **No campaign tokens.** A `?c=CMP_...` token is a working discount
 *      credential. Attribution therefore reports *that* a visit arrived through
 *      a campaign QR, never which token it was - BiletFlow's own analytics
 *      already attributes revenue per campaign, server-side, where the token
 *      is not being handed to a third party.
 *
 * The whole module is inert when NEXT_PUBLIC_GA4_MEASUREMENT_ID is unset, which
 * is the default: an academic MVP should not be quietly reporting to Google
 * because somebody forgot to configure it.
 */

import { analyticsEnabled } from "@/lib/analytics-config";

export { analyticsEnabled, GA4_MEASUREMENT_ID } from "@/lib/analytics-config";

type GtagArgs =
  | ["js", Date]
  | ["config", string, Record<string, unknown>?]
  | ["event", string, Record<string, unknown>?];

declare global {
  interface Window {
    dataLayer?: unknown[];
    gtag?: (...args: GtagArgs) => void;
  }
}

/**
 * Send one event, or do nothing when analytics is switched off.
 *
 * This pushes onto `dataLayer` rather than calling `window.gtag`, because the
 * tag loads with `afterInteractive` and a page-view fires from an effect on
 * mount - so `gtag` frequently does not exist yet, and calling it would drop
 * the very first event of every visit on the floor. That was not theoretical:
 * it is what the browser showed.
 *
 * Queuing is the documented behaviour of the measurement snippet: gtag.js
 * reads whatever is already in `dataLayer` when it loads. The arguments object
 * is pushed rather than an array so the shape is byte-for-byte what
 * `gtag(...)` itself would have pushed.
 */
function send(name: string, params: Record<string, unknown> = {}): void {
  if (!analyticsEnabled || typeof window === "undefined") return;

  window.dataLayer = window.dataLayer || [];

  // A real `arguments` object, so what lands in the queue is exactly what
  // `gtag("event", ...)` would have pushed. TypeScript needs the indirection
  // spelled out; the runtime shape is the point.
  const push = function (this: void, ...args: unknown[]) {
    // eslint-disable-next-line prefer-rest-params
    window.dataLayer?.push(arguments);
    void args;
  };
  push("event", name, params);
}

/**
 * Where this visit came from.
 *
 * UTM parameters and the referrer's host - not the full referring URL, which
 * can carry a search query or a path that identifies somebody.
 */
export function trafficSource(search: URLSearchParams): Record<string, string> {
  const source: Record<string, string> = {};

  for (const key of ["utm_source", "utm_medium", "utm_campaign", "utm_content"]) {
    const value = search.get(key);
    if (value) source[key] = value.slice(0, 100);
  }

  if (typeof document !== "undefined" && document.referrer) {
    try {
      source.referrer_host = new URL(document.referrer).host;
    } catch {
      // A malformed referrer is not worth reporting.
    }
  }
  return source;
}

/**
 * A public event page was viewed.
 *
 * `via_campaign_qr` is a boolean, deliberately: it answers "did the poster
 * work?" without sending the discount token to Google.
 */
export function trackEventView(input: {
  slug: string;
  category?: string;
  viaCampaignQR: boolean;
  search: URLSearchParams;
}): void {
  send("view_event", {
    event_slug: input.slug,
    event_category: input.category ?? "uncategorised",
    via_campaign_qr: input.viaCampaignQR,
    ...trafficSource(input.search),
  });
}

/** A visit that arrived through a campaign link or QR code. */
export function trackCampaignVisit(input: {
  slug: string;
  search: URLSearchParams;
}): void {
  send("campaign_visit", {
    event_slug: input.slug,
    ...trafficSource(input.search),
  });
}

/**
 * The attendee opened the checkout.
 *
 * GA4's own recommended name, so the funnel report works without custom
 * configuration. `value` is the basket total in KZT - a price, not a person.
 */
export function trackBeginCheckout(input: {
  slug: string;
  valueKZT: number;
  tickets: number;
  viaCampaignQR: boolean;
}): void {
  send("begin_checkout", {
    event_slug: input.slug,
    currency: "KZT",
    value: input.valueKZT,
    quantity: input.tickets,
    via_campaign_qr: input.viaCampaignQR,
  });
}

/**
 * The purchase completed.
 *
 * No transaction_id: an order id is the capability that opens the order page,
 * and it has no business being in a third party's logs.
 */
export function trackPurchase(input: {
  slug: string;
  valueKZT: number;
  tickets: number;
  discounted: boolean;
}): void {
  send("purchase", {
    event_slug: input.slug,
    currency: "KZT",
    value: input.valueKZT,
    quantity: input.tickets,
    discounted: input.discounted,
  });
}

/** Parse a money string ("10350.00") into the number GA4 expects. */
export function toAnalyticsValue(amount: string | undefined): number {
  const parsed = Number.parseFloat(amount ?? "0");
  return Number.isFinite(parsed) ? parsed : 0;
}
