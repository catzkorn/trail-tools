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
  path: RoutePoint[],
  target: LatLngValue,
  from: number
): number => {
  let best = from;
  let bestDistance = Infinity;
  for (const [index, point] of path.entries()) {
    if (index < from) {
      continue;
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
 * InsertIndex returns where to insert a waypoint clicked on the route path:
 * between the two waypoints whose path section contains the clicked point.
 */
export const insertIndex = (
  path: RoutePoint[],
  waypoints: LatLngValue[],
  clicked: LatLngValue
): number => {
  if (path.length === 0 || waypoints.length < 2) {
    return waypoints.length;
  }
  const clickedIndex = nearestPathIndex(path, clicked, 0);
  let from = 0;
  for (let index = 0; index < waypoints.length - 1; index++) {
    const waypoint = waypoints[index];
    const nextWaypoint = waypoints[index + 1];
    if (waypoint === undefined || nextWaypoint === undefined) {
      break;
    }
    const start = nearestPathIndex(path, waypoint, from);
    const end = nearestPathIndex(path, nextWaypoint, start);
    if (clickedIndex >= start && clickedIndex <= end) {
      return index + 1;
    }
    from = start;
  }
  return waypoints.length - 1;
};
