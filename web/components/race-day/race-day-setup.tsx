import { create } from "@bufbuild/protobuf";
import { timestampFromDate, timestampNow } from "@bufbuild/protobuf/wkt";
import { parseLocalDateTime } from "@components/forecast/forecast-utils";
import {
  inputClass,
  labelClass,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import { ChevronDownIcon, ChevronRightIcon } from "@heroicons/react/24/outline";
import type { RaceDay, RaceDayStatus } from "gen/races/v1/race_plans_pb";
import { RaceDaySchema, StartTimeSource } from "gen/races/v1/race_plans_pb";
import type { Race } from "gen/races/v1/races_pb";
import { formatKm, formatSpeedAsPace } from "helpers/format";
import React, { useState } from "react";

import {
  formatClockTime,
  formatRelative,
  plannedStartInput,
  timestampInput,
} from "./race-day-utils";

interface RaceDaySetupProps {
  status: RaceDayStatus;
  race: Race | undefined;
  now: number;
  saving: boolean;
  onSave: (raceDay: RaceDay, paths: string[]) => void;
}

// RaceDaySetup sets the start time and the LiveTrack link.
const RaceDaySetup: React.FC<RaceDaySetupProps> = ({
  status,
  race,
  now,
  saving,
  onSave,
}) => {
  const raceDay = status.raceDay ?? create(RaceDaySchema);
  const needsSetup =
    raceDay.startTime === undefined || raceDay.livetrackUrl === "";
  const [open, setOpen] = useState(needsSetup);
  const [start, setStart] = useState(() =>
    raceDay.startTime === undefined
      ? plannedStartInput(race)
      : timestampInput(raceDay.startTime)
  );
  const [livetrackUrl, setLivetrackUrl] = useState(raceDay.livetrackUrl);
  const [validation, setValidation] = useState("");
  const { livetrack } = status;

  return (
    <section className="rounded-lg border border-gray-200 bg-white">
      <button
        type="button"
        className="flex w-full items-center justify-between gap-2 px-4 py-3 text-left"
        onClick={() => {
          setOpen(!open);
        }}
      >
        <span className="font-semibold text-gray-900">
          Setup: start time and LiveTrack
        </span>
        <span className="flex items-center gap-2 text-xs text-gray-500">
          {status.startTime === undefined
            ? "No start time"
            : `Started ${formatClockTime(status.startTime, now)}`}
          {open ? (
            <ChevronDownIcon className="size-4" />
          ) : (
            <ChevronRightIcon className="size-4" />
          )}
        </span>
      </button>
      {status.livetrackError !== "" && (
        <p className="mx-4 mb-3 rounded-md border border-amber-300 bg-amber-50 p-2 text-sm text-amber-900">
          LiveTrack could not be read: {status.livetrackError}. Estimates use
          the recorded checkpoint times instead.
        </p>
      )}
      {open && (
        <div className="flex flex-col gap-5 border-t border-gray-100 px-4 py-4">
          <form
            className="flex flex-col gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              const date = parseLocalDateTime(start);
              if (date === null) {
                setValidation("Enter the start date and time.");
                return;
              }
              setValidation("");
              onSave(
                create(RaceDaySchema, { startTime: timestampFromDate(date) }),
                ["start_time"]
              );
            }}
          >
            <span className={labelClass}>Start time (your local time)</span>
            <div className="flex flex-wrap gap-2">
              <input
                type="datetime-local"
                value={start}
                className={`${inputClass} w-auto grow text-base`}
                aria-label="Start time"
                onChange={(event) => {
                  setStart(event.target.value);
                }}
              />
              <button
                type="submit"
                className={secondaryButton}
                disabled={saving}
              >
                Save
              </button>
              <button
                type="button"
                className={primaryButton}
                disabled={saving}
                onClick={() => {
                  onSave(create(RaceDaySchema, { startTime: timestampNow() }), [
                    "start_time",
                  ]);
                }}
              >
                Started now
              </button>
            </div>
            {validation !== "" && (
              <p className="text-sm text-red-700">{validation}</p>
            )}
            {status.startTimeSource === StartTimeSource.LIVETRACK && (
              <p className="text-xs text-gray-500">
                Using the LiveTrack start time{" "}
                {formatClockTime(status.startTime, now)}.
              </p>
            )}
          </form>

          <form
            className="flex flex-col gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              onSave(
                create(RaceDaySchema, { livetrackUrl: livetrackUrl.trim() }),
                ["livetrack_url"]
              );
            }}
          >
            <span className={labelClass}>Garmin LiveTrack link</span>
            <div className="flex flex-wrap gap-2">
              <input
                type="url"
                value={livetrackUrl}
                className={`${inputClass} w-auto grow text-base`}
                placeholder="https://livetrack.garmin.com/session/…/token/…"
                aria-label="LiveTrack link"
                onChange={(event) => {
                  setLivetrackUrl(event.target.value);
                }}
              />
              <button
                type="submit"
                className={secondaryButton}
                disabled={saving}
              >
                Save
              </button>
              {raceDay.livetrackUrl !== "" && (
                <button
                  type="button"
                  className={secondaryButton}
                  disabled={saving}
                  onClick={() => {
                    setLivetrackUrl("");
                    onSave(create(RaceDaySchema, { livetrackUrl: "" }), [
                      "livetrack_url",
                    ]);
                  }}
                >
                  Remove
                </button>
              )}
            </div>
            <p className="text-xs text-gray-500">
              Paste the Garmin LiveTrack link the runner shares, like
              https://livetrack.garmin.com/session/…/token/…. LiveTrack is
              optional: recording checkpoint times always works.
            </p>
          </form>

          {livetrack !== undefined && (
            <div className="grid grid-cols-2 gap-3 rounded-md bg-gray-50 p-3 text-sm sm:grid-cols-4">
              <div>
                <p className={labelClass}>Last update</p>
                <p className="font-medium">
                  {formatRelative(livetrack.time, now)}
                </p>
              </div>
              <div>
                <p className={labelClass}>Watch distance</p>
                <p className="font-medium">
                  {formatKm(livetrack.distanceMeters)}
                </p>
                <p className="text-xs text-gray-500">
                  {formatKm(livetrack.courseDistanceMeters)} on course
                </p>
              </div>
              <div>
                <p className={labelClass}>Current pace</p>
                <p className="font-medium">
                  {formatSpeedAsPace(livetrack.speedMps)}
                </p>
              </div>
              <div>
                <p className={labelClass}>Activity</p>
                <p className="font-medium">
                  {livetrack.ended ? "Activity ended" : "In progress"}
                </p>
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  );
};

export default RaceDaySetup;
