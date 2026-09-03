import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import type { ReactNode } from "react";

import { AnalyticsScripts } from "@/components/analytics-scripts";
import { AuthProvider } from "@/lib/auth-context";
import { getLocale } from "@/lib/i18n/server";
import { I18nProvider } from "@/lib/i18n/context";
import "./globals.css";

// Cyrillic alongside Latin: BiletFlow ships in Kazakh and Russian, and without
// the subset every Kazakh string would silently fall back to a system font.
const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin", "cyrillic"],
});
const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin", "cyrillic"],
});

export const metadata: Metadata = {
  title: {
    default: "BiletFlow Organizer",
    template: "%s · BiletFlow",
  },
  description: "Create events and manage tickets on BiletFlow.",
};

export default async function RootLayout({ children }: { children: ReactNode }) {
  // The locale is resolved once, on the server, and drives both <html lang>
  // and the client-side translator (SRS §685).
  const locale = await getLocale();

  return (
    <html
      lang={locale}
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full font-sans">
        {/* The provider owns the session, so every route can read the user. */}
        <I18nProvider locale={locale}>
          <AuthProvider>{children}</AuthProvider>
        </I18nProvider>
        {/* Renders nothing unless a measurement id is configured. */}
        <AnalyticsScripts />
      </body>
    </html>
  );
}
