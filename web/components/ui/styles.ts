// Shared Tailwind class names for common elements.

export const primaryButton =
  "inline-flex items-center justify-center gap-1.5 rounded-md bg-emerald-700 px-3 py-1.5 text-sm font-medium text-white shadow-sm hover:bg-emerald-800 disabled:cursor-not-allowed disabled:opacity-50";

export const secondaryButton =
  "inline-flex items-center justify-center gap-1.5 rounded-md bg-white px-3 py-1.5 text-sm font-medium text-gray-800 shadow-sm ring-1 ring-gray-300 ring-inset hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50";

export const dangerButton =
  "inline-flex items-center justify-center gap-1.5 rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white shadow-sm hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50";

export const ghostButton =
  "inline-flex items-center justify-center gap-1 rounded-md px-2 py-1 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50";

// Destructive actions, such as deleting, in the secondary and ghost styles.
export const secondaryDangerButton =
  "inline-flex items-center justify-center gap-1.5 rounded-md bg-white px-3 py-1.5 text-sm font-medium text-red-700 shadow-sm ring-1 ring-red-300 ring-inset hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50";

export const ghostDangerButton =
  "inline-flex items-center justify-center gap-1 rounded-md px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50 hover:text-red-700 disabled:cursor-not-allowed disabled:opacity-50";

export const iconButton =
  "inline-flex items-center justify-center rounded-md p-1 text-gray-500 hover:bg-gray-100 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-40";

export const inputClass =
  "block w-full rounded-md border border-gray-300 bg-white px-2.5 py-1.5 text-sm text-gray-900 shadow-sm placeholder:text-gray-400 focus:border-emerald-600 focus:ring-1 focus:ring-emerald-600 focus:outline-none";

export const labelClass = "block text-xs font-medium text-gray-600";

export const linkClass =
  "font-medium text-emerald-700 hover:text-emerald-900 hover:underline";

export const cardClass =
  "rounded-lg border border-gray-200 bg-white shadow-sm print:shadow-none";

export const tableClass = "min-w-full divide-y divide-gray-200 text-sm";

export const thClass =
  "px-2 py-2 text-left text-xs font-semibold tracking-wide text-gray-500 uppercase whitespace-nowrap";

export const tdClass = "px-2 py-2 align-top text-gray-800";
