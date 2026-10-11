import { Badge } from "@components/ui/misc";
import {
  CalendarIcon,
  ClockIcon,
  MapPinIcon,
} from "@heroicons/react/24/outline";
import type {
  RacePlan,
  RacePlanAnalysis,
  RuleResult,
} from "gen/races/v1/race_plans_pb";
import { RuleStatus } from "gen/races/v1/race_plans_pb";
import type { Race } from "gen/races/v1/races_pb";
import type { RoutePoint } from "gen/routes/v1/routes_pb";
import {
  formatDate,
  formatDaysUntil,
  formatDuration,
  formatDurationLong,
  formatKm,
} from "helpers/format";
import { priorityClass, priorityLabel, ruleStatusLabel } from "helpers/labels";
import type { ReactNode } from "react";
import React from "react";

import CourseCard from "./analysis-table";
import CheckpointsSection from "./checkpoints-section";
import GoalsSection from "./goals-section";
import {
  ContingenciesSection,
  GearSection,
  RaceMorningSection,
} from "./list-sections";
import NutritionSection from "./nutrition-section";
import type { SaveSection } from "./section";
import { StatusIcon, sectionElementId } from "./section";
import {
  CrewSection,
  MentalSection,
  NotesSection,
  PacingSection,
} from "./simple-sections";

interface RaceHeaderProps {
  race: Race | undefined;
  children?: ReactNode;
}

// RaceHeader shows the race the plan is for.
export const RaceHeader: React.FC<RaceHeaderProps> = ({ race, children }) => {
  if (race === undefined) {
    return null;
  }
  const facts: string[] = [];
  if (race.distanceMeters > 0) {
    facts.push(formatKm(race.distanceMeters));
  }
  if (race.elevationGainMeters > 0) {
    facts.push(`${Math.round(race.elevationGainMeters)} m gain`);
  }
  if (race.cutoffSeconds > 0) {
    facts.push(`${formatDuration(race.cutoffSeconds)} time limit`);
  }
  return (
    <div className="flex flex-col gap-2">
      {children}
      <div className="flex flex-wrap items-center gap-2">
        <h1 className="text-2xl font-bold text-gray-900">{race.name}</h1>
        <Badge className={priorityClass(race.priority)}>
          {priorityLabel(race.priority)} race
        </Badge>
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-gray-700">
        <span className="inline-flex items-center gap-1">
          <CalendarIcon className="size-4 text-gray-400" />
          {formatDate(race.date)} · {formatDaysUntil(race.date)}
        </span>
        {race.startTime !== "" && (
          <span className="inline-flex items-center gap-1">
            <ClockIcon className="size-4 text-gray-400" />
            Start {race.startTime}
          </span>
        )}
        {race.location !== "" && (
          <span className="inline-flex items-center gap-1">
            <MapPinIcon className="size-4 text-gray-400" />
            {race.location}
          </span>
        )}
        {facts.length > 0 && <span>{facts.join(" · ")}</span>}
        {race.websiteUrl !== "" && (
          <a
            href={race.websiteUrl}
            target="_blank"
            rel="noreferrer"
            className="text-emerald-700 hover:underline print:hidden"
          >
            Race website
          </a>
        )}
      </div>
    </div>
  );
};

const STATUS_ORDER = [
  RuleStatus.FAIL,
  RuleStatus.WARN,
  RuleStatus.PASS,
  RuleStatus.UNSPECIFIED,
];

const scrollToSection = (section: string) => {
  const element = document.querySelector(`#${sectionElementId(section)}`);
  if (element !== null) {
    element.scrollIntoView({ behavior: "smooth", block: "start" });
  }
};

interface RulesPanelProps {
  analysis: RacePlanAnalysis | undefined;
}

