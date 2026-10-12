import Loading from "@components/loading";
import RaceDialog from "@components/races/race-dialog";
import ConfirmButton from "@components/ui/confirm-button";
import { Badge, EmptyState, PageHeader } from "@components/ui/misc";
import QueryError, { ErrorText } from "@components/ui/query-error";
import {
  cardClass,
  ghostButton,
  ghostDangerButton,
  linkClass,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  CalendarDaysIcon,
  ClipboardDocumentListIcon,
  DocumentTextIcon,
  MapIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  createRacePlan,
  listRacePlans,
} from "gen/races/v1/race_plans-RacePlanService_connectquery";
import type { RacePlanSummary } from "gen/races/v1/race_plans_pb";
import { RacePlanService } from "gen/races/v1/race_plans_pb";
import {
  createRaceReport,
  listRaceReports,
} from "gen/races/v1/race_reports-RaceReportService_connectquery";
import type { RaceReport } from "gen/races/v1/race_reports_pb";
import { RaceReportService } from "gen/races/v1/race_reports_pb";
import {
  deleteRace,
  listRaces,
  updateRace,
} from "gen/races/v1/races-RaceService_connectquery";
import type { Race } from "gen/races/v1/races_pb";
import { RacePriority, RaceService } from "gen/races/v1/races_pb";
import { parseDate, startOfMonth, today } from "helpers/dates";
import {
  formatDaysUntil,
  formatDuration,
  formatKm,
  formatMonth,
  formatShortMonth,
} from "helpers/format";
import { priorityClass, priorityLabel } from "helpers/labels";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";
import { Link, useNavigate } from "react-router";

// DateTile shows a date like a page of a desk calendar.
const DateTile: React.FC<{ date: string }> = ({ date }) => (
  <div className="flex w-12 shrink-0 flex-col items-center self-start rounded-md border border-gray-200 bg-gray-50 py-1 leading-none">
    <span className="text-[10px] font-semibold text-rose-700 uppercase">
      {formatShortMonth(date)}
    </span>
    <span className="text-lg font-bold text-gray-900">
      {parseDate(date).getDate()}
    </span>
    <span className="text-[10px] text-gray-500">{date.slice(0, 4)}</span>
  </div>
);

interface RaceRowProps {
  race: Race;
  plan: RacePlanSummary | undefined;
  report: RaceReport | undefined;
  past: boolean;
  onEdit: () => void;
}

