import type { ReactNode } from "react";
import React from "react";

import { inputClass, labelClass } from "./styles";

interface TextFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  placeholder?: string;
  hint?: string;
  required?: boolean;
  autoFocus?: boolean;
  className?: string;
  inputMode?: "text" | "decimal" | "numeric";
}

export const TextField: React.FC<TextFieldProps> = ({
  label,
  value,
  onChange,
  type = "text",
  placeholder,
  hint,
  required = false,
  autoFocus = false,
  className = "",
  inputMode,
}) => (
  <label className={`flex flex-col gap-1 ${className}`}>
    <span className={labelClass}>{label}</span>
    <input
      type={type}
      value={value}
      placeholder={placeholder}
      required={required}
      autoFocus={autoFocus}
      inputMode={inputMode}
      className={inputClass}
      onChange={(event) => {
        onChange(event.target.value);
      }}
    />
    {hint !== undefined && (
      <span className="text-xs text-gray-500">{hint}</span>
    )}
  </label>
);

interface TextAreaFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  rows?: number;
  placeholder?: string;
  className?: string;
}

export const TextAreaField: React.FC<TextAreaFieldProps> = ({
  label,
  value,
  onChange,
  rows = 3,
  placeholder,
  className = "",
}) => (
  <label className={`flex flex-col gap-1 ${className}`}>
    <span className={labelClass}>{label}</span>
    <textarea
      value={value}
      rows={rows}
      placeholder={placeholder}
      className={inputClass}
      onChange={(event) => {
        onChange(event.target.value);
      }}
    />
  </label>
);

export interface SelectOption {
  value: string;
  label: string;
}

interface SelectFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  className?: string;
}

export const SelectField: React.FC<SelectFieldProps> = ({
  label,
  value,
  onChange,
  options,
  className = "",
}) => (
  <label className={`flex flex-col gap-1 ${className}`}>
    <span className={labelClass}>{label}</span>
    <select
      value={value}
      className={inputClass}
      onChange={(event) => {
        onChange(event.target.value);
      }}
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  </label>
);

interface CheckboxFieldProps {
  label: ReactNode;
  checked: boolean;
  onChange: (checked: boolean) => void;
  className?: string;
}

export const CheckboxField: React.FC<CheckboxFieldProps> = ({
  label,
  checked,
  onChange,
  className = "",
}) => (
  <label
    className={`inline-flex cursor-pointer items-center gap-2 text-sm text-gray-800 ${className}`}
  >
    <input
      type="checkbox"
      checked={checked}
      className="size-4 rounded border-gray-300 accent-brand-700"
      onChange={(event) => {
        onChange(event.target.checked);
      }}
    />
    <span>{label}</span>
  </label>
);
