"use client";

import { useState } from "react";

export default function Navbar() {
  const [mobileOpen, setMobileOpen] = useState(false);

  const scrollTo = (id: string) => {
    const el = document.getElementById(id);
    if (el) {
      el.scrollIntoView({ behavior: "smooth" });
    }
    setMobileOpen(false);
  };

  return (
    <nav className="sticky top-0 z-50 bg-slate-900/80 backdrop-blur-md border-b border-slate-800">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <button
            onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })}
            className="text-xl font-bold text-white tracking-tight hover:text-brand-400 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:rounded"
          >
            Chronoscope
          </button>

          {/* Desktop links */}
          <div className="hidden md:flex items-center space-x-8">
            <button
              onClick={() => scrollTo("features")}
              className="text-slate-300 hover:text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:rounded"
            >
              Features
            </button>
            <button
              onClick={() => scrollTo("pricing")}
              className="text-slate-300 hover:text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:rounded"
            >
              Pricing
            </button>
            <a
              href="https://github.com/chronoscope"
              target="_blank"
              rel="noopener noreferrer"
              className="text-slate-300 hover:text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:rounded-sm"
            >
              GitHub
            </a>
          </div>

          {/* Mobile hamburger */}
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="md:hidden p-2 text-slate-300 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:rounded"
            aria-label="Toggle menu"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              viewBox="0 0 24 24"
            >
              {mobileOpen ? (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M6 18L18 6M6 6l12 12"
                />
              ) : (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M4 6h16M4 12h16M4 18h16"
                />
              )}
            </svg>
          </button>
        </div>
      </div>

      {/* Mobile menu */}
      {mobileOpen && (
        <div className="md:hidden bg-slate-900 border-b border-slate-800">
          <div className="px-4 pt-2 pb-4 space-y-2">
            <button
              onClick={() => scrollTo("features")}
              className="block w-full text-left px-3 py-2 rounded-md text-slate-300 hover:text-white hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400"
            >
              Features
            </button>
            <button
              onClick={() => scrollTo("pricing")}
              className="block w-full text-left px-3 py-2 rounded-md text-slate-300 hover:text-white hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400"
            >
              Pricing
            </button>
            <a
              href="https://github.com/chronoscope"
              target="_blank"
              rel="noopener noreferrer"
              className="block px-3 py-2 rounded-md text-slate-300 hover:text-white hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:rounded-sm"
            >
              GitHub
            </a>
          </div>
        </div>
      )}
    </nav>
  );
}
