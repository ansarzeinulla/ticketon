import type { Metadata } from "next";

import { ProfileForm } from "./profile-form";

export const metadata: Metadata = { title: "Your profile" };

/** The organizer's profile and account settings (SRS 4.1). */
export default function ProfilePage() {
  return <ProfileForm />;
}
