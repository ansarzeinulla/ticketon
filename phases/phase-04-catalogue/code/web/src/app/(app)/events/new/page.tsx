import type { Metadata } from "next";

import { CreateEventForm } from "./create-event-form";

export const metadata: Metadata = { title: "Create an event" };

export default function NewEventPage() {
  return <CreateEventForm />;
}
