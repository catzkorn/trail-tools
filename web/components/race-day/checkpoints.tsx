import { TextBlock } from "@components/race-plan/section";
import { Badge } from "@components/ui/misc";
import { cardClass } from "@components/ui/styles";
import { CheckCircleIcon } from "@heroicons/react/20/solid";
import type {
  Checkpoint,
  CheckpointEstimate,
} from "gen/races/v1/race_plans_pb";
import { CheckpointStatus } from "gen/races/v1/race_plans_pb";
import { formatKm } from "helpers/format";
import React from "react";

import {
  bufferClass,
  delayClass,
  formatBuffer,
  formatClockTime,
  formatDelay,
  formatRelative,
  statusClass,
  statusLabel,
} from "./race-day-utils";

interface TimeValueProps {
  estimate: CheckpointEstimate;
  departure: boolean;
  now: number;
}

// TimeValue shows a recorded time in bold with a check, or an estimate.
const TimeValue: React.FC<TimeValueProps> = ({ estimate, departure, now }) => {
  const time = departure ? estimate.departureTime : estimate.arrivalTime;
  const recorded = departure
    ? estimate.departureRecorded
    : estimate.arrivalRecorded;
  if (time === undefined) {
    return <span className="text-gray-400">–</span>;
  }
  if (recorded) {
    return (
      <span className="inline-flex items-center gap-0.5 font-semibold text-gray-900">
        <CheckCircleIcon className="size-3.5 text-emerald-600" />
        {formatClockTime(time, now)}
      </span>
    );
  }
  return <span className="text-gray-600">~{formatClockTime(time, now)}</span>;
};

interface NextCheckpointProps {
  estimate: CheckpointEstimate;
  checkpoint: Checkpoint | undefined;
  now: number;
  saving: boolean;
  onArrived: () => void;
  onLeft: () => void;
}

/**
 * NextCheckpoint shows where the runner is heading, with big buttons for
 * crew to record the arrival and departure.
 */
export const NextCheckpoint: React.FC<NextCheckpointProps> = ({
  estimate,
  checkpoint,
  now,
  saving,
  onArrived,
  onLeft,
}) => {
  const atCheckpoint = estimate.status === CheckpointStatus.ARRIVED;
  return (
    <section className={`${cardClass} flex flex-col gap-3 p-4`}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <p className="text-xs font-semibold tracking-wide text-gray-500 uppercase">
            {atCheckpoint ? "At checkpoint" : "Next checkpoint"}
          </p>
          <h3 className="text-xl font-bold text-gray-900">
            {estimate.name === "" ? "Checkpoint" : estimate.name}
          </h3>
          <p className="text-sm text-gray-600">
            {formatKm(estimate.distanceMeters)}
          </p>
        </div>
        {estimate.overdue && (
          <Badge className="bg-red-600 text-white ring-red-700">OVERDUE</Badge>
        )}
      </div>

      {estimate.arrivalTime !== undefined && (
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <span className="text-3xl font-bold text-gray-900 tabular-nums">
            {atCheckpoint ? "" : "~"}
            {formatClockTime(estimate.arrivalTime, now)}
          </span>
          <span className="text-base text-gray-700">
            {atCheckpoint ? "arrived " : ""}
            {formatRelative(estimate.arrivalTime, now)}
          </span>
          <span
            className={`text-base font-semibold ${delayClass(estimate.delaySeconds)}`}
          >
            {formatDelay(estimate.delaySeconds)}
          </span>
        </div>
      )}
      {atCheckpoint && estimate.departureTime !== undefined && (
        <p className="text-sm text-gray-700">
          Expected to leave ~{formatClockTime(estimate.departureTime, now)} (
          {formatRelative(estimate.departureTime, now)})
        </p>
      )}
      {estimate.cutoffTime !== undefined && (
        <p className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-gray-700">
            Cutoff {formatClockTime(estimate.cutoffTime, now)}
          </span>
          <span
            className={`rounded px-1.5 py-0.5 font-medium ${bufferClass(estimate.cutoffBufferSeconds)}`}
          >
            {formatBuffer(estimate.cutoffBufferSeconds)}
          </span>
        </p>
      )}

      <div className="grid grid-cols-2 gap-3">
        <button
          type="button"
          disabled={saving || estimate.arrivalRecorded}
          className="h-14 rounded-lg bg-emerald-700 text-lg font-semibold text-white shadow active:bg-emerald-800 disabled:opacity-40"
          onClick={onArrived}
        >
          Arrived now
        </button>
        <button
          type="button"
          disabled={saving || estimate.departureRecorded}
          className="h-14 rounded-lg bg-gray-800 text-lg font-semibold text-white shadow active:bg-gray-900 disabled:opacity-40"
          onClick={onLeft}
        >
          Left now
        </button>
      </div>

      {checkpoint !== undefined && (
        <div className="flex flex-col gap-2 border-t border-gray-100 pt-3">
          <TextBlock
            label="Crew instructions"
            text={checkpoint.crewInstructions}
          />
          <TextBlock label="Drop bag" text={checkpoint.dropBagContents} />
          <TextBlock label="Runner's tasks" text={checkpoint.tasks} />
          <TextBlock label="Aid" text={checkpoint.aid} />
        </div>
      )}
    </section>
  );
};

