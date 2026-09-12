import type { Metadata } from "next";

import { ReportsQueue } from "./reports-queue";

export const metadata: Metadata = { title: "Reported events" };

/** The platform moderation queue (SRS 4.12). */
export default function ReportsPage() {
  return <ReportsQueue />;
}
