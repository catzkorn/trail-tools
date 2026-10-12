import Loading from "@components/loading";
import {
  groupActivitiesByDate,
  groupWorkoutsByDate,
} from "@components/plans/plan-utils";
import WeekDays from "@components/plans/week-days";
import { PageHeader } from "@components/ui/misc";
import QueryError from "@components/ui/query-error";
import {
  cardClass,
  ghostButton,
  iconButton,
  secondaryButton,
} from "@components/ui/styles";
import { useQuery } from "@connectrpc/connect-query";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  TrophyIcon,
} from "@heroicons/react/24/outline";
import { listActivities } from "gen/activities/v1/activities-ActivityService_connectquery";
import { listWorkouts } from "gen/plans/v1/plans-TrainingPlanService_connectquery";
import type { Workout } from "gen/plans/v1/plans_pb";
import { listRaces } from "gen/races/v1/races-RaceService_connectquery";
import type { Race } from "gen/races/v1/races_pb";
import {
  addMonths,
  daysOfWeek,
  endOfMonth,
  endOfWeek,
  isDateString,
  startOfMonth,
  startOfWeek,
  today,
  weeksBetween,
} from "helpers/dates";
import { formatKm, formatMonth, formatShortDate } from "helpers/format";
import { FOOT_SPORT_TYPES } from "helpers/labels";
import React from "react";
import { Link, useNavigate, useSearchParams } from "react-router";

// The month shown when the URL doesn't name one, e.g. "2026-10-01".
const currentMonth = (): string => startOfMonth(today());

// ParseMonth parses the month parameter, "YYYY-MM" or "YYYY-MM-DD".
const parseMonth = (value: string | null): string => {
  if (value === null) {
    return currentMonth();
  }
  const date = value.length === 7 ? `${value}-01` : value;
  return isDateString(date) ? startOfMonth(date) : currentMonth();
};

/**
 * CalendarPage shows a month of the races on the user's calendar, the
 * workouts of their training plans and the activities they did.
 */
const CalendarPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const month = parseMonth(searchParams.get("month"));
  const monthEnd = endOfMonth(month);
  const viewStart = startOfWeek(month);
  const viewEnd = endOfWeek(monthEnd);
  const selectMonth = (value: string) => {
    setSearchParams({ month: value.slice(0, 7) }, { replace: true });
  };

  const races = useQuery(listRaces, { onCalendarOnly: true });
  const workouts = useQuery(listWorkouts, {
    endDate: viewEnd,
    startDate: viewStart,
  });
  const activities = useQuery(listActivities, {
    endDate: viewEnd,
    pageSize: 500,
    startDate: viewStart,
  });

  let content: React.ReactNode = null;
  const error = races.error ?? workouts.error ?? activities.error;
  if (error !== null) {
    content = (
      <QueryError
        error={error}
        onRetry={() => {
          void races.refetch();
          void workouts.refetch();
          void activities.refetch();
        }}
      />
    );
  } else if (
    races.data === undefined ||
    workouts.data === undefined ||
    activities.data === undefined
  ) {
    content = <Loading />;
  } else {
    const racesByDate = new Map<string, Race>();
    for (const race of races.data.races) {
      racesByDate.set(race.date, race);
    }
    const workoutsByDate = groupWorkoutsByDate(workouts.data.workouts);
    const activitiesByDate = groupActivitiesByDate(activities.data.activities);
    const monthRaces = races.data.races.filter(
      (race) => race.date >= month && race.date <= monthEnd
    );
    const openWorkout = (workout: Workout) => {
      void navigate(
        `/plans/${workout.planId}?month=${startOfMonth(workout.date).slice(0, 7)}`
      );
    };
    content = (
      <div className="flex flex-col gap-3">
        {monthRaces.length > 0 && (
          <ul className="flex flex-wrap gap-2">
            {monthRaces.map((race) => (
              <li key={race.id}>
                <Link
                  to="/races"
                  className="inline-flex items-center gap-1 rounded-md bg-yellow-100 px-2 py-1 text-sm font-medium text-yellow-900 ring-1 ring-yellow-300 hover:bg-yellow-200"
                >
                  <TrophyIcon className="size-4" />
                  {race.name}
                </Link>
              </li>
            ))}
          </ul>
        )}
        {weeksBetween(viewStart, viewEnd).map((monday) => {
          const days = daysOfWeek(monday);
          let distance = 0;
          let elevation = 0;
          for (const day of days) {
            for (const activity of activitiesByDate.get(day) ?? []) {
              if (FOOT_SPORT_TYPES.includes(activity.sportType)) {
                distance += activity.distanceMeters;
                elevation += activity.elevationGainMeters;
              }
            }
          }
          return (
            <section key={monday} className="flex flex-col gap-1">
              {distance > 0 && (
                <p className="px-1 text-xs text-gray-500 tabular-nums">
                  Week of {formatShortDate(monday)}: {formatKm(distance)}
                  {elevation > 0 &&
                    ` · ${Math.round(elevation).toLocaleString()} m`}{" "}
                  on foot
                </p>
              )}
              <WeekDays
                days={days}
                workoutsByDate={workoutsByDate}
                activitiesByDate={activitiesByDate}
                goalRacesByDate={racesByDate}
                rangeStart={viewStart}
                rangeEnd={viewEnd}
                monthStart={month}
                monthEnd={monthEnd}
                onEditWorkout={openWorkout}
                hideEmptyDays
              />
            </section>
          );
        })}
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Calendar"
        subtitle="The races you're running, your planned workouts and your activities."
        actions={
          <Link to="/races" className={secondaryButton}>
            <TrophyIcon className="size-4" />
            Races
          </Link>
        }
      />
      <section className={`${cardClass} flex flex-col gap-3 p-3 sm:p-4`}>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-1">
            <button
              type="button"
              className={iconButton}
              aria-label="Previous month"
              onClick={() => {
                selectMonth(addMonths(month, -1));
              }}
            >
              <ChevronLeftIcon className="size-5" />
            </button>
            <h2 className="min-w-40 text-center text-lg font-semibold">
              {formatMonth(month)}
            </h2>
            <button
              type="button"
              className={iconButton}
              aria-label="Next month"
              onClick={() => {
                selectMonth(addMonths(month, 1));
              }}
            >
              <ChevronRightIcon className="size-5" />
            </button>
          </div>
          {month !== currentMonth() && (
            <button
              type="button"
              className={ghostButton}
              onClick={() => {
                selectMonth(currentMonth());
              }}
            >
              Today
            </button>
          )}
        </div>
        {content}
      </section>
    </div>
  );
};

export default CalendarPage;
