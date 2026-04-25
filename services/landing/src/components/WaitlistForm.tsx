"use client";

import { useState } from "react";

export default function WaitlistForm() {
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!email.trim()) return;

    try {
      const res = await fetch("/api/waitlist", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      if (!res.ok) throw new Error("Submission failed");
      setSubmitted(true);
    } catch {
      setError("Waitlist is not yet active. Please check back soon.");
    }
  };

  return (
    <div className="w-full max-w-md mx-auto">
      {submitted ? (
        <div className="bg-emerald-500/10 border border-emerald-500/30 rounded-lg p-6 text-center">
          <p className="text-emerald-400 font-medium text-lg">
            Thanks! We&apos;ll be in touch.
          </p>
        </div>
      ) : (
        <form
          onSubmit={handleSubmit}
          className="flex flex-col sm:flex-row gap-3"
        >
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@company.com"
            required
            className="flex-1 px-4 py-3 rounded-lg bg-slate-800 border border-slate-700 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
          />
          <button
            type="submit"
            className="px-6 py-3 rounded-lg bg-brand-600 hover:bg-brand-700 text-white font-medium transition-colors whitespace-nowrap focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400"
          >
            Join Waitlist
          </button>
        </form>
      )}
      {error && (
        <p className="mt-3 text-sm text-amber-400 text-center">{error}</p>
      )}
    </div>
  );
}
