import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import type {
  CheckpointEstimate,
  CheckpointSplit,
  RaceDayStatus,
} from "gen/races/v1/race_plans_pb";
import { CheckpointStatus, EstimateSource } from "gen/races/v1/race_plans_pb";
import type { Race } from "gen/races/v1/races_pb";
import { toDateString } from "helpers/dates";
import { useEffect, useState } from "react";

const MINUTE_MS = 60 * 1000;

// UseNow returns the current time in milliseconds, updated every 30 seconds.
export const useNow = (): number => {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = globalThis.setInterval(() => {
      setNow(Date.now());
    }, 30 * 1000);
    return () => {
      globalThis.clearInterval(timer);
    };
  }, []);
  return now;
};

const pad = (value: number): string => String(value).padStart(2, "0");

/**
 * FormatClockTime formats a timestamp as a local time, "HH:MM", prefixed with
 * the weekday if it is not today.
 */
export const formatClockTime = (
  timestamp: Timestamp | undefined,
  now: number
): string => {
  if (timestamp === undefined) {
    return "–";
  }
  const date = new Date(timestampMs(timestamp));
  const clock = `${pad(date.getHours())}:${pad(date.getMinutes())}`;
  if (toDateString(date) === toDateString(new Date(now))) {
    return clock;
  }
  const weekday = date.toLocaleDateString(undefined, { weekday: "short" });
  return `${weekday} ${clock}`;
};

// FormatMinutes formats a number of minutes as e.g. "42 min" or "2 h 05 min".
export const formatMinutes = (totalMinutes: number): string => {
  const minutes = Math.round(Math.abs(totalMinutes));
  if (minutes < 60) {
    return `${minutes} min`;
  }
  return `${Math.floor(minutes / 60)} h ${pad(minutes % 60)} min`;
};

// FormatRelative describes a timestamp relative to now, e.g. "in 42 min".
export const formatRelative = (
  timestamp: Timestamp | undefined,
  now: number
): string => {
  if (timestamp === undefined) {
    return "";
  }
  const minutes = (timestampMs(timestamp) - now) / MINUTE_MS;
  if (Math.abs(minutes) < 1) {
    return "now";
  }
  if (minutes > 0) {
    return `in ${formatMinutes(minutes)}`;
  }
  return `${formatMinutes(minutes)} ago`;
};

// FormatDelay formats a delay against the plan, e.g. "+12 min" or "−5 min".
export const formatDelay = (seconds: number): string => {
  const minutes = Math.round(seconds / 60);
  if (minutes === 0) {
    return "on plan";
  }
  if (minutes > 0) {
    return `+${formatMinutes(minutes)}`;
  }
  return `−${formatMinutes(minutes)}`;
};

// DelayClass colors a delay: red when behind the plan, green when ahead.
export const delayClass = (seconds: number): string => {
  const minutes = Math.round(seconds / 60);
  if (minutes > 0) {
    return "text-red-700";
  }
  if (minutes < 0) {
    return "text-emerald-700";
  }
  return "text-gray-600";
};

// BufferClass colors a cutoff buffer: red if missed, amber under 30 min.
export const bufferClass = (seconds: number): string => {
  if (seconds < 0) {
    return "bg-red-100 text-red-800";
  }
  if (seconds < 30 * 60) {
    return "bg-amber-100 text-amber-800";
  }
  return "bg-emerald-50 text-emerald-800";
};

export const formatBuffer = (seconds: number): string => {
  if (seconds < 0) {
    return `missed by ${formatMinutes(seconds / 60)}`;
  }
  return `${formatMinutes(seconds / 60)} buffer`;
};

// PaceText describes the pace factor, e.g. "8% slower than plan".
export const paceText = (paceFactor: number): string => {
  if (paceFactor <= 0) {
    return "on plan";
  }
  const percent = Math.round((paceFactor - 1) * 100);
  if (percent === 0) {
    return "on plan";
  }
  if (percent > 0) {
    return `${percent}% slower than plan`;
  }
  return `${-percent}% faster than plan`;
};

// StatusLabel describes a checkpoint's status. Reaching the finish ends the race.
export const statusLabel = (
  status: CheckpointStatus,
  finish: boolean
): string => {
  switch (status) {
    case CheckpointStatus.ARRIVED: {
      return finish ? "Finished" : "At checkpoint";
    }
    case CheckpointStatus.DEPARTED: {
      return "Left";
    }
    default: {
      return "Upcoming";
    }
  }
};

export const statusClass = (
  status: CheckpointStatus,
  finish: boolean
): string => {
  switch (status) {
    case CheckpointStatus.ARRIVED: {
      return finish
        ? "bg-emerald-100 text-emerald-800 ring-emerald-300"
        : "bg-sky-100 text-sky-800 ring-sky-300";
    }
    case CheckpointStatus.DEPARTED: {
      return "bg-emerald-100 text-emerald-800 ring-emerald-300";
    }
    default: {
      return "bg-gray-100 text-gray-700 ring-gray-300";
    }
  }
};

// NextCheckpoint returns the first checkpoint the runner hasn't left yet.
export const nextCheckpoint = (
  status: RaceDayStatus
): CheckpointEstimate | undefined =>
  status.finished
    ? undefined
    : status.checkpoints.find(
        (checkpoint) =>
          checkpoint.status === CheckpointStatus.UPCOMING ||
          checkpoint.status === CheckpointStatus.ARRIVED
      );

// LastRecorded returns the last checkpoint with a recorded time.
const lastRecorded = (
  status: RaceDayStatus
): CheckpointEstimate | undefined => {
  let last: CheckpointEstimate | undefined = undefined;
  for (const checkpoint of status.checkpoints) {
    if (checkpoint.arrivalRecorded || checkpoint.departureRecorded) {
      last = checkpoint;
    }
  }
  return last;
};

// EstimateSourceText describes what the estimates are based on.
export const estimateSourceText = (
  status: RaceDayStatus,
  now: number
): string => {
  switch (status.estimateSource) {
    case EstimateSource.LIVETRACK: {
      return `Based on LiveTrack, updated ${formatRelative(status.livetrack?.time, now)}`;
    }
    case EstimateSource.CHECKPOINT: {
      const checkpoint = lastRecorded(status);
      return checkpoint === undefined
        ? "Based on the recorded checkpoint times"
        : `Based on the ${checkpoint.name} time`;
    }
    default: {
      return "Following the plan";
    }
  }
};

export const splitFor = (
  splits: CheckpointSplit[],
  index: number
): CheckpointSplit | undefined =>
  splits.find((split) => split.checkpointIndex === index);

/**
 * PlannedStartInput returns the race's planned start as a datetime-local
 * value, interpreted in the viewer's time zone.
 */
export const plannedStartInput = (race: Race | undefined): string => {
  if (race === undefined || race.date === "") {
    return "";
  }
  return `${race.date}T${race.startTime === "" ? "07:00" : race.startTime}`;
};

// DateTimeInput formats milliseconds as a local datetime-local value.
export const dateTimeInput = (milliseconds: number): string => {
  const date = new Date(milliseconds);
  return `${toDateString(date)}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

// TimestampInput formats a timestamp for a datetime-local input.
export const timestampInput = (timestamp: Timestamp | undefined): string =>
  timestamp === undefined ? "" : dateTimeInput(timestampMs(timestamp));
