import { cardClass } from "@components/ui/styles";
import {
  CheckCircleIcon,
  FlagIcon,
  TrophyIcon,
  XCircleIcon,
} from "@heroicons/react/24/solid";
import type { RaceComparison, RaceReport } from "gen/races/v1/race_reports_pb";
import { ActualTimesSource } from "gen/races/v1/race_reports_pb";
import { formatDate, formatDurationLong, formatKm } from "helpers/format";
import React from "react";
import { Link } from "react-router";

// The sections of a race report's story, in order.
export type SectionKey =
  | "summary"
  | "preparation"
  | "raceDay"
  | "nutrition"
  | "wentWell"
  | "wentBadly"
  | "lessons";

export interface Section {
  key: SectionKey;
  label: string;
  placeholder: string;
}

export const SECTIONS: Section[] = [
  {
    key: "summary",
    label: "Summary",
    placeholder: "The race in a few sentences.",
  },
  {
    key: "preparation",
    label: "Preparation",
    placeholder: "The training block, taper, recce of the course, kit choices.",
  },
  {
    key: "raceDay",
    label: "Race day",
    placeholder:
      "How it went: pacing, the course, the conditions, low and high points.",
  },
  {
    key: "nutrition",
    label: "Nutrition",
    placeholder: "What you planned to eat and drink, what you did, problems.",
  },
  {
    key: "wentWell",
    label: "What went well",
    placeholder: "What worked, and should be kept for the next race.",
  },
  {
    key: "wentBadly",
    label: "What went badly",
    placeholder: "What didn't work.",
  },
  {
    key: "lessons",
    label: "Lessons",
    placeholder: "What to change for the next race.",
  },
];

// Paragraphs splits plain text into paragraphs separated by blank lines.
export const Paragraphs: React.FC<{ text: string; className?: string }> = ({
  text,
  className = "",
}) => (
  <>
    {text
      .split(/\n\s*\n/u)
      .map((paragraph) => paragraph.trim())
      .filter((paragraph) => paragraph !== "")
      .map((paragraph, index) => (
        // eslint-disable-next-line react/no-array-index-key -- paragraphs have no identity
        <p key={index} className={`whitespace-pre-line ${className}`}>
          {paragraph}
        </p>
      ))}
  </>
);

// FormatDifference formats a difference in seconds, e.g. "+0:04:10".
export const formatDifference = (seconds: number): string => {
  let sign = "±";
  if (seconds > 0) {
    sign = "+";
  } else if (seconds < 0) {
    sign = "−";
  }
  return `${sign}${formatDurationLong(Math.abs(seconds))}`;
};

// DifferenceText shows a difference in seconds, green if ahead of the plan.
const DifferenceText: React.FC<{ seconds: number }> = ({ seconds }) => (
  <span
    className={`tabular-nums ${seconds <= 0 ? "text-emerald-700" : "text-rose-700"}`}
  >
    {formatDifference(seconds)}
  </span>
);

// DnfBadge marks the result of a race the runner did not finish.
export const DnfBadge: React.FC = () => (
  <span
    title="Did not finish"
    className="inline-flex items-center rounded-full bg-rose-50 px-2 py-0.5 text-xs font-semibold tracking-wide text-rose-800 ring-1 ring-rose-300 ring-inset"
  >
    DNF
  </span>
);

/**
 * StopText describes where and when a runner who did not finish stopped,
 * e.g. "Stopped at 71.7 km, Function Junction, after 11:29:00".
 */
export const stopText = (report: RaceReport): string => {
  const where = [
    report.stopDistanceMeters > 0 ? formatKm(report.stopDistanceMeters) : "",
    report.stopCheckpoint,
  ]
    .filter((part) => part !== "")
    .join(", ");
  const parts: string[] = [];
  if (where !== "") {
    parts.push(`at ${where}`);
  }
  if (report.stopSeconds > 0) {
    parts.push(`after ${formatDurationLong(report.stopSeconds)}`);
  }
  return parts.length === 0 ? "" : `Stopped ${parts.join(", ")}`;
};

interface RaceResultProps {
  report: RaceReport;
  className?: string;
  iconClassName?: string;
}

/**
 * RaceResult shows the result of a race: the finish time and placement, or
 * that the runner did not finish, and where and when they stopped.
 */
export const RaceResult: React.FC<RaceResultProps> = ({
  report,
  className = "",
  iconClassName = "size-4",
}) => {
  if (report.didNotFinish) {
    return (
      <p className={`inline-flex flex-wrap items-center gap-1.5 ${className}`}>
        <FlagIcon className={`${iconClassName} text-rose-500`} />
        <DnfBadge />
        {[stopText(report), report.placement]
          .filter((part) => part !== "")
          .join(" · ")}
      </p>
    );
  }
  if (report.finishSeconds <= 0 && report.placement === "") {
    return null;
  }
  return (
    <p className={`inline-flex items-center gap-1.5 ${className}`}>
      <TrophyIcon className={`${iconClassName} text-amber-500`} />
      {[
        report.finishSeconds > 0
          ? formatDurationLong(report.finishSeconds)
          : "",
        report.placement,
      ]
        .filter((part) => part !== "")
        .join(" · ")}
    </p>
  );
};

interface ComparisonProps {
  comparison: RaceComparison | undefined;
  finishSeconds: number;
  didNotFinish: boolean;
}

