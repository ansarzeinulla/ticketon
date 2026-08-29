import { EditEventForm } from "./edit-event-form";

/**
 * Editing an event (SRS 4.2). `params` is a promise in Next.js 15+.
 */
export default async function EditEventPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <EditEventForm eventID={id} />;
}
