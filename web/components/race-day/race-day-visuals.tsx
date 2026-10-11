import ElevationProfile from "@components/charts/elevation-profile";
import type { ProfileMarker } from "@components/charts/elevation-profile";
import { FitBounds, PointMarker, PolylineLayer } from "@components/map/layers";
import MapView from "@components/map/map-view";
import { toTuples } from "@components/routes/route-geometry";
import Card from "@components/ui/card";
import type { RaceDayStatus } from "gen/races/v1/race_plans_pb";
import type { RoutePoint } from "gen/routes/v1/routes_pb";
import type { LatLngTuple } from "leaflet";
import React, { useMemo } from "react";

import { formatClockTime } from "./race-day-utils";

// PointAt returns the course point closest to a distance, for checkpoint dots.
const pointAt = (
  course: RoutePoint[],
  distanceMeters: number
): RoutePoint | undefined => {
  let best: RoutePoint | undefined = undefined;
  let bestGap = Infinity;
  for (const point of course) {
    const gap = Math.abs(point.distanceMeters - distanceMeters);
    if (gap < bestGap) {
      bestGap = gap;
      best = point;
    }
  }
  return best;
};

interface RaceDayVisualsProps {
  status: RaceDayStatus;
  course: RoutePoint[];
  now: number;
}

// RaceDayVisuals shows the runner's position on the course profile and map.
const RaceDayVisuals: React.FC<RaceDayVisualsProps> = ({
  status,
  course,
  now,
}) => {
  const tuples = useMemo(() => toTuples(course), [course]);
  const started = status.startTime !== undefined;
  const markers: ProfileMarker[] = [];
  for (const estimate of status.checkpoints) {
    let label = estimate.name;
    if (estimate.arrivalTime !== undefined) {
      label = `${estimate.arrivalRecorded ? "" : "~"}${formatClockTime(estimate.arrivalTime, now)}`;
    }
    markers.push({ distanceMeters: estimate.distanceMeters, label });
  }

  const location = status.estimatedLocation;
  let position: LatLngTuple | null = null;
  if (location !== undefined) {
    position = [location.lat, location.lng];
  }

  if (course.length < 2 && position === null) {
    return null;
  }
  let fitPositions: LatLngTuple[] = tuples;
  if (tuples.length < 2 && position !== null) {
    fitPositions = [position];
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      {course.length > 1 && (
        <Card title="Course">
          <ElevationProfile
            points={course}
            markers={markers}
            positionMeters={
              started ? status.estimatedDistanceMeters : undefined
            }
            positionLabel="Runner"
            height={180}
          />
        </Card>
      )}
      <div className="overflow-hidden rounded-lg border border-gray-200">
        <MapView className="h-72 lg:h-full lg:min-h-72">
          {tuples.length > 1 && (
            <PolylineLayer positions={tuples} color="#047857" weight={4} />
          )}
          {status.checkpoints.map((estimate) => {
            const point = pointAt(course, estimate.distanceMeters);
            if (point === undefined) {
              return null;
            }
            return (
              <PointMarker
                key={estimate.checkpointIndex}
                position={[point.lat, point.lng]}
                color="#374151"
                radius={5}
                tooltip={estimate.name}
              />
            );
          })}
          {position !== null && (
            <PointMarker
              position={position}
              color="#10b981"
              radius={9}
              tooltip="Runner (estimated position)"
            />
          )}
          <FitBounds
            positions={fitPositions}
            fitKey={`course-${tuples.length}`}
          />
        </MapView>
      </div>
    </div>
  );
};

export default RaceDayVisuals;