// RulesPanel shows the plan checklist and the planned finish.
export const RulesPanel: React.FC<RulesPanelProps> = ({ analysis }) => {
  if (analysis === undefined) {
    return null;
  }
  const rules: RuleResult[] = [];
  for (const status of STATUS_ORDER) {
    for (const rule of analysis.rules) {
      if (rule.status === status) {
        rules.push(rule);
      }
    }
  }
  const fraction =
    analysis.rulesTotal > 0 ? analysis.rulesPassed / analysis.rulesTotal : 0;
  return (
    <div className="flex flex-col gap-4">
      {analysis.plannedFinishSeconds > 0 && (
        <div className="flex flex-col">
          <span className="text-xs font-medium tracking-wide text-gray-500 uppercase">
            Planned finish
          </span>
          <span className="text-2xl font-bold text-gray-900 tabular-nums">
            {formatDurationLong(analysis.plannedFinishSeconds)}
          </span>
          {analysis.plannedFinishClockTime !== "" && (
            <span className="text-sm text-gray-600">
              at {analysis.plannedFinishClockTime}
            </span>
          )}
        </div>
      )}
      <div className="flex flex-col gap-1">
        <div className="flex items-baseline justify-between text-sm">
          <span className="font-semibold text-gray-900">Plan checklist</span>
          <span className="text-gray-600 tabular-nums">
            {analysis.rulesPassed}/{analysis.rulesTotal}
          </span>
        </div>
        <div className="h-2 w-full overflow-hidden rounded-full bg-gray-200">
          <div
            className={`h-full rounded-full ${fraction === 1 ? "bg-emerald-500" : "bg-amber-400"}`}
            style={{ width: `${fraction * 100}%` }}
          />
        </div>
      </div>
      <ul className="flex flex-col gap-1">
        {rules.map((rule) => (
          <li key={rule.id}>
            <button
              type="button"
              className="flex w-full items-start gap-2 rounded-md p-1.5 text-left hover:bg-gray-50"
              title={rule.description}
              onClick={() => {
                scrollToSection(rule.section);
              }}
            >
              <StatusIcon
                status={rule.status}
                className="mt-0.5 size-4 shrink-0"
              />
              <span className="min-w-0">
                <span className="block text-sm font-medium text-gray-900">
                  {rule.title}
                  <span className="sr-only">
                    {" "}
                    ({ruleStatusLabel(rule.status)})
                  </span>
                </span>
                {rule.message !== "" && (
                  <span className="block text-xs text-gray-600">
                    {rule.message}
                  </span>
                )}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
};

const rulesFor = (
  analysis: RacePlanAnalysis | undefined,
  section: string
): RuleResult[] =>
  (analysis?.rules ?? []).filter((rule) => rule.section === section);

interface PlanDocumentProps {
  plan: RacePlan;
  race: Race | undefined;
  analysis: RacePlanAnalysis | undefined;
  coursePoints: RoutePoint[];
  // Undefined when the plan is read only.
  onSave: SaveSection | undefined;
}

// PlanDocument renders all sections of a race plan in order.
const PlanDocument: React.FC<PlanDocumentProps> = ({
  plan,
  race,
  analysis,
  coursePoints,
  onSave,
}) => (
  <div className="flex flex-col gap-4 print:gap-3">
    <GoalsSection
      plan={plan}
      race={race}
      rules={rulesFor(analysis, "goals")}
      onSave={onSave}
    />
    <CheckpointsSection
      plan={plan}
      rules={rulesFor(analysis, "checkpoints")}
      onSave={onSave}
    />
    <CourseCard plan={plan} analysis={analysis} coursePoints={coursePoints} />
    <PacingSection
      plan={plan}
      rules={rulesFor(analysis, "pacing")}
      onSave={onSave}
    />
    <NutritionSection
      plan={plan}
      analysis={analysis}
      rules={rulesFor(analysis, "nutrition")}
      onSave={onSave}
    />
    <GearSection
      plan={plan}
      rules={rulesFor(analysis, "gear")}
      onSave={onSave}
    />
    <CrewSection
      plan={plan}
      rules={rulesFor(analysis, "crew")}
      onSave={onSave}
    />
    <ContingenciesSection
      plan={plan}
      rules={rulesFor(analysis, "contingencies")}
      onSave={onSave}
    />
    <RaceMorningSection
      plan={plan}
      rules={rulesFor(analysis, "race_morning")}
      onSave={onSave}
    />
    <MentalSection
      plan={plan}
      rules={rulesFor(analysis, "mental")}
      onSave={onSave}
    />
    <NotesSection
      plan={plan}
      rules={rulesFor(analysis, "notes")}
      onSave={onSave}
    />
  </div>
);

export default PlanDocument;
