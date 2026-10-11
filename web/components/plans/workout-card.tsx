import { CheckCircleIcon, MapIcon } from "@heroicons/react/24/solid";
import type { Workout } from "gen/plans/v1/plans_pb";
import { formatHoursMinutes } from "helpers/format";
import { workoutStyle } from "helpers/labels";
import React from "react";

interface WorkoutCardProps {
  workout: Workout;
  onClick?: (() => void) | undefined;
}

const targets = (workout: Workout): string => {
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
  return parts.join(" · ");
};

// WorkoutCard is a compact, color coded workout in a calendar day.
const WorkoutCard: React.FC<WorkoutCardProps> = ({ workout, onClick }) => {
  const style = workoutStyle(workout.type);
  const detail = targets(workout);
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
