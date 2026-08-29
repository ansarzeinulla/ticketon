import type { Metadata } from "next";

import { AdminPortal } from "./admin-portal";

export const metadata: Metadata = { title: "Administration" };

export default function AdminPage() {
  return <AdminPortal />;
}
