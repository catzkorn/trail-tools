import ElevationProfile from "@components/charts/elevation-profile";
import type { ProfileMarker } from "@components/charts/elevation-profile";
import { cardClass, tableClass, tdClass, thClass } from "@components/ui/styles";
import type {
  CheckpointAnalysis,
  RacePlan,
  RacePlanAnalysis,
} from "gen/races/v1/race_plans_pb";
import type { RoutePoint } from "gen/routes/v1/routes_pb";
import { formatDuration, formatPace } from "helpers/format";
import React from "react";

const bufferClass = (analysis: CheckpointAnalysis): string => {
  if (!analysis.hasCutoff) {
    return "text-gray-400";
  }
  if (analysis.cutoffBufferSeconds < 0) {
    return "bg-red-100 text-red-800 font-semibold";
  }
  if (analysis.cutoffBufferSeconds < 30 * 60) {
    return "bg-amber-100 text-amber-800 font-semibold";
  }
  return "bg-emerald-50 text-emerald-800";
};

interface FuelCellProps {
  planned: number;
  target: number;
}

// FuelCell shows planned intake against the target, highlighting shortfalls.
const FuelCell: React.FC<FuelCellProps> = ({ planned, target }) => {
  let className = "text-gray-800";
  if (target > 0 && planned < target * 0.8) {
    className = "text-amber-700 font-medium";
  }
  if (target > 0 && planned < target * 0.5) {
    className = "text-red-700 font-medium";
  }
  return (
    <td className={`${tdClass} text-right whitespace-nowrap tabular-nums`}>
      <span className={className}>{Math.round(planned)}</span>
      {target > 0 && (
        <span className="text-gray-400"> / {Math.round(target)}</span>
      )}
    </td>
  );
};

interface AnalysisTableProps {
  analysis: RacePlanAnalysis | undefined;
}

// AnalysisTable shows the computed timings and fuel for each segment.
export const AnalysisTable: React.FC<AnalysisTableProps> = ({ analysis }) => {
  if (analysis === undefined || analysis.checkpoints.length === 0) {
    return (
      <p className="text-sm text-gray-500 italic">
        Add checkpoints with planned segment times to see arrival times and
        cutoff buffers.
      </p>
    );
  }
  return (
    <div className="-mx-4 overflow-x-auto px-4">
      <table className={tableClass}>
        <thead>
          <tr>
            <th className={thClass}>Checkpoint</th>
            <th className={`${thClass} text-right`}>Seg km</th>
            <th className={`${thClass} text-right`}>+/− m</th>
            <th className={`${thClass} text-right`}>Pace</th>
            <th className={`${thClass} text-right`}>Arrive</th>
            <th className={`${thClass} text-right`}>Clock</th>
            <th className={`${thClass} text-right`}>Buffer</th>
            <th className={`${thClass} text-right`}>Carbs g</th>
            <th className={`${thClass} text-right`}>Fluid ml</th>
            <th className={`${thClass} text-right`}>Sodium mg</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {analysis.checkpoints.map((checkpoint) => (
            <tr key={`${checkpoint.arrivalSeconds}-${checkpoint.name}`}>
              <td className={`${tdClass} font-medium whitespace-nowrap`}>
                {checkpoint.name}
              </td>
              <td className={`${tdClass} text-right tabular-nums`}>
                {(checkpoint.segmentDistanceMeters / 1000).toFixed(1)}
              </td>
              <td
                className={`${tdClass} text-right whitespace-nowrap tabular-nums`}
              >
                +{Math.round(checkpoint.segmentElevationGainMeters)} / −
                {Math.round(checkpoint.segmentElevationLossMeters)}
              </td>
              <td
                className={`${tdClass} text-right whitespace-nowrap tabular-nums`}
              >
                {formatPace(checkpoint.segmentPaceSecondsPerKm)}
              </td>
              <td className={`${tdClass} text-right tabular-nums`}>
                {formatDuration(checkpoint.arrivalSeconds)}
              </td>
              <td
                className={`${tdClass} text-right whitespace-nowrap tabular-nums`}
              >
                {checkpoint.arrivalClockTime}
              </td>
              <td className={`${tdClass} text-right`}>
                <span
                  className={`rounded px-1.5 py-0.5 whitespace-nowrap tabular-nums ${bufferClass(checkpoint)}`}
                >
                  {checkpoint.hasCutoff
                    ? formatDuration(checkpoint.cutoffBufferSeconds)
                    : "–"}
                </span>
              </td>
              <FuelCell
                planned={checkpoint.carbsGrams}
                target={checkpoint.carbsTargetGrams}
              />
              <FuelCell
                planned={checkpoint.fluidMl}
                target={checkpoint.fluidTargetMl}
              />
              <FuelCell
                planned={checkpoint.sodiumMg}
                target={checkpoint.sodiumTargetMg}
              />
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr className="border-t-2 border-gray-200 font-semibold">
            <td className={tdClass}>Finish</td>
            <td className={tdClass} />
            <td className={tdClass} />
            <td className={tdClass} />
            <td className={`${tdClass} text-right tabular-nums`}>
              {formatDuration(analysis.plannedFinishSeconds)}
            </td>
            <td className={`${tdClass} text-right whitespace-nowrap`}>
              {analysis.plannedFinishClockTime}
            </td>
            <td className={tdClass} />
            <td className={`${tdClass} text-right tabular-nums`}>
              {Math.round(analysis.totalCarbsGrams)}
            </td>
            <td className={`${tdClass} text-right tabular-nums`}>
              {Math.round(analysis.totalFluidMl)}
            </td>
            <td className={`${tdClass} text-right tabular-nums`}>
              {Math.round(analysis.totalSodiumMg)}
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
  );
};

interface CourseCardProps {
  plan: RacePlan;
  analysis: RacePlanAnalysis | undefined;
  coursePoints: RoutePoint[];
}

/**
 * CourseCard shows the course profile with checkpoint markers and the
 * analysis table.
 */
const CourseCard: React.FC<CourseCardProps> = ({
  plan,
  analysis,
  coursePoints,
}) => {
  const markers: ProfileMarker[] = plan.checkpoints.map((checkpoint) => ({
    distanceMeters: checkpoint.distanceMeters,
    label: checkpoint.name,
  }));
  return (
    <section className={`${cardClass} break-inside-avoid-page`}>
      <div className="border-b border-gray-100 px-4 py-3">
        <h2 className="text-base font-semibold text-gray-900">
          Timings & fuel by segment
        </h2>
      </div>
      <div className="flex flex-col gap-4 p-4">
        {coursePoints.length > 1 && (
          <ElevationProfile points={coursePoints} markers={markers} />
        )}
        <AnalysisTable analysis={analysis} />
        <p className="text-xs text-gray-500">
          Buffer is the time between the planned arrival and the cutoff: red
          when the plan misses the cutoff, amber under 30 minutes. Fuel shows
          planned / target for each segment.
        </p>
      </div>
    </section>
  );
};

export default CourseCard;