const RaceRow: React.FC<RaceRowProps> = ({
  race,
  plan,
  report,
  past,
  onEdit,
}) => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const remove = useMutation(deleteRace, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [RaceService, RacePlanService]);
    },
  });
  const setOnCalendar = useMutation(updateRace, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [RaceService, RacePlanService]);
    },
  });
  const createPlan = useMutation(createRacePlan, {
    onSuccess: async (response) => {
      await invalidateServices(queryClient, [RacePlanService]);
      if (response.plan !== undefined) {
        void navigate(`/race-plans/${response.plan.id}`);
      }
    },
  });
  const toggleCalendar = (onCalendar: boolean) => {
    setOnCalendar.mutate({
      race: { id: race.id, onCalendar },
      updateMask: { paths: ["on_calendar"] },
    });
  };

  const facts = [
    race.distanceMeters > 0 ? formatKm(race.distanceMeters) : "",
    race.elevationGainMeters > 0
      ? `${Math.round(race.elevationGainMeters)} m gain`
      : "",
    race.cutoffSeconds > 0 ? `${formatDuration(race.cutoffSeconds)} limit` : "",
    race.startTime === "" ? "" : `starts ${race.startTime}`,
  ].filter((part) => part !== "");

  const writeReport = useMutation(createRaceReport, {
    onSuccess: async (response) => {
      await invalidateServices(queryClient, [RaceReportService]);
      if (response.report !== undefined) {
        void navigate(`/race-reports/${response.report.id}`);
      }
    },
  });
  let reportAction: React.ReactNode = null;
  if (report !== undefined) {
    reportAction = (
      <Link to={`/race-reports/${report.id}`} className={secondaryButton}>
        <DocumentTextIcon className="size-4" />
        Race report
      </Link>
    );
  } else if (race.onCalendar && past) {
    reportAction = (
      <button
        type="button"
        className={secondaryButton}
        disabled={writeReport.isPending}
        onClick={() => {
          writeReport.mutate({ raceId: race.id });
        }}
      >
        <DocumentTextIcon className="size-4" />
        Write race report
      </button>
    );
  }

  let planAction: React.ReactNode = null;
  if (plan !== undefined) {
    planAction = (
      <Link to={`/race-plans/${plan.id}`} className={primaryButton}>
        <ClipboardDocumentListIcon className="size-4" />
        Race plan
        <span className="rounded bg-white/20 px-1 text-xs">
          {plan.rulesPassed}/{plan.rulesTotal}
        </span>
      </Link>
    );
  } else if (race.onCalendar && !past) {
    planAction = (
      <button
        type="button"
        className={primaryButton}
        disabled={createPlan.isPending}
        onClick={() => {
          createPlan.mutate({ raceId: race.id });
        }}
      >
        <ClipboardDocumentListIcon className="size-4" />
        Create race plan
      </button>
    );
  }

  return (
    <li className={`${cardClass} flex gap-3 p-3 ${past ? "opacity-70" : ""}`}>
      <DateTile date={race.date} />
      <div className="flex min-w-0 grow flex-col gap-2">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <h3 className="leading-snug font-semibold text-gray-900">
              {race.websiteUrl === "" ? (
                race.name
              ) : (
                <a
                  href={race.websiteUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="hover:underline"
                >
                  {race.name}
                </a>
              )}
            </h3>
            <p className="text-xs text-gray-700">
              <span className="sm:hidden">
                {formatDaysUntil(race.date)}
                {facts.length > 0 && " · "}
              </span>
              {facts.join(" · ")}
            </p>
            {race.location !== "" && (
              <p className="truncate text-xs text-gray-500">{race.location}</p>
            )}
          </div>
          <span className="hidden shrink-0 text-xs whitespace-nowrap text-gray-500 sm:block">
            {formatDaysUntil(race.date)}
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-1">
          {race.priority !== RacePriority.UNSPECIFIED && (
            <Badge className={priorityClass(race.priority)}>
              {priorityLabel(race.priority)}
            </Badge>
          )}
          {!race.onCalendar && !past && (
            <button
              type="button"
              className={secondaryButton}
              disabled={setOnCalendar.isPending}
              onClick={() => {
                toggleCalendar(true);
              }}
            >
              <CalendarDaysIcon className="size-4" />
              Add to calendar
            </button>
          )}
          {planAction}
          {reportAction}
          {race.courseRouteId !== "" && (
            <Link
              to={`/routes/${race.courseRouteId}`}
              className={`${linkClass} inline-flex items-center gap-1 px-1 text-xs`}
            >
              <MapIcon className="size-3.5" />
              Course
            </Link>
          )}
          <div className="ml-auto flex items-center">
            {race.onCalendar && (
              <button
                type="button"
                className={ghostButton}
                disabled={setOnCalendar.isPending}
                title="Remove from calendar"
                aria-label="Remove from calendar"
                onClick={() => {
                  toggleCalendar(false);
                }}
              >
                <XMarkIcon className="size-4" />
              </button>
            )}
            <button
              type="button"
              className={ghostButton}
              onClick={onEdit}
              title="Edit race"
              aria-label="Edit race"
            >
              <PencilSquareIcon className="size-4" />
            </button>
            <ConfirmButton
              className={ghostDangerButton}
              title="Delete race?"
              message={`This deletes "${race.name}" and its race plan.`}
              disabled={remove.isPending}
              onConfirm={() => {
                remove.mutate({ id: race.id });
              }}
            >
              <TrashIcon className="size-4" aria-label="Delete race" />
            </ConfirmButton>
          </div>
        </div>
        <ErrorText
          error={
            remove.error ??
            setOnCalendar.error ??
            createPlan.error ??
            writeReport.error
          }
        />
      </div>
    </li>
  );
};

interface RaceListProps {
  races: Race[];
  planByRace: Map<string, RacePlanSummary>;
  reportByRace: Map<string, RaceReport>;
  past?: boolean;
  onEdit: (race: Race) => void;
}

const RaceList: React.FC<RaceListProps> = ({
  races,
  planByRace,
  reportByRace,
  past = false,
  onEdit,
}) => (
  <ul className="flex flex-col gap-2">
    {races.map((race) => (
      <RaceRow
        key={race.id}
        race={race}
        plan={planByRace.get(race.id)}
        report={reportByRace.get(race.id)}
        past={past}
        onEdit={() => {
          onEdit(race);
        }}
      />
    ))}
  </ul>
);

