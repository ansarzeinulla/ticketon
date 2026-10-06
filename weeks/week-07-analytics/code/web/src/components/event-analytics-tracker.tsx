"use client";

import { useEffect, useRef } from "react";

import { trackCampaignVisit, trackEventView } from "@/lib/analytics";

/**
 * Reports a public event page view (bonus).
 *
 * A component rather than a call inside the page, because the page is a Server
 * Component and analytics needs the browser: the referrer and the query string
 * only exist there.
 *
 * It fires once per mount. React 18's development StrictMode deliberately runs
 * effects twice, which would double-count every view, so a ref guards it.
 */
export function EventAnalyticsTracker({
  slug,
  category,
  viaCampaignQR,
}: {
  slug: string;
  category?: string;
  viaCampaignQR: boolean;
}) {
  const reported = useRef(false);

  useEffect(() => {
    if (reported.current) return;
    reported.current = true;

    const search = new URLSearchParams(window.location.search);
    trackEventView({ slug, category, viaCampaignQR, search });

    // A campaign visit is reported separately so "did the poster work?" is one
    // metric rather than a filter over every page view.
    if (viaCampaignQR) {
      trackCampaignVisit({ slug, search });
    }
  }, [slug, category, viaCampaignQR]);

  return null;
}
