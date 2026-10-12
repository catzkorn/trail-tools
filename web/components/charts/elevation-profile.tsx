import { useElementWidth } from "helpers/hooks";
import type { PointerEvent } from "react";
import React, { useMemo, useRef, useState } from "react";

export interface ProfilePoint {
  distanceMeters: number;
  elevationMeters: number;
}

export interface ProfileMarker {
  distanceMeters: number;
  label: string;
}

interface ElevationProfileProps {
  points: ProfilePoint[];
  // An optional second series with one value per point, e.g. heart rate.
  // Zero values are treated as missing.
  secondary?: number[];
  secondaryLabel?: string;
  markers?: ProfileMarker[];
  /**
   * An optional reference line in meters of elevation, e.g. the freezing
   * level along the route. Parts far above the profile are not drawn.
   */
  levelLine?: ProfilePoint[];
  levelLabel?: string;
  // An optional highlighted position in meters, e.g. the runner's position.
  positionMeters?: number | undefined;
  positionLabel?: string;
  height?: number;
  // Called with the index of the hovered point, or null when not hovering.
  onHover?: (index: number | null) => void;
}

const NO_MARKERS: ProfileMarker[] = [];
const NO_LEVELS: ProfilePoint[] = [];

const NO_POINT: ProfilePoint = { distanceMeters: 0, elevationMeters: 0 };

// PointAt returns the point at an index that callers keep within range.
const pointAt = (points: ProfilePoint[], index: number): ProfilePoint =>
  points[index] ?? NO_POINT;

/**
 * NearestIndex finds the index of the point closest to the distance using a
 * binary search. Points must be sorted by distance.
 */
const nearestIndex = (points: ProfilePoint[], distance: number): number => {
  let low = 0;
  let high = points.length - 1;
  while (high - low > 1) {
    const middle = Math.floor((low + high) / 2);
    if (pointAt(points, middle).distanceMeters < distance) {
      low = middle;
    } else {
      high = middle;
    }
  }
  const lowGap = Math.abs(pointAt(points, low).distanceMeters - distance);
  const highGap = Math.abs(pointAt(points, high).distanceMeters - distance);
  return lowGap <= highGap ? low : high;
};

// ElevationAt linearly interpolates the elevation at a distance.
const elevationAt = (points: ProfilePoint[], distance: number): number => {
  const index = nearestIndex(points, distance);
  const point = pointAt(points, index);
  let other = index + 1;
  if (point.distanceMeters > distance) {
    other = index - 1;
  }
  if (other < 0 || other >= points.length) {
    return point.elevationMeters;
  }
  const neighbor = pointAt(points, other);
  const span = neighbor.distanceMeters - point.distanceMeters;
  if (span === 0) {
    return point.elevationMeters;
  }
  const fraction = (distance - point.distanceMeters) / span;
  return (
    point.elevationMeters +
    fraction * (neighbor.elevationMeters - point.elevationMeters)
  );
};

/**
 * GradeAt returns the grade in percent around a distance, measured over a
 * window of at least 100 meters to smooth out noise.
 */
const gradeAt = (
  points: ProfilePoint[],
  distance: number,
  window: number
): number => {
  const half = Math.max(window, 100) / 2;
  const before = elevationAt(points, distance - half);
  const after = elevationAt(points, distance + half);
  return ((after - before) / (half * 2)) * 100;
};

const SMOOTHING_HALF_WINDOW_METERS = 50;

/**
 * SmoothProfile returns the profile with each elevation replaced by the
 * average elevation within 50 meters, to remove elevation model noise.
 */
const smoothProfile = (points: ProfilePoint[]): ProfilePoint[] => {
  const result: ProfilePoint[] = [];
  let low = 0;
  let high = 0;
  let sum = 0;
  for (const point of points) {
    const center = point.distanceMeters;
    while (
      high < points.length &&
      pointAt(points, high).distanceMeters <=
        center + SMOOTHING_HALF_WINDOW_METERS
    ) {
      sum += pointAt(points, high).elevationMeters;
      high++;
    }
    while (
      low < high &&
      pointAt(points, low).distanceMeters <
        center - SMOOTHING_HALF_WINDOW_METERS
    ) {
      sum -= pointAt(points, low).elevationMeters;
      low++;
    }
    result.push({
      distanceMeters: center,
      elevationMeters: high > low ? sum / (high - low) : point.elevationMeters,
    });
  }
  return result;
};