// GroupByMonth groups races in date order by the month they take place in.
const groupByMonth = (races: Race[]): [string, Race[]][] => {
  const groups: [string, Race[]][] = [];
  for (const race of races) {
    const month = startOfMonth(race.date);
    const last = groups.at(-1);
    if (last !== undefined && last[0] === month) {
      last[1].push(race);
    } else {
      groups.push([month, [race]]);
    }
  }
  return groups;
};

interface SectionProps {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}

const Section: React.FC<SectionProps> = ({ title, subtitle, children }) => (
  <section className="flex flex-col gap-2">
    <div>
      <h2 className="text-base font-semibold text-gray-900">{title}</h2>
      {subtitle !== undefined && (
        <p className="text-sm text-gray-600">{subtitle}</p>
      )}
    </div>
    {children}
  </section>
);

interface RaceEdit {
  race: Race | undefined;
}

const RacesPage: React.FC = () => {
  const { isPending, isError, data, error, refetch } = useQuery(listRaces, {});
  const plans = useQuery(listRacePlans);
  const reports = useQuery(listRaceReports, {});
  const reportByRace = new Map<string, RaceReport>();
  for (const report of reports.data?.reports ?? []) {
    if (report.raceId !== "") {
      reportByRace.set(report.raceId, report);
    }
  }
  const [editing, setEditing] = useState<RaceEdit | null>(null);
  const edit = (race: Race) => {
    setEditing({ race });
  };

  const planByRace = new Map<string, RacePlanSummary>();
  for (const plan of plans.data?.plans ?? []) {
    if (plan.race !== undefined) {
      planByRace.set(plan.race.id, plan);
    }
  }

  let content: React.ReactNode = null;
  if (isPending) {
    content = <Loading />;
  } else if (isError) {
    content = (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  } else if (data.races.length === 0) {
    content = (
      <EmptyState title="No races yet">
        Add the races you could run, then add the ones you&apos;re running to
        your calendar to plan them.
      </EmptyState>
    );
  } else {
    const now = today();
    const upcoming = data.races.filter((race) => race.date >= now);
    const onCalendar = upcoming.filter((race) => race.onCalendar);
    const couldRun = upcoming.filter((race) => !race.onCalendar);
    // The most recent past race first.
    const past: Race[] = [];
    for (const race of data.races) {
      if (race.date < now) {
        past.unshift(race);
      }
    }
    content = (
      <div className="flex flex-col gap-8">
        <Section
          title="On my calendar"
          subtitle="The races you're running, with a race plan for each."
        >
          {onCalendar.length === 0 ? (
            <p className="text-sm text-gray-500">
              No races on your calendar. Add the races you&apos;re running from
              the races you could run below.
            </p>
          ) : (
            <RaceList
              races={onCalendar}
              planByRace={planByRace}
              reportByRace={reportByRace}
              onEdit={edit}
            />
          )}
        </Section>
        {couldRun.length > 0 && (
          <Section title="Races I could run">
            {groupByMonth(couldRun).map(([month, races]) => (
              <div key={month} className="flex flex-col gap-2">
                <h3 className="pt-1 text-xs font-semibold tracking-wide text-gray-500 uppercase">
                  {formatMonth(month)}
                </h3>
                <RaceList
                  races={races}
                  planByRace={planByRace}
                  reportByRace={reportByRace}
                  onEdit={edit}
                />
              </div>
            ))}
          </Section>
        )}
        {past.length > 0 && (
          <Section title="Past races">
            <RaceList
              races={past}
              planByRace={planByRace}
              reportByRace={reportByRace}
              past
              onEdit={edit}
            />
          </Section>
        )}
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Races"
        subtitle="Races you could run, and the ones on your calendar."
        actions={
          <div className="flex gap-2">
            <Link to="/calendar" className={secondaryButton}>
              <CalendarDaysIcon className="size-4" />
              Calendar
            </Link>
            <button
              type="button"
              className={primaryButton}
              onClick={() => {
                setEditing({ race: undefined });
              }}
            >
              <PlusIcon className="size-4" />
              Add race
            </button>
          </div>
        }
      />
      {content}
      {editing !== null && (
        <RaceDialog
          race={editing.race}
          onClose={() => {
            setEditing(null);
          }}
        />
      )}
    </div>
  );
};

export default RacesPage;
