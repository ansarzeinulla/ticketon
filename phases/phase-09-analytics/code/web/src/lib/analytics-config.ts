/**
 * GA4 configuration (bonus).
 *
 * Deliberately in its own module with **no** "use client" directive: the root
 * layout is a Server Component and has to read the measurement id to decide
 * whether to render the tag at all. Importing it from a client module made
 * Next.js hand the server a client-reference stub, and the script src came out
 * as a serialized error rather than an id - which is exactly the kind of bug
 * that only shows up when you look at the rendered page.
 */
export const GA4_MEASUREMENT_ID = process.env.NEXT_PUBLIC_GA4_MEASUREMENT_ID ?? "";

/**
 * Analytics is off unless a measurement id is configured, which is the
 * default: an academic MVP should not quietly report to Google because
 * somebody forgot to set an environment variable.
 */
export const analyticsEnabled = GA4_MEASUREMENT_ID !== "";
