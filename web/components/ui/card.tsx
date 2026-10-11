import type { ReactNode } from "react";
import React from "react";

import { cardClass } from "./styles";

interface CardProps {
  title?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}

const Card: React.FC<CardProps> = ({
  title,
  actions,
  children,
  className = "",
}) => (
  <section className={`${cardClass} ${className}`}>
    {(title !== undefined || actions !== undefined) && (
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-100 px-4 py-3">
        <h2 className="text-base font-semibold text-gray-900">{title}</h2>
        {actions !== undefined && (
          <div className="flex flex-wrap items-center gap-2">{actions}</div>
        )}
      </div>
    )}
    <div className="p-4">{children}</div>
  </section>
);

export default Card;
