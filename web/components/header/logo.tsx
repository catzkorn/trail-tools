import React from "react";

interface LogoMarkProps {
  // The color of the summit dot: white on dark grounds, ink on light ones.
  summit?: string;
  className?: string;
}

// LogoMark is the trailrunn.ing mark: a ridge line with a dot on its summit.
export const LogoMark: React.FC<LogoMarkProps> = ({
  summit = "#ffffff",
  className = "h-[25px] w-[38px]",
}) => (
  <svg aria-hidden="true" viewBox="0 0 40 26" fill="none" className={className}>
    <path
      d="M2 22 L9 15 L13 18 L21 6 L26 12 L30 9 L38 20"
      className="stroke-ember"
      strokeWidth="2.6"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <circle cx="21" cy="6" r="2.6" fill={summit} />
    <path
      d="M2 24.5 L38 24.5"
      stroke="#404040"
      strokeWidth="1.5"
      strokeLinecap="round"
    />
  </svg>
);

// Wordmark is the trailrunn.ing name, with .ing in the accent color.
export const Wordmark: React.FC<{ className?: string }> = ({
  className = "",
}) => (
  <span className={`font-brand font-bold ${className}`}>
    trailrunn<span className="text-ember">.ing</span>
  </span>
);