const ELEVATION_STEPS = [10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000];
const DISTANCE_STEPS_KM = [0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 25, 50, 100];

const chooseStep = (range: number, maxTicks: number, steps: number[]) => {
  for (const step of steps) {
    if (range / step <= maxTicks) {
      return step;
    }
  }
  return steps.at(-1) ?? 1;
};

const ticksFor = (min: number, max: number, step: number): number[] => {
  const ticks: number[] = [];
  for (let tick = Math.ceil(min / step) * step; tick <= max; tick += step) {
    ticks.push(Math.round(tick * 1000) / 1000);
  }
  return ticks;
};

/**
 * ElevationProfile draws an elevation profile as an SVG area chart, with a
 * hover crosshair.
 */
const ElevationProfile: React.FC<ElevationProfileProps> = ({
  points,
  secondary,
  secondaryLabel = "",
  markers = NO_MARKERS,
  levelLine = NO_LEVELS,
  levelLabel = "",
  positionMeters,
  positionLabel = "",
  height = 200,
  onHover,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const width = useElementWidth(containerRef);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const smoothed = useMemo(() => smoothProfile(points), [points]);

  const [first] = points;
  const last = points.at(-1);
  if (points.length < 2 || first === undefined || last === undefined) {
    /**
     * The measured container is rendered without data too, so that its width
     * is known when data arrives, e.g. while drawing a new route.
     */
    return (
      <div ref={containerRef} className="relative w-full select-none">
        <div className="flex h-24 items-center justify-center text-sm text-gray-500">
          No elevation data
        </div>
      </div>
    );
  }

  const hasSecondary =
    secondary !== undefined &&
    secondary.length === points.length &&
    secondary.some((value) => value > 0);
  const marginLeft = 44;
  const marginRight = hasSecondary ? 40 : 12;
  const marginTop = markers.length > 0 ? 30 : 10;
  const marginBottom = 22;
  const plotWidth = Math.max(width - marginLeft - marginRight, 10);
  const plotHeight = height - marginTop - marginBottom;

  const minDistance = first.distanceMeters;
  const maxDistance = Math.max(last.distanceMeters, minDistance + 1);
  let minElevation = Infinity;
  let maxElevation = -Infinity;
  for (const point of points) {
    minElevation = Math.min(minElevation, point.elevationMeters);
    maxElevation = Math.max(maxElevation, point.elevationMeters);
  }
  /**
   * Include the level line in the vertical range when it is reasonably close
   * to the profile, so that it is visible without flattening the profile.
   */
  const profileRange = Math.max(maxElevation - minElevation, 50);
  for (const level of levelLine) {
    if (level.elevationMeters < maxElevation + profileRange) {
      minElevation = Math.min(minElevation, level.elevationMeters);
      maxElevation = Math.max(maxElevation, level.elevationMeters);
    }
  }
  const elevationRange = Math.max(maxElevation - minElevation, 50);
  const elevationStep = chooseStep(elevationRange * 1.2, 4, ELEVATION_STEPS);
  const yMin =
    Math.floor((minElevation - elevationRange * 0.05) / elevationStep) *
    elevationStep;
  const yMax =
    Math.ceil((maxElevation + elevationRange * 0.05) / elevationStep) *
    elevationStep;

  const xFor = (distance: number): number =>
    marginLeft +
    ((distance - minDistance) / (maxDistance - minDistance)) * plotWidth;
  const yFor = (elevation: number): number =>
    marginTop + (1 - (elevation - yMin) / (yMax - yMin)) * plotHeight;
  const baseline = marginTop + plotHeight;

  // The outline and the area use at most about one point per pixel.
  const stride = Math.max(1, Math.floor(points.length / plotWidth));

  const pointString = (distance: number, elevation: number): string =>
    `${xFor(distance).toFixed(1)},${yFor(elevation).toFixed(1)}`;

  const outline: string[] = [];
  for (let index = 0; index < points.length; index += stride) {
    const point = pointAt(points, index);
    outline.push(pointString(point.distanceMeters, point.elevationMeters));
  }
  outline.push(pointString(last.distanceMeters, last.elevationMeters));
  const area = `M${xFor(minDistance)},${baseline} L${outline.join(" L")} L${xFor(last.distanceMeters)},${baseline} Z`;

  let secondaryPath = "";
  let secondaryMin = 0;
  let secondaryMax = 0;
  if (hasSecondary) {
    secondaryMin = Infinity;
    secondaryMax = -Infinity;
    for (const value of secondary) {
      if (value > 0) {
        secondaryMin = Math.min(secondaryMin, value);
        secondaryMax = Math.max(secondaryMax, value);
      }
    }
    const span = Math.max(secondaryMax - secondaryMin, 10);
    const commands: string[] = [];
    let drawing = false;
    for (let index = 0; index < points.length; index += stride) {
      const value = secondary[index] ?? 0;
      if (value <= 0) {
        drawing = false;
        continue;
      }
      const xValue = xFor(pointAt(points, index).distanceMeters).toFixed(1);
      const yValue = (
        marginTop +
        (1 - (value - secondaryMin) / span) * plotHeight
      ).toFixed(1);
      commands.push(`${drawing ? "L" : "M"}${xValue},${yValue}`);
      drawing = true;
    }
    secondaryPath = commands.join(" ");
  }

  const levelCommands: string[] = [];
  let levelDrawing = false;
  for (const level of levelLine) {
    if (level.elevationMeters < yMin || level.elevationMeters > yMax) {
      levelDrawing = false;
      continue;
    }
    levelCommands.push(
      `${levelDrawing ? "L" : "M"}${xFor(level.distanceMeters).toFixed(1)},${yFor(level.elevationMeters).toFixed(1)}`
    );
    levelDrawing = true;
  }
  const levelPath = levelCommands.join(" ");

  const distanceRangeKm = (maxDistance - minDistance) / 1000;
  const distanceStep = chooseStep(
    distanceRangeKm,
    Math.max(Math.floor(plotWidth / 70), 2),
    DISTANCE_STEPS_KM
  );
  const distanceTicks = ticksFor(
    minDistance / 1000,
    maxDistance / 1000,
    distanceStep
  );
  const elevationTicks = ticksFor(yMin, yMax, elevationStep);

  const handlePointer = (event: PointerEvent<SVGSVGElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    const xPosition = event.clientX - rect.left;
    if (xPosition < marginLeft || xPosition > marginLeft + plotWidth) {
      setHoverIndex(null);
      onHover?.(null);
      return;
    }
    const distance =
      minDistance +
      ((xPosition - marginLeft) / plotWidth) * (maxDistance - minDistance);
    const index = nearestIndex(points, distance);
    setHoverIndex(index);
    onHover?.(index);
  };

  const hovered = hoverIndex === null ? undefined : points.at(hoverIndex);
  const hoveredSecondary =
    hoverIndex !== null && hasSecondary ? (secondary[hoverIndex] ?? 0) : 0;

  return (
    <div ref={containerRef} className="relative w-full select-none">
      {width > 0 && (
        <svg
          width={width}
          height={height}
          className="block touch-pan-y"
          onPointerMove={handlePointer}
          onPointerDown={handlePointer}
          onPointerLeave={() => {
            setHoverIndex(null);
            onHover?.(null);
          }}
        >
          {elevationTicks.map((tick) => (
            <g key={`y-${tick}`}>
              <line
                x1={marginLeft}
                x2={marginLeft + plotWidth}
                y1={yFor(tick)}
                y2={yFor(tick)}
                stroke="#e5e7eb"
              />
              <text
                x={marginLeft - 6}
                y={yFor(tick)}
                textAnchor="end"
                dominantBaseline="middle"
                className="fill-gray-500 text-[10px]"
              >
                {tick}
              </text>
            </g>
          ))}
          <path d={area} fill="#e2e8f0" stroke="none" />
          <polyline
            points={outline.join(" ")}
            fill="none"
            stroke="#475569"
            strokeWidth={1.5}
            strokeLinejoin="round"
          />
          {secondaryPath !== "" && (
            <path
              d={secondaryPath}
              fill="none"
              stroke="#dc2626"
              strokeWidth={1}
              opacity={0.8}
            />
          )}
          {levelPath !== "" && (
            <path
              d={levelPath}
              fill="none"
              stroke="#2563eb"
              strokeDasharray="5 3"
              strokeWidth={1.5}
            />
          )}
          {hasSecondary && (
            <>
              <text
                x={marginLeft + plotWidth + 4}
                y={marginTop}
                dominantBaseline="hanging"
                className="fill-red-600 text-[10px]"
              >
                {Math.round(secondaryMax)}
              </text>
              <text
                x={marginLeft + plotWidth + 4}
                y={marginTop + plotHeight}
                className="fill-red-600 text-[10px]"
              >
                {Math.round(secondaryMin)}
              </text>
            </>
          )}
          {distanceTicks.map((tick) => (
            <text
              key={`x-${tick}`}
              x={xFor(tick * 1000)}
              y={height - 6}
              textAnchor="middle"
              className="fill-gray-500 text-[10px]"
            >
              {tick} km
            </text>
          ))}
          {markers.map((marker, index) => {
            const xMarker = xFor(marker.distanceMeters);
            return (
              <g key={`marker-${marker.distanceMeters}-${marker.label}`}>
                <line
                  x1={xMarker}
                  x2={xMarker}
                  y1={marginTop - 4}
                  y2={baseline}
                  stroke="#1f2937"
                  strokeDasharray="3 3"
                  strokeWidth={1}
                />
                <text
                  x={xMarker}
                  y={index % 2 === 0 ? 10 : 21}
                  textAnchor="middle"
                  className="fill-gray-700 text-[10px] font-medium"
                >
                  {marker.label.length > 14
                    ? `${marker.label.slice(0, 13)}…`
                    : marker.label}
                </text>
              </g>
            );
          })}
          {positionMeters !== undefined &&
            positionMeters >= minDistance &&
            positionMeters <= maxDistance && (
              <g pointerEvents="none">
                <line
                  x1={xFor(positionMeters)}
                  x2={xFor(positionMeters)}
                  y1={marginTop}
                  y2={baseline}
                  stroke="#047857"
                  strokeWidth={2}
                />
                <circle
                  cx={xFor(positionMeters)}
                  cy={yFor(elevationAt(points, positionMeters))}
                  r={6}
                  fill="#10b981"
                  stroke="#ffffff"
                  strokeWidth={2}
                />
                {positionLabel !== "" && (
                  <text
                    x={xFor(positionMeters) + 8}
                    y={yFor(elevationAt(points, positionMeters)) - 8}
                    className="fill-emerald-800 text-[11px] font-semibold"
                  >
                    {positionLabel}
                  </text>
                )}
              </g>
            )}
          {hovered !== undefined && (
            <g pointerEvents="none">
              <line
                x1={xFor(hovered.distanceMeters)}
                x2={xFor(hovered.distanceMeters)}
                y1={marginTop}
                y2={baseline}
                stroke="#111827"
                strokeWidth={1}
              />
              <circle
                cx={xFor(hovered.distanceMeters)}
                cy={yFor(hovered.elevationMeters)}
                r={4}
                fill="#facc15"
                stroke="#111827"
              />
            </g>
          )}
        </svg>
      )}
      {hovered !== undefined && hoverIndex !== null && (
        <div
          className="pointer-events-none absolute top-1 rounded bg-gray-900/90 px-2 py-1 text-xs whitespace-nowrap text-white tabular-nums"
          style={{
            left: Math.min(
              Math.max(xFor(hovered.distanceMeters) + 8, 0),
              Math.max(width - 150, 0)
            ),
          }}
        >
          {(hovered.distanceMeters / 1000).toFixed(2)} km ·{" "}
          {Math.round(hovered.elevationMeters)} m ·{" "}
          {gradeAt(smoothed, hovered.distanceMeters, 100).toFixed(1)}%
          {hoveredSecondary > 0 &&
            ` · ${Math.round(hoveredSecondary)} ${secondaryLabel}`}
        </div>
      )}
      {(levelLine.length > 0 || hasSecondary) && (
        <div className="mt-1 flex flex-wrap items-center gap-3 text-[11px] text-gray-500">
          {levelLine.length > 0 && (
            <span className="inline-flex items-center gap-1">
              <span className="inline-block w-3 border-t-2 border-dashed border-blue-600" />
              {levelPath === "" ? `${levelLabel} (above chart)` : levelLabel}
            </span>
          )}
          {hasSecondary && (
            <span className="inline-flex items-center gap-1">
              <span className="inline-block h-0.5 w-3 bg-red-600" />
              {secondaryLabel === "" ? "Secondary" : secondaryLabel}
            </span>
          )}
        </div>
      )}
    </div>
  );
};

export default ElevationProfile;
