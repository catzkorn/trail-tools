import type { ActivitySummary } from "gen/activities/v1/activities_pb";
import { formatHoursMinutes, formatShortDate } from "helpers/format";
import { useElementWidth } from "helpers/hooks";
import React, { useRef, useState } from "react";

interface VolumeChartProps {
  summaries: ActivitySummary[];
  height?: number;
}

const TICK_COUNT = 4;
const TICK_STEPS = [
  1, 2, 5, 10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000, 2500, 5000, 10_000,
];

/**
 * AxisMax returns the top of an axis with four whole number tick steps that
 * covers the value, and is at least the minimum.
 */
const axisMax = (value: number, minimum: number): number => {
  const target = Math.max(value, minimum);
  for (const step of TICK_STEPS) {
    if (step * TICK_COUNT >= target) {
      return step * TICK_COUNT;
    }
  }
  const largest = TICK_STEPS.at(-1) ?? 1;
  return Math.ceil(target / (largest * TICK_COUNT)) * largest * TICK_COUNT;
};

/**
 * VolumeChart draws weekly (or monthly) distance as bars with elevation gain
 * as a line on a secondary axis.
 */
const VolumeChart: React.FC<VolumeChartProps> = ({
  summaries,
  height = 220,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const width = useElementWidth(containerRef);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);

  const marginLeft = 40;
  const marginRight = 48;
  const marginTop = 12;
  const marginBottom = 24;
  const plotWidth = Math.max(width - marginLeft - marginRight, 10);
  const plotHeight = height - marginTop - marginBottom;

  let maxDistanceKm = 0;
  let maxElevation = 0;
  for (const summary of summaries) {
    maxDistanceKm = Math.max(maxDistanceKm, summary.distanceMeters / 1000);
    maxElevation = Math.max(maxElevation, summary.elevationGainMeters);
  }
  const distanceTop = axisMax(maxDistanceKm, 10);
  const elevationTop = axisMax(maxElevation, 100);
  const slot = plotWidth / Math.max(summaries.length, 1);
  const barWidth = Math.max(slot * 0.6, 2);
  const yDistance = (km: number): number =>
    marginTop + (1 - km / distanceTop) * plotHeight;
  const yElevation = (meters: number): number =>
    marginTop + (1 - meters / elevationTop) * plotHeight;
  const labelEvery = Math.max(1, Math.ceil(60 / slot));

  const elevationLine = summaries
    .map(
      (summary, index) =>
        `${(marginLeft + slot * (index + 0.5)).toFixed(1)},${yElevation(summary.elevationGainMeters).toFixed(1)}`
    )
    .join(" ");

  const hovered = hoverIndex === null ? undefined : summaries[hoverIndex];

  return (
    <div ref={containerRef} className="relative w-full select-none">
      {width > 0 && (
        <svg
          width={width}
          height={height}
          className="block"
          onPointerLeave={() => {
            setHoverIndex(null);
          }}
        >
          {[0, 0.25, 0.5, 0.75, 1].map((fraction) => (
            <g key={fraction}>
              <line
                x1={marginLeft}
                x2={marginLeft + plotWidth}
                y1={yDistance(distanceTop * fraction)}
                y2={yDistance(distanceTop * fraction)}
                stroke="#e5e7eb"
              />
              <text
                x={marginLeft - 6}
                y={yDistance(distanceTop * fraction)}
                textAnchor="end"
                dominantBaseline="middle"
                className="fill-emerald-700 text-[10px]"
              >
                {Math.round(distanceTop * fraction)}
              </text>
              <text
                x={marginLeft + plotWidth + 6}
                y={yElevation(elevationTop * fraction)}
                dominantBaseline="middle"
                className="fill-violet-700 text-[10px]"
              >
                {Math.round(elevationTop * fraction)}
              </text>
            </g>
          ))}
          <text x={4} y={8} className="fill-emerald-700 text-[10px]">
            km
          </text>
          <text
            x={width - 4}
            y={8}
            textAnchor="end"
            className="fill-violet-700 text-[10px]"
          >
            m gain
          </text>
          {summaries.map((summary, index) => {
            const km = summary.distanceMeters / 1000;
            const xCenter = marginLeft + slot * (index + 0.5);
            return (
              <g
                key={summary.startDate}
                onPointerEnter={() => {
                  setHoverIndex(index);
                }}
              >
                <rect
                  x={marginLeft + slot * index}
                  y={marginTop}
                  width={slot}
                  height={plotHeight}
                  fill={hoverIndex === index ? "#f3f4f6" : "transparent"}
                />
                <rect
                  x={xCenter - barWidth / 2}
                  y={yDistance(km)}
                  width={barWidth}
                  height={Math.max(marginTop + plotHeight - yDistance(km), 0)}
                  rx={2}
                  className="fill-emerald-500"
                />
                {index % labelEvery === 0 && (
                  <text
                    x={xCenter}
                    y={height - 6}
                    textAnchor="middle"
                    className="fill-gray-500 text-[10px]"
                  >
                    {formatShortDate(summary.startDate)}
                  </text>
                )}
              </g>
            );
          })}
          <polyline
            points={elevationLine}
            fill="none"
            stroke="#7c3aed"
            strokeWidth={2}
            pointerEvents="none"
          />
          {summaries.map((summary, index) => (
            <circle
              key={`dot-${summary.startDate}`}
              cx={marginLeft + slot * (index + 0.5)}
              cy={yElevation(summary.elevationGainMeters)}
              r={2.5}
              fill="#7c3aed"
              pointerEvents="none"
            />
          ))}
        </svg>
      )}
      {hovered !== undefined && hoverIndex !== null && (
        <div
          className="pointer-events-none absolute top-0 rounded bg-gray-900/90 px-2 py-1 text-xs whitespace-nowrap text-white tabular-nums"
          style={{
            left: Math.min(
              marginLeft + slot * hoverIndex,
              Math.max(width - 180, 0)
            ),
          }}
        >
          <div className="font-medium">
            {formatShortDate(hovered.startDate)} –{" "}
            {formatShortDate(hovered.endDate)}
          </div>
          <div>
            {(hovered.distanceMeters / 1000).toFixed(1)} km ·{" "}
            {Math.round(hovered.elevationGainMeters)} m ·{" "}
            {formatHoursMinutes(hovered.movingTimeSeconds)}
          </div>
          <div>
            {hovered.activityCount} activities · longest{" "}
            {(hovered.longestDistanceMeters / 1000).toFixed(1)} km
          </div>
        </div>
      )}
    </div>
  );
};

export default VolumeChart;
