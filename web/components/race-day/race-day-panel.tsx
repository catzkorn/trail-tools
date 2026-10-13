import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampMs, timestampNow } from "@bufbuild/protobuf/wkt";
import Loading from "@components/loading";
import { ProgressBar } from "@components/ui/misc";
import QueryError, { ErrorText } from "@components/ui/query-error";
import {
  cardClass,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import { SignalIcon } from "@heroicons/react/24/outline";
import type {
  CheckpointEstimate,
  CheckpointSplit,
  RaceDay,
  RacePlan,
  RaceDayStatus,
} from "gen/races/v1/race_plans_pb";
import { CheckpointSplitSchema } from "gen/races/v1/race_plans_pb";
import type { Race } from "gen/races/v1/races_pb";
import type { RoutePoint } from "gen/routes/v1/routes_pb";
import { daysBetween, today } from "helpers/dates";
import { formatDurationLong, formatKm } from "helpers/format";
import React, { useEffect, useState } from "react";

import { CheckpointList, NextCheckpoint } from "./checkpoints";
import RaceDaySetup from "./race-day-setup";
import {
  estimateSourceText,
  formatClockTime,
  formatRelative,
  nextCheckpoint,
  paceText,
  splitFor,
  useNow,
} from "./race-day-utils";
import RaceDayVisuals from "./race-day-visuals";
import SplitDialog from "./split-dialog";
import type { RaceDayState } from "./use-race-day";

interface UndoAction {
  message: string;
  // The checkpoint's split before the change, to restore it.
  split: CheckpointSplit;
}

// RaceDistance is the race distance from the race, the plan or the course.
const raceDistance = (
  race: Race | undefined,
  plan: RacePlan,
  course: RoutePoint[]
): number => {
  if (race !== undefined && race.distanceMeters > 0) {
    return race.distanceMeters;
  }
  const lastCheckpoint = plan.checkpoints.at(-1);
  if (lastCheckpoint !== undefined && lastCheckpoint.distanceMeters > 0) {
    return lastCheckpoint.distanceMeters;
  }
  return course.at(-1)?.distanceMeters ?? 0;
};

const hasRaceDayData = (status: RaceDayStatus): boolean => {
  const { raceDay } = status;
  if (status.startTime !== undefined) {
    return true;
  }
  if (raceDay === undefined) {
    return false;
  }
  return raceDay.livetrackUrl !== "" || raceDay.splits.length > 0;
};

interface SummaryProps {
  status: RaceDayStatus;
  totalMeters: number;
  now: number;
}

const Summary: React.FC<SummaryProps> = ({ status, totalMeters, now }) => {
  if (status.startTime === undefined) {
    return (
      <p className="text-sm text-gray-700">
        Set the start time below, or link LiveTrack, to see estimated arrival
        times.
      </p>
    );
  }
  const finish = status.estimatedFinishTime;
  const elapsed =
    finish === undefined
      ? 0
      : (timestampMs(finish) - timestampMs(status.startTime)) / 1000;
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end gap-x-6 gap-y-2">
        <div>
          <p className="text-xs font-semibold tracking-wide text-gray-500 uppercase">
            {status.finished ? "Finished" : "Estimated finish"}
          </p>
          <p className="text-3xl font-bold text-gray-900 tabular-nums">
            {formatClockTime(finish, now)}
          </p>
          {finish !== undefined && (
            <p className="text-sm text-gray-600">
              {formatDurationLong(elapsed)} elapsed ·{" "}
              {formatRelative(finish, now)}
            </p>
          )}
        </div>
        <div>
          <p className="text-lg font-semibold text-gray-900">
            {paceText(status.paceFactor)}
          </p>
          <p className="text-sm text-gray-600">
            {estimateSourceText(status, now)}
          </p>
        </div>
      </div>
      <div className="flex flex-col gap-1">
        <p className="text-sm text-gray-700 tabular-nums">
          About {formatKm(status.estimatedDistanceMeters)}
          {totalMeters > 0 && ` of ${formatKm(totalMeters)}`} covered
          {status.distanceMeters > 0 && (
            <span className="text-gray-500">
              {" "}
              (last known {formatKm(status.distanceMeters)})
            </span>
          )}
        </p>
        {totalMeters > 0 && (
          <ProgressBar
            value={status.estimatedDistanceMeters}
            target={totalMeters}
            className="bg-emerald-500"
          />
        )}
      </div>
    </div>
  );
};

interface RaceDayPanelProps {
  /**
   * The race day state from useRaceDay, which the page calls so the status
   * loads alongside the plan.
   */
  raceDay: RaceDayState;
  plan: RacePlan;
  race: Race | undefined;
  course: RoutePoint[];
  // Shown on the shared page: anyone with the link can record times.
  shared: boolean;
}

/**
 * RaceDayPanel lets crew follow the runner on race day: estimated arrival
 * times, the next checkpoint with buttons to record arrival and departure,
 * all checkpoint times, the LiveTrack link and the runner's position.
 */
