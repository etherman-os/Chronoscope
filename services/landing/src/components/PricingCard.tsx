import React from "react";

interface PricingCardProps {
  title: string;
  price: string;
  description: string;
  features: string[];
  highlighted?: boolean;
  buttonText?: string;
}

export default function PricingCard({
  title,
  price,
  description,
  features,
  highlighted = false,
  buttonText = "Get Started",
}: PricingCardProps) {
  return (
    <div
      className={`rounded-xl p-6 border ${
        highlighted
          ? "border-brand-500 bg-brand-900/20 ring-1 ring-brand-500/50"
          : "border-slate-700 bg-slate-800/50"
      } flex flex-col`}
    >
      <h3 className="text-lg font-semibold text-white">{title}</h3>
      <div className="mt-2 mb-1">
        <span className="text-3xl font-bold text-white">{price}</span>
      </div>
      <p className="text-slate-400 text-sm mb-4">{description}</p>
      <ul className="space-y-2 mb-6 flex-1">
        {features.map((feature, index) => (
          <li key={index} className="flex items-start text-slate-300 text-sm">
            <span className="mr-2 text-emerald-400">✓</span>
            {feature}
          </li>
        ))}
      </ul>
      <button
        type="button"
        className={`w-full py-2.5 rounded-lg font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 ${
          highlighted
            ? "bg-brand-600 hover:bg-brand-700 text-white"
            : "bg-slate-700 hover:bg-slate-600 text-white"
        }`}
      >
        {buttonText}
      </button>
    </div>
  );
}
