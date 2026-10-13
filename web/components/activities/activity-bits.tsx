import type { Activity } from "gen/activities/v1/activities_pb";
import { formatHoursMinutes, formatKm } from "helpers/format";
import { sportLabel } from "helpers/labels";
import { decodePolyline } from "helpers/polyline";
import React from "react";
import { Link } from "react-router";

interface ActivityChipProps {
  activity: Activity;
}

// ActivityChip is a compact link to an activity, used in calendars.
export const ActivityChip: React.FC<ActivityChipProps> = ({ activity }) => (
  <Link
    to={`/activities/${activity.id}`}
    title={`${activity.name} (${sportLabel(activity.sportType)}, ${formatHoursMinutes(activity.movingTimeSeconds)})`}
    className="flex flex-col rounded border border-gray-300 bg-white px-1.5 py-0.5 text-[11px] leading-tight text-gray-700 hover:border-brand-500 hover:bg-brand-50"
  >
    <span className="truncate font-medium">{activity.name}</span>
    <span className="tabular-nums text-gray-500">
      {formatKm(activity.distanceMeters)}
      {activity.elevationGainMeters > 0 &&
        ` · ${Math.round(activity.elevationGainMeters)} m`}
    </span>
  </Link>
);

interface RouteThumbnailProps {
  polyline: string;
  size?: number;
}

// RouteThumbnail draws an encoded polyline as a small SVG.
export const RouteThumbnail: React.FC<RouteThumbnailProps> = ({
  polyline,
  size = 48,
}) => {
  const points = decodePolyline(polyline);
  if (points.length < 2) {
    return (
      <div
        className="rounded bg-gray-100"
        style={{ height: size, width: size }}
      />
    );
  }
  let minLat = Infinity;
  let maxLat = -Infinity;
  let minLng = Infinity;
  let maxLng = -Infinity;
  for (const [lat, lng] of points) {
    minLat = Math.min(minLat, lat);
    maxLat = Math.max(maxLat, lat);
    minLng = Math.min(minLng, lng);
    maxLng = Math.max(maxLng, lng);
  }
  /**
   * Scale longitude by the cosine of the latitude for a rough equal-area
   * projection.
   */
  const scaleLng = Math.cos((((minLat + maxLat) / 2) * Math.PI) / 180);
  const spanX = Math.max((maxLng - minLng) * scaleLng, 1e-6);
  const spanY = Math.max(maxLat - minLat, 1e-6);
  const span = Math.max(spanX, spanY);
  // Room for the start dot at the edges.
  const padding = 4;
  const inner = size - padding * 2;
  const offsetX = (inner - (spanX / span) * inner) / 2;
  const offsetY = (inner - (spanY / span) * inner) / 2;
  const coords = points.map(([lat, lng]): [number, number] => [
    padding + offsetX + (((lng - minLng) * scaleLng) / span) * inner,
    padding + offsetY + ((maxLat - lat) / span) * inner,
  ]);
  const path = coords
    .map(([xValue, yValue]) => `${xValue.toFixed(1)},${yValue.toFixed(1)}`)
    .join(" ");
  const [startX, startY] = coords[0] ?? [0, 0];
  return (
    <svg
      width={size}
      height={size}
      className="shrink-0 rounded bg-stone-100"
      aria-hidden="true"
    >
      <polyline
        points={path}
        fill="none"
        stroke="#292524"
        strokeWidth={1.5}
        strokeLinejoin="round"
      />
      <circle cx={startX} cy={startY} r={2.5} className="fill-ember" />
    </svg>
  );
};
