import type { Metadata } from "next";
import Link from "next/link";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Finance Tracker",
  description: "Track your finances",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="scroll-smooth">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased text-gray-900 bg-gray-100`}
      >
        <header className="top-0 left-0 w-full bg-cyan-500 shadow z-50">
          <div className="container mx-auto px-4 py-4">
            <h1 className="text-2xl font-bold text-white">
              <Link href="/">Finance Tracker</Link>
            </h1>
          </div>
        </header>
        <main className="container mx-auto px-4 py-8">
          {children}
        </main>
        <footer className="bg-white shadow mt-8">
          <div className="container mx-auto px-4 py-6 text-center">
            © {new Date().getFullYear()} Finance Tracker. All rights reserved.
          </div>
        </footer>
      </body>
    </html>
  );
}
