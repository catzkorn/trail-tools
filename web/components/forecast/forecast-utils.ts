import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { ForecastPointKind } from "gen/routes/v1/routes_pb";
import { toDateString } from "helpers/dates";

export const forecastKindLabel = (kind: ForecastPointKind): string => {
  switch (kind) {
    case ForecastPointKind.START: {
      return "Start";
    }
    case ForecastPointKind.HIGH_POINT: {
      return "High point";
    }
    case ForecastPointKind.FINISH: {
      return "Finish";
    }
    default: {
      return "";
    }
  }
};

// FormatClock formats a timestamp as a local time, e.g. "Sat 14:00".
export const formatClock = (timestamp: Timestamp | undefined): string => {
  if (timestamp === undefined) {
    return "";
  }
  return timestampDate(timestamp).toLocaleString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    weekday: "short",
  });
};

// FormatTime formats a timestamp as a local time of day, e.g. "14:00".
export const formatTime = (timestamp: Timestamp | undefined): string => {
  if (timestamp === undefined) {
    return "";
  }
  return timestampDate(timestamp).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
  });
};

/**
 * DefaultStart returns tomorrow at 07:00 local time, formatted for a
 * datetime-local input.
 */
export const defaultStart = (): string => {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return `${toDateString(date)}T07:00`;
};

/**
 * ParseLocalDateTime parses a datetime-local input value ("YYYY-MM-DDTHH:MM")
 * as a time in the browser's time zone.
 */
export const parseLocalDateTime = (value: string): Date | null => {
  const [datePart, timePart] = value.split("T");
  if (datePart === undefined || timePart === undefined) {
    return null;
  }
  const [year = 0, month = 1, day = 1] = datePart.split("-").map(Number);
  const [hours = 0, minutes = 0] = timePart.split(":").map(Number);
  const date = new Date(year, month - 1, day, hours, minutes);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date;
};
