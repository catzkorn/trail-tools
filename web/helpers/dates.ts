/**
 * Date helpers. All dates are local calendar dates formatted as YYYY-MM-DD.
 * Date math is done on local Date objects; never via toISOString, which
 * converts to UTC and can shift the date by one day.
 */

const MILLISECONDS_PER_DAY = 24 * 60 * 60 * 1000;

const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/u;

const pad = (value: number): string => String(value).padStart(2, "0");

export const WEEKDAY_NAMES = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

export const isDateString = (value: string): boolean =>
  DATE_PATTERN.test(value);

// ToDateString formats a local Date as YYYY-MM-DD.
export const toDateString = (date: Date): string =>
  `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;

// ParseDate parses YYYY-MM-DD into a Date at local midnight.
export const parseDate = (value: string): Date => {
  const [year = 0, month = 1, day = 1] = value.split("-").map(Number);
  return new Date(year, month - 1, day);
};

export const today = (): string => toDateString(new Date());

export const addDays = (value: string, days: number): string => {
  const date = parseDate(value);
  date.setDate(date.getDate() + days);
  return toDateString(date);
};

// DaysBetween returns the number of days from one date to another.
export const daysBetween = (from: string, to: string): number =>
  Math.round(
    (parseDate(to).getTime() - parseDate(from).getTime()) / MILLISECONDS_PER_DAY
  );

// StartOfWeek returns the Monday on or before the date.
export const startOfWeek = (value: string): string => {
  const weekday = (parseDate(value).getDay() + 6) % 7;
  return addDays(value, -weekday);
};

export const endOfWeek = (value: string): string =>
  addDays(startOfWeek(value), 6);

export const startOfMonth = (value: string): string => `${value.slice(0, 8)}01`;

export const endOfMonth = (value: string): string => {
  const date = parseDate(value);
  return toDateString(new Date(date.getFullYear(), date.getMonth() + 1, 0));
};

export const addMonths = (value: string, months: number): string => {
  const date = parseDate(value);
  return toDateString(
    new Date(date.getFullYear(), date.getMonth() + months, 1)
  );
};

export const minDate = (first: string, second: string): string =>
  first < second ? first : second;

export const maxDate = (first: string, second: string): string =>
  first > second ? first : second;

// MonthsBetween returns the first day of every month that overlaps the range.
export const monthsBetween = (start: string, end: string): string[] => {
  const months: string[] = [];
  let month = startOfMonth(start);
  while (month <= end) {
    months.push(month);
    month = addMonths(month, 1);
  }
  return months;
};

// WeeksBetween returns the Monday of every week that overlaps the range.
export const weeksBetween = (start: string, end: string): string[] => {
  const weeks: string[] = [];
  let week = startOfWeek(start);
  while (week <= end) {
    weeks.push(week);
    week = addDays(week, 7);
  }
  return weeks;
};

// DaysOfWeek returns the seven dates of the week starting on the Monday.
export const daysOfWeek = (monday: string): string[] => {
  const days: string[] = [];
  for (let offset = 0; offset < 7; offset++) {
    days.push(addDays(monday, offset));
  }
  return days;
};

// IsWithin reports whether the date is within the inclusive range.
export const isWithin = (value: string, start: string, end: string): boolean =>
  value >= start && value <= end;