/**
 * RaceComparisonView shows a race plan against what happened: the result
 * against the goals, and the planned and actual arrival times at the
 * checkpoints.
 */
export const RaceComparisonView: React.FC<ComparisonProps> = ({
  comparison,
  finishSeconds,
  didNotFinish,
}) => {
  if (comparison === undefined || !comparison.hasPlan) {
    return (
      <p className="text-sm text-gray-500">
        The race has no race plan to compare with.
      </p>
    );
  }
  const { checkpoints } = comparison;
  return (
    <div className="flex flex-col gap-4">
      {comparison.goals.length > 0 && (
        <ul className="flex flex-wrap gap-2">
          {comparison.goals.map((goal) => (
            <li
              key={goal.goal}
              className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm ring-1 ${
                goal.met
                  ? "bg-emerald-50 text-emerald-900 ring-emerald-200"
                  : "bg-gray-50 text-gray-700 ring-gray-200"
              }`}
            >
              {goal.met ? (
                <CheckCircleIcon className="size-4 text-emerald-600" />
              ) : (
                <XCircleIcon className="size-4 text-gray-400" />
              )}
              <span className="font-semibold">{goal.goal} goal</span>
              <span className="tabular-nums">
                {formatDurationLong(goal.targetSeconds)}
              </span>
              {didNotFinish && (
                <span className="font-semibold text-rose-700">DNF</span>
              )}
              {!didNotFinish && finishSeconds > 0 && (
                <DifferenceText seconds={goal.differenceSeconds} />
              )}
            </li>
          ))}
        </ul>
      )}
      {checkpoints.length > 0 && (
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs font-medium text-gray-500">
                <th className="py-1.5 pr-3">Checkpoint</th>
                <th className="py-1.5 pr-3 text-right">km</th>
                <th className="py-1.5 pr-3 text-right">Planned</th>
                <th className="py-1.5 pr-3 text-right">Actual</th>
                <th className="py-1.5 text-right">Difference</th>
              </tr>
            </thead>
            <tbody>
              {checkpoints.map((cp) => (
                <tr
                  key={`${cp.name}-${cp.distanceMeters}`}
                  className={`border-b border-gray-100 last:border-0 ${cp.notReached ? "text-gray-400" : ""}`}
                >
                  <td
                    className={`py-1.5 pr-3 ${cp.notReached ? "" : "text-gray-900"}`}
                  >
                    {cp.name}
                  </td>
                  <td
                    className={`py-1.5 pr-3 text-right tabular-nums ${cp.notReached ? "" : "text-gray-600"}`}
                  >
                    {(cp.distanceMeters / 1000).toFixed(1)}
                  </td>
                  <td className="py-1.5 pr-3 text-right tabular-nums">
                    {formatDurationLong(cp.plannedArrivalSeconds)}
                  </td>
                  <td className="py-1.5 pr-3 text-right tabular-nums">
                    {cp.notReached && "DNF"}
                    {!cp.notReached &&
                      (cp.actualArrivalSeconds === undefined
                        ? "–"
                        : formatDurationLong(cp.actualArrivalSeconds))}
                  </td>
                  <td className="py-1.5 text-right">
                    {cp.differenceSeconds === undefined ? (
                      "–"
                    ) : (
                      <DifferenceText seconds={cp.differenceSeconds} />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="mt-2 text-xs text-gray-500">
            {comparison.actualTimesSource === ActualTimesSource.ACTIVITY &&
              (comparison.actualTimesScaled
                ? "Actual times from the race's activity, scaled to the race distance."
                : "Actual times from the race's activity, up to where its recording ends.")}
            {comparison.actualTimesSource === ActualTimesSource.RACE_DAY &&
              "Actual times from the checkpoint times recorded on race day."}
            {comparison.actualTimesSource === ActualTimesSource.UNSPECIFIED &&
              "Link the race's activity, or record checkpoint times on race day, to see the actual times."}
            {checkpoints.some((cp) => cp.notReached) &&
              " Checkpoints marked DNF are beyond where the race ended."}
          </p>
        </div>
      )}
    </div>
  );
};

interface RaceReportCardProps {
  report: RaceReport;
  to: string;
  showStatus?: boolean;
}

// RaceReportCard links to a race report with its result.
export const RaceReportCard: React.FC<RaceReportCardProps> = ({
  report,
  to,
  showStatus = false,
}) => (
  <Link
    to={to}
    className={`${cardClass} flex flex-col gap-2 p-4 hover:border-brand-400 hover:shadow`}
  >
    <div className="flex items-start justify-between gap-2">
      <h2 className="leading-snug font-semibold text-gray-900">
        {report.title}
      </h2>
      {showStatus && (
        <span
          className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${
            report.published
              ? "bg-emerald-100 text-emerald-800"
              : "bg-gray-100 text-gray-700"
          }`}
        >
          {report.published ? "Published" : "Draft"}
        </span>
      )}
    </div>
    <p className="text-sm text-gray-600">
      {[
        report.raceName === report.title ? "" : report.raceName,
        formatDate(report.raceDate),
        report.distanceMeters > 0 ? formatKm(report.distanceMeters) : "",
      ]
        .filter((part) => part !== "")
        .join(" · ")}
    </p>
    <RaceResult report={report} className="text-sm font-medium text-gray-900" />
    {report.summary !== "" && (
      <p className="line-clamp-3 text-sm text-gray-700">{report.summary}</p>
    )}
  </Link>
);