interface CheckpointListProps {
  estimates: CheckpointEstimate[];
  now: number;
  onSelect: (estimate: CheckpointEstimate) => void;
}

/**
 * CheckpointList lists every checkpoint's planned, recorded and estimated
 * times. Rows open a dialog to edit the recorded times.
 */
export const CheckpointList: React.FC<CheckpointListProps> = ({
  estimates,
  now,
  onSelect,
}) => (
  <section className={`${cardClass} overflow-hidden`}>
    <div className="hidden grid-cols-[2fr_1fr_1fr_1fr_1fr_1.3fr_1fr] gap-2 border-b border-gray-100 px-4 py-2 text-xs font-semibold tracking-wide text-gray-500 uppercase md:grid">
      <span>Checkpoint</span>
      <span>Planned</span>
      <span>Arrival</span>
      <span>Departure</span>
      <span>Delay</span>
      <span>Cutoff</span>
      <span>Status</span>
    </div>
    <ul className="divide-y divide-gray-100">
      {estimates.map((estimate) => (
        <li key={estimate.checkpointIndex}>
          <button
            type="button"
            className="grid w-full grid-cols-2 gap-x-2 gap-y-1 px-4 py-3 text-left text-sm hover:bg-gray-50 active:bg-gray-100 md:grid-cols-[2fr_1fr_1fr_1fr_1fr_1.3fr_1fr] md:items-center"
            onClick={() => {
              onSelect(estimate);
            }}
          >
            <span className="min-w-0">
              <span className="block truncate font-semibold text-gray-900">
                {estimate.name}
              </span>
              <span className="text-xs text-gray-500">
                {formatKm(estimate.distanceMeters)}
              </span>
            </span>
            <span className="justify-self-end md:hidden">
              <Badge className={statusClass(estimate.status)}>
                {statusLabel(estimate.status)}
              </Badge>
              {estimate.overdue && (
                <Badge className="ml-1 bg-red-600 text-white ring-red-700">
                  Overdue
                </Badge>
              )}
            </span>
            <span className="text-gray-600 tabular-nums">
              <span className="text-xs text-gray-400 md:hidden">Plan </span>
              {formatClockTime(estimate.plannedArrivalTime, now)}
            </span>
            <span className="tabular-nums">
              <span className="text-xs text-gray-400 md:hidden">Arr </span>
              <TimeValue estimate={estimate} departure={false} now={now} />
            </span>
            <span className="tabular-nums">
              <span className="text-xs text-gray-400 md:hidden">Dep </span>
              <TimeValue estimate={estimate} departure now={now} />
            </span>
            <span
              className={`tabular-nums ${estimate.arrivalTime === undefined ? "text-gray-400" : delayClass(estimate.delaySeconds)}`}
            >
              {estimate.arrivalTime === undefined
                ? "–"
                : formatDelay(estimate.delaySeconds)}
            </span>
            <span>
              {estimate.cutoffTime === undefined ? (
                <span className="text-gray-400">–</span>
              ) : (
                <span
                  className={`rounded px-1.5 py-0.5 text-xs whitespace-nowrap ${bufferClass(estimate.cutoffBufferSeconds)}`}
                >
                  {formatClockTime(estimate.cutoffTime, now)} ·{" "}
                  {formatBuffer(estimate.cutoffBufferSeconds)}
                </span>
              )}
            </span>
            <span className="hidden md:block">
              <Badge className={statusClass(estimate.status)}>
                {statusLabel(estimate.status)}
              </Badge>
              {estimate.overdue && (
                <Badge className="ml-1 bg-red-600 text-white ring-red-700">
                  Overdue
                </Badge>
              )}
            </span>
          </button>
        </li>
      ))}
    </ul>
    <p className="border-t border-gray-100 px-4 py-2 text-xs text-gray-500">
      Tap a checkpoint to record or correct its times. ~ marks an estimate.
    </p>
  </section>
);
