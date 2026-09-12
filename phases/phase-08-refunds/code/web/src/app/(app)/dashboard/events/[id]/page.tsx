import { EventDetail } from "./event-detail";

/**
 * The organizer's view of one event. `params` is a promise in Next.js 15+.
 */
export default async function OrganizerEventPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <EventDetail eventID={id} />;
}
