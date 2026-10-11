import type { RoutePoint } from "gen/routes/v1/routes_pb";
import type { LatLngTuple } from "leaflet";

export interface LatLngValue {
  lat: number;
  lng: number;
}

export const toTuples = (points: LatLngValue[]): LatLngTuple[] =>
  points.map((point): LatLngTuple => [point.lat, point.lng]);

// SquaredDistance is a cheap distance measure for comparing nearby points.
const squaredDistance = (first: LatLngValue, second: LatLngValue): number => {
  const scale = Math.cos((first.lat * Math.PI) / 180);
  const dLat = first.lat - second.lat;
  const dLng = (first.lng - second.lng) * scale;
  return dLat * dLat + dLng * dLng;
};

const nearestPathIndex = (
  path: LatLngValue[],
  target: LatLngValue,
  from: number
): number => {
  let best = from;
  let bestDistance = Infinity;
  for (let index = from; index < path.length; index++) {
    const point = path[index];
    if (point === undefined) {
      break;
    }
    const distance = squaredDistance(point, target);
    if (distance < bestDistance) {
      bestDistance = distance;
      best = index;
    }
  }
  return best;
};

/**
 * WaypointPathIndexes returns the index of the path point at each waypoint.
 * Each leg starts and ends at its waypoints, so they are found in order along
 * the path.
 */
export const waypointPathIndexes = (
  path: LatLngValue[],
  waypoints: LatLngValue[]
): number[] => {
  const indexes: number[] = [];
  let from = 0;
  for (const [position, waypoint] of waypoints.entries()) {
    /**
     * The finish is the end of the path, even if it is also the start, as in
     * a loop.
     */
    const index =
      position === waypoints.length - 1
        ? path.length - 1
        : nearestPathIndex(path, waypoint, from);
    indexes.push(index);
    from = index;
  }
  return indexes;
};

export interface DistanceMarker {
  km: number;
  position: LatLngTuple;
}

// DistanceStepKm returns how often to mark the distance on a route.
export const distanceStepKm = (totalMeters: number): number => {
  if (totalMeters <= 30_000) {
    return 1;
  }
  if (totalMeters <= 100_000) {
    return 5;
  }
  return 10;
};

/**
 * DistanceMarkers returns the positions along a path at every stepKm
 * kilometers, interpolated between the path points.
 */
export const distanceMarkers = (
  path: RoutePoint[],
  stepKm: number
): DistanceMarker[] => {
  const markers: DistanceMarker[] = [];
  let km = stepKm;
  for (let index = 1; index < path.length; index++) {
    const previous = path[index - 1];
    const point = path[index];
    if (previous === undefined || point === undefined) {
      break;
    }
    while (point.distanceMeters >= km * 1000) {
      const span = point.distanceMeters - previous.distanceMeters;
      const fraction =
        span > 0 ? (km * 1000 - previous.distanceMeters) / span : 0;
      markers.push({
        km,
        position: [
          previous.lat + (point.lat - previous.lat) * fraction,
          previous.lng + (point.lng - previous.lng) * fraction,
        ],
      });
      km += stepKm;
    }
  }
  return markers;
};
