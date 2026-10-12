import { ProgressBar } from "@components/ui/misc";
import { TrophyIcon } from "@heroicons/react/24/solid";
import type { ActivitySummary } from "gen/activities/v1/activities_pb";
import type { PlanPeriod } from "gen/plans/v1/plans_pb";
import type { Race } from "gen/races/v1/races_pb";
import { endOfMonth, isWithin, today } from "helpers/dates";
import { formatShortMonth } from "helpers/format";
import React from "react";

interface MonthStripProps {
  months: string[];
  selected: string;
  onSelect: (month: string) => void;
  periods: Map<string, PlanPeriod>;
  summaries: Map<string, ActivitySummary>;
  goalRaces: Race[];
}

/**
 * MonthStrip is a horizontal timeline of the plan's months with their phase
 * and distance against the target.
 */
const MonthStrip: React.FC<MonthStripProps> = ({
  months,
  selected,
  onSelect,
  periods,
  summaries,
  goalRaces,
}) => {
  const now = today();
  return (
    <div className="flex gap-2 overflow-x-auto pb-2">
      {months.map((month) => {
        const period = periods.get(month);
        const summary = summaries.get(month);
        const monthEnd = endOfMonth(month);
        const races = goalRaces.filter((race) =>
          isWithin(race.date, month, monthEnd)
        );
        const isCurrent = isWithin(now, month, monthEnd);
        const actualKm = (summary?.distanceMeters ?? 0) / 1000;
        const targetKm = (period?.targetDistanceMeters ?? 0) / 1000;
        const isSelected = month === selected;
        return (
          <button
            key={month}
            type="button"
            onClick={() => {
              onSelect(month);
            }}
            className={`flex w-36 shrink-0 flex-col gap-1 rounded-lg border p-2 text-left transition ${
              isSelected
                ? "border-emerald-600 bg-emerald-50 ring-1 ring-emerald-600"
                : "border-gray-200 bg-white hover:border-gray-400"
            }`}
          >
            <span className="flex items-center justify-between text-xs font-semibold text-gray-500 uppercase">
              <span>
                {formatShortMonth(month)} {month.slice(0, 4)}
              </span>
              {isCurrent && (
                <span className="rounded bg-emerald-600 px-1 text-[10px] text-white">
                  now
                </span>
              )}
            </span>
            <span className="truncate text-sm font-medium text-gray-900">
              {period?.title !== undefined && period.title !== ""
                ? period.title
                : "–"}
            </span>
            <span className="text-xs text-gray-600 tabular-nums">
              {actualKm.toFixed(0)}
              {targetKm > 0 ? ` / ${targetKm.toFixed(0)}` : ""} km
            </span>
            <ProgressBar value={actualKm} target={targetKm} />
            {races.map((race) => (
              <span
                key={race.id}
                className="flex items-center gap-1 truncate text-[11px] font-medium text-yellow-800"
              >
                <TrophyIcon className="size-3 shrink-0" />
                <span className="truncate">{race.name}</span>
              </span>
            ))}
          </button>
        );
      })}
    </div>
  );
};

export default MonthStrip;
