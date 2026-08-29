import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import type { ReactNode } from "react";

import { AuthProvider } from "@/lib/auth-context";
import "./globals.css";

const geistSans = Geist({ variable: "--font-geist-sans", subsets: ["latin"] });
const geistMono = Geist_Mono({ variable: "--font-geist-mono", subsets: ["latin"] });

export const metadata: Metadata = {
  title: {
    default: "BiletFlow Organizer",
    template: "%s · BiletFlow",
  },
  description: "Create events and manage tickets on BiletFlow.",
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
