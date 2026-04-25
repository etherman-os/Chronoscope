"use client";

import Navbar from "@/components/Navbar";
import FeatureCard from "@/components/FeatureCard";
import PricingCard from "@/components/PricingCard";
import WaitlistForm from "@/components/WaitlistForm";

export default function Home() {
  const scrollToWaitlist = () => {
    const el = document.getElementById("waitlist");
    if (el) {
      el.scrollIntoView({ behavior: "smooth" });
    }
  };

  return (
    <main className="min-h-screen bg-slate-900 text-white">
      <Navbar />

      {/* Hero Section */}
      <section className="relative overflow-hidden">
        {/* Gradient background */}
        <div className="absolute inset-0 bg-gradient-to-br from-slate-900 via-slate-900 to-brand-900/30 pointer-events-none" />
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-150 h-150 bg-brand-500/10 rounded-full blur-3xl pointer-events-none" />

        <div className="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-24 sm:py-32 text-center">
          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight mb-6">
            Session Replay for{" "}
            <span className="text-brand-400">Desktop Apps</span>
          </h1>
          <p className="text-lg sm:text-xl text-slate-400 max-w-2xl mx-auto mb-10">
            See every click. Fix every bug. Finally understand how users
            interact with your native applications.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <button
              onClick={scrollToWaitlist}
              className="px-8 py-3 rounded-lg bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-colors w-full sm:w-auto focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400"
            >
              Join Waitlist
            </button>
            <a
              href="https://github.com/chronoscope"
              target="_blank"
              rel="noopener noreferrer"
              className="px-8 py-3 rounded-lg bg-slate-800 hover:bg-slate-700 border border-slate-700 text-white font-semibold transition-colors w-full sm:w-auto focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400"
            >
              View on GitHub
            </a>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20 sm:py-24">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold mb-4">
              Built for modern teams
            </h2>
            <p className="text-slate-400 text-lg max-w-xl mx-auto">
              Everything you need to understand your users, without compromising
              on privacy or control.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <FeatureCard
              icon={
                <svg
                  className="w-10 h-10 text-brand-400"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25m18 0A2.25 2.25 0 0 0 18.75 3H5.25A2.25 2.25 0 0 0 3 5.25m18 0V12a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 12V5.25"
                  />
                </svg>
              }
              title="Cross-Platform"
              description="macOS, Windows, Linux. One SDK for all. Ship consistent replay across every desktop OS your users rely on."
            />
            <FeatureCard
              icon={
                <svg
                  className="w-10 h-10 text-emerald-400"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"
                  />
                </svg>
              }
              title="Privacy First"
              description="PII detection, GDPR compliance, audit logs built-in. Collect what you need, redact what you don't, automatically."
            />
            <FeatureCard
              icon={
                <svg
                  className="w-10 h-10 text-violet-400"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125"
                  />
                </svg>
              }
              title="Self-Hosted"
              description="Your data, your infrastructure. No vendor lock-in. Deploy on-premise or in your own cloud with full control."
            />
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section id="pricing" className="py-20 sm:py-24 bg-slate-950/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold mb-4">
              Simple, transparent pricing
            </h2>
            <p className="text-slate-400 text-lg max-w-xl mx-auto">
              Start free, scale as you grow. No hidden fees, no surprises.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
            <PricingCard
              title="Free"
              price="$0"
              description="Open source, self-hosted, community support"
              features={[
                "Self-hosted deployment",
                "Community support",
                "Core replay features",
                "MIT license",
              ]}
              buttonText="Get Started"
            />
            <PricingCard
              title="Pro"
              price="$49/mo"
              description="Cloud hosted, priority support, advanced analytics"
              features={[
                "Cloud hosting included",
                "Priority email support",
                "Advanced analytics",
                "SSO & team management",
              ]}
              highlighted
              buttonText="Join Waitlist"
            />
            <PricingCard
              title="Enterprise"
              price="Custom"
              description="SLA, dedicated support, on-premise deployment"
              features={[
                "Custom SLA",
                "Dedicated support engineer",
                "On-premise deployment",
                "Custom integrations",
              ]}
              buttonText="Contact Sales"
            />
          </div>
        </div>
      </section>

      {/* Waitlist Section */}
      <section id="waitlist" className="py-20 sm:py-24">
        <div className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Join the waitlist
          </h2>
          <p className="text-slate-400 text-lg mb-10">
            Be the first to get access to Chronoscope. No spam, ever.
          </p>
          <WaitlistForm />
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-slate-800 py-10">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-slate-500 text-sm">
            Built with ❤️ by the open source community
          </p>
          <div className="flex items-center gap-6 text-sm">
            <a
              href="https://github.com/chronoscope"
              target="_blank"
              rel="noopener noreferrer"
              className="text-slate-500 hover:text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:rounded-sm"
            >
              GitHub
            </a>
            <span className="text-slate-600">•</span>
            <span className="text-slate-500">MIT License</span>
          </div>
        </div>
      </footer>
    </main>
  );
}
