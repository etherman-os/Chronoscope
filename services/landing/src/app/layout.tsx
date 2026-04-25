import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Chronoscope — Session Replay for Desktop Apps",
  description:
    "Free, open source session replay infrastructure for macOS, Windows, and Linux desktop applications.",
  keywords: [
    "session replay",
    "desktop apps",
    "macOS",
    "Windows",
    "Linux",
    "open source",
  ],
  openGraph: {
    title: "Chronoscope — Session Replay for Desktop Apps",
    description: "See every click. Fix every bug. Free and open source.",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Chronoscope",
    description: "Session replay for desktop apps. Free. Open source.",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
