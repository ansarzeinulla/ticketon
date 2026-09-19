/**
 * Small display helpers shared by the pages.
 *
 * Deliberately hand-rolled for now: prices are whole tenge and every event is
 * in Almaty, so there is nothing a library would add yet.
 */

/** "Free" or "from 5,000 ₸". */
export function formatPrice(kzt: number): string {
  return kzt === 0 ? "Free" : `from ${kzt.toLocaleString("en-US")} ₸`;
}

/** "10 October, 19:00" in Almaty time, whatever the viewer's timezone. */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Almaty",
  });
}
