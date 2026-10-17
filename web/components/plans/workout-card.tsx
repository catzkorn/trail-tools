import { useAerobicThreshold } from "@components/lactate/use-aerobic-threshold";
import { CheckCircleIcon, MapIcon } from "@heroicons/react/24/solid";
import type { Workout, WorkoutWeather } from "gen/plans/v1/plans_pb";
import { WorkoutType } from "gen/plans/v1/plans_pb";
import { formatHoursMinutes } from "helpers/format";
import { workoutStyle } from "helpers/labels";
import React from "react";

import { WeatherBadge } from "./workout-weather";

interface WorkoutCardProps {
  workout: Workout;
  // The forecast for the workout's day, if there is one.
  weather?: WorkoutWeather | undefined;
  onClick?: (() => void) | undefined;
}

/**
 * Targets formats the workout's targets, with the aerobic threshold heart
 * rate as the cap of base workouts, if there is one.
 */
const targets = (workout: Workout, capBpm: number | undefined): string => {
  const parts: string[] = [];
  if (workout.targetDistanceMeters > 0) {
    parts.push(`${(workout.targetDistanceMeters / 1000).toFixed(1)} km`);
  }
  if (workout.targetElevationGainMeters > 0) {
    parts.push(`${Math.round(workout.targetElevationGainMeters)} m`);
  }
  if (workout.targetDurationSeconds > 0 && parts.length < 2) {
    parts.push(formatHoursMinutes(workout.targetDurationSeconds));
  }
  if (workout.type === WorkoutType.BASE && capBpm !== undefined) {
    parts.push(`<${capBpm} bpm`);
  }
  return parts.join(" · ");
};

// WorkoutCard is a compact, color coded workout in a calendar day.
const WorkoutCard: React.FC<WorkoutCardProps> = ({
  workout,
  weather,
  onClick,
}) => {
  const style = workoutStyle(workout.type);
  const threshold = useAerobicThreshold();
  const detail = targets(workout, threshold?.heartRateBpm);
  const content = (
    <>
      <span className="flex items-start justify-between gap-1">
        <span className="line-clamp-2 font-medium">
          {workout.title === "" ? style.label : workout.title}
        </span>
        <span className="flex shrink-0 items-center gap-0.5">
          {workout.routeId !== "" && (
            <MapIcon className="size-3.5 opacity-70" aria-label="Has route" />
          )}
          {workout.completedActivityId !== "" && (
            <CheckCircleIcon
              className="size-3.5 text-emerald-600"
              aria-label="Completed"
            />
          )}
        </span>
      </span>
      {detail !== "" && (
        <span className="tabular-nums opacity-80">{detail}</span>
      )}
      {weather?.day !== undefined && <WeatherBadge day={weather.day} />}
    </>
  );
  const className = `flex w-full flex-col rounded border-l-4 border px-1.5 py-1 text-left text-[11px] leading-tight ${style.cardClass}`;
  if (onClick === undefined) {
    return (
      <div className={className} title={workout.description}>
        {content}
      </div>
    );
  }
  return (
    <button
      type="button"
      className={`${className} hover:shadow`}
      title={workout.description}
      onClick={onClick}
    >
      {content}
    </button>
  );
};

export default WorkoutCard;
