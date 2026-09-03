import Script from "next/script";

import { GA4_MEASUREMENT_ID, analyticsEnabled } from "@/lib/analytics-config";

/**
 * Loads GA4, and only when it is configured (bonus).
 *
 * `afterInteractive` rather than `beforeInteractive`: an analytics tag has no
 * business delaying the page an attendee came to read, and measurement that
 * starts a few hundred milliseconds late is still measurement.
 *
 * `anonymize_ip` and the two ads signals are switched off explicitly. GA4
 * defaults some of them on, and an academic ticketing MVP has no business
 * building advertising profiles of the people buying tickets.
 */
export function AnalyticsScripts() {
  if (!analyticsEnabled) return null;

  return (
    <>
      <Script
        src={`https://www.googletagmanager.com/gtag/js?id=${GA4_MEASUREMENT_ID}`}
        strategy="afterInteractive"
      />
      <Script id="ga4-init" strategy="afterInteractive">
        {`
          window.dataLayer = window.dataLayer || [];
          function gtag(){dataLayer.push(arguments);}
          window.gtag = gtag;
          gtag('js', new Date());
          gtag('config', '${GA4_MEASUREMENT_ID}', {
            anonymize_ip: true,
            allow_google_signals: false,
            allow_ad_personalization_signals: false
          });
        `}
      </Script>
    </>
  );
}
