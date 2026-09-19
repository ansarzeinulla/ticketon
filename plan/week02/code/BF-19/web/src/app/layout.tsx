import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import type { ReactNode } from "react";

import { AuthProvider } from "@/lib/auth-context";
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
    default: "BiletFlow",
    template: "%s · BiletFlow",
  },
  description: "Discover events and get tickets on BiletFlow.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full font-sans">
        {/* The provider owns the session, so every route can read the user. */}
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