const RaceDayPanel: React.FC<RaceDayPanelProps> = ({
  raceDay,
  plan,
  race,
  course,
  shared,
}) => {
  const now = useNow();
  const [expanded, setExpanded] = useState<boolean | null>(null);
  const [editing, setEditing] = useState<CheckpointEstimate | null>(null);
  const [undo, setUndo] = useState<UndoAction | null>(null);

  // Hide the undo toast after a while.
  useEffect(() => {
    if (undo === null) {
      return undefined;
    }
    const timer = globalThis.setTimeout(() => {
      setUndo(null);
    }, 15 * 1000);
    return () => {
      globalThis.clearTimeout(timer);
    };
  }, [undo]);

  if (raceDay.isPending) {
    return (
      <div className={`${cardClass} p-4`}>
        <Loading />
      </div>
    );
  }
  if (raceDay.error !== null || raceDay.status === undefined) {
    return raceDay.error === null ? null : (
      <QueryError error={raceDay.error} onRetry={raceDay.refetch} />
    );
  }
  const { status } = raceDay;

  const daysAway =
    race === undefined || race.date === ""
      ? 0
      : daysBetween(today(), race.date);
  const autoOpen = hasRaceDayData(status) || Math.abs(daysAway) <= 1;
  const open = expanded ?? autoOpen;

  if (!open) {
    return (
      <section
        className={`${cardClass} flex flex-wrap items-center justify-between gap-3 p-4 print:hidden`}
      >
        <div className="flex items-start gap-3">
          <SignalIcon className="mt-0.5 size-6 shrink-0 text-emerald-700" />
          <div>
            <h2 className="font-semibold text-gray-900">Race day tracking</h2>
            <p className="text-sm text-gray-600">
              On race day, follow the runner here: link Garmin LiveTrack and
              record checkpoint times for live arrival estimates.
            </p>
          </div>
        </div>
        <button
          type="button"
          className={secondaryButton}
          onClick={() => {
            setExpanded(true);
          }}
        >
          Open
        </button>
      </section>
    );
  }

  const splits = status.raceDay?.splits ?? [];

  const save = (raceDayUpdate: RaceDay, paths: string[]) => {
    raceDay.save(raceDayUpdate, paths).catch((error: unknown) => {
      console.error("failed to update race day:", error);
    });
  };

  const record = (
    estimate: CheckpointEstimate,
    arrival: Timestamp | undefined,
    departure: Timestamp | undefined,
    message: string
  ) => {
    const index = estimate.checkpointIndex;
    const previous =
      splitFor(splits, index) ??
      create(CheckpointSplitSchema, { checkpointIndex: index });
    raceDay
      .saveSplit(
        create(CheckpointSplitSchema, {
          arrivalTime: arrival,
          checkpointIndex: index,
          departureTime: departure,
        })
      )
      .then(() => {
        setUndo({ message, split: previous });
      })
      .catch((error: unknown) => {
        console.error("failed to record checkpoint times:", error);
      });
  };

  const next =
    status.startTime === undefined ? undefined : nextCheckpoint(status);
  const totalMeters = raceDistance(race, plan, course);

  return (
    <section className="flex flex-col gap-4 print:hidden">
      <div className={`${cardClass} flex flex-col gap-3 p-4`}>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="flex items-center gap-2 text-lg font-bold text-gray-900">
            <SignalIcon className="size-5 text-emerald-700" />
            Race day
          </h2>
          <button
            type="button"
            className="text-sm text-gray-500 hover:underline"
            onClick={() => {
              setExpanded(false);
            }}
          >
            Hide
          </button>
        </div>
        <Summary status={status} totalMeters={totalMeters} now={now} />
        {shared && (
          <p className="text-xs text-gray-500">
            Anyone with this link can record times and set the LiveTrack link,
            so crew can update the race without an account.
          </p>
        )}
        <ErrorText error={raceDay.saveError} />
      </div>

      {next !== undefined && (
        <NextCheckpoint
          estimate={next}
          finish={next.checkpointIndex === status.checkpoints.length - 1}
          checkpoint={plan.checkpoints.at(next.checkpointIndex)}
          now={now}
          saving={raceDay.saving}
          onArrived={() => {
            const split = splitFor(splits, next.checkpointIndex);
            record(
              next,
              timestampNow(),
              split?.departureTime,
              next.checkpointIndex === status.checkpoints.length - 1
                ? "Finished"
                : `Arrived at ${next.name}`
            );
          }}
          onLeft={() => {
            const split = splitFor(splits, next.checkpointIndex);
            record(
              next,
              split?.arrivalTime,
              timestampNow(),
              `Left ${next.name}`
            );
          }}
        />
      )}

      {status.checkpoints.length === 0 ? (
        <p className="rounded-lg border border-dashed border-gray-300 bg-white/60 p-4 text-sm text-gray-600">
          The plan has no checkpoints yet, so there is nothing to track.
        </p>
      ) : (
        <CheckpointList
          estimates={status.checkpoints}
          now={now}
          onSelect={setEditing}
        />
      )}

      <RaceDaySetup
        status={status}
        race={race}
        now={now}
        saving={raceDay.saving}
        onSave={save}
      />

      <RaceDayVisuals status={status} course={course} now={now} />

      {editing !== null && (
        <SplitDialog
          name={editing.name}
          finish={editing.checkpointIndex === status.checkpoints.length - 1}
          split={splitFor(splits, editing.checkpointIndex)}
          saving={raceDay.saving}
          error={raceDay.saveError}
          onClose={() => {
            setEditing(null);
          }}
          onSave={(arrival, departure) => {
            const estimate = editing;
            setEditing(null);
            record(estimate, arrival, departure, `Updated ${estimate.name}`);
          }}
        />
      )}

      {undo !== null && (
        <div className="fixed inset-x-3 bottom-4 z-[1500] mx-auto flex max-w-md items-center justify-between gap-3 rounded-lg bg-gray-900 px-4 py-3 text-white shadow-xl">
          <span className="text-sm">{undo.message}</span>
          <button
            type="button"
            className={`${primaryButton} px-4`}
            disabled={raceDay.saving}
            onClick={() => {
              const previous = undo.split;
              setUndo(null);
              raceDay.saveSplit(previous).catch((error: unknown) => {
                console.error("failed to undo:", error);
              });
            }}
          >
            Undo
          </button>
        </div>
      )}
    </section>
  );
};

export default RaceDayPanel;
