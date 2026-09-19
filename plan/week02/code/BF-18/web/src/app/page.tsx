import { formatDate, formatPrice } from "@/lib/format";
import { mockEvents } from "@/lib/mock-data";

/**
 * The home page lists upcoming events. Until the API is ready it renders
 * placeholder data, so the layout can be built and reviewed early.
 */
export default function HomePage() {
  return (
    <main className="mx-auto max-w-3xl px-4 py-10">
      <h1 className="text-2xl font-semibold">Upcoming events</h1>
      <ul className="mt-6 space-y-3">
        {mockEvents.map((event) => (
          <li key={event.id} className="rounded-lg border border-gray-200 p-4">
            <p className="font-medium">{event.title}</p>
            <p className="text-sm text-gray-500">
              {event.city} · {formatDate(event.startsAt)}
            </p>
            <p className="mt-1 text-sm">{formatPrice(event.priceFromKzt)}</p>
          </li>
        ))}
      </ul>
    </main>
  );
}
