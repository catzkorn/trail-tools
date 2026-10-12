import type { RoutePoint } from "gen/routes/v1/routes_pb";
import type { TripPhoto } from "gen/trips/v1/trips_pb";
import { formatMeters } from "helpers/format";
import type { LatLngTuple } from "leaflet";

// DistanceRange is a stretch of a track, in meters along it.
export interface DistanceRange {
  startMeters: number;
  endMeters: number;
}

interface HasDistances {
  startDistanceMeters?: number | undefined;
  endDistanceMeters?: number | undefined;
}

// SectionRange returns the stretch of the track a section is about, if any.
export const sectionRange = (
  section: HasDistances
): DistanceRange | undefined => {
  if (
    section.startDistanceMeters === undefined ||
    section.endDistanceMeters === undefined
  ) {
    return undefined;
  }
  return {
    endMeters: section.endDistanceMeters,
    startMeters: section.startDistanceMeters,
  };
};

// TrackSlice returns the positions of the track within a stretch.
export const trackSlice = (
  track: RoutePoint[],
  range: DistanceRange
): LatLngTuple[] =>
  track
    .filter(
      (point) =>
        point.distanceMeters >= range.startMeters &&
        point.distanceMeters <= range.endMeters
    )
    .map((point) => [point.lat, point.lng]);

// ElevationThreshold filters noise when adding up climbing, as on the server.
const ELEVATION_THRESHOLD = 4;

/**
 * ClimbWithin returns the elevation gain and loss along a stretch of the
 * track, ignoring changes smaller than a few meters.
 */
export const climbWithin = (
  track: RoutePoint[],
  range: DistanceRange
): { gain: number; loss: number } => {
  let gain = 0;
  let loss = 0;
  let reference: number | undefined = undefined;
  for (const point of track) {
    if (
      point.distanceMeters < range.startMeters ||
      point.distanceMeters > range.endMeters
    ) {
      continue;
    }
    if (reference === undefined) {
      reference = point.elevationMeters;
      continue;
    }
    const difference = point.elevationMeters - reference;
    if (Math.abs(difference) >= ELEVATION_THRESHOLD) {
      if (difference > 0) {
        gain += difference;
      } else {
        loss -= difference;
      }
      reference = point.elevationMeters;
    }
  }
  return { gain, loss };
};

// FormatRange formats a stretch of the track, e.g. "km 3.2–9.8".
export const formatRange = (range: DistanceRange): string =>
  `km ${(range.startMeters / 1000).toFixed(1)}–${(range.endMeters / 1000).toFixed(1)}`;

/**
 * SectionSummary describes a stretch of the track, e.g. "km 3.2–9.8 ·
 * ↑ 640 m ↓ 120 m". The climbing is left out without elevations.
 */
export const sectionSummary = (
  track: RoutePoint[],
  range: DistanceRange
): string => {
  const parts = [formatRange(range)];
  if (track.some((point) => point.elevationMeters !== 0)) {
    const { gain, loss } = climbWithin(track, range);
    const climbing: string[] = [];
    if (gain >= 10) {
      climbing.push(`↑ ${formatMeters(gain)}`);
    }
    if (loss >= 10) {
      climbing.push(`↓ ${formatMeters(loss)}`);
    }
    if (climbing.length > 0) {
      parts.push(climbing.join(" "));
    }
  }
  return parts.join(" · ");
};

// Photos taken this close together along the track or in time are grouped.
const GROUP_METERS = 300;
const GROUP_SECONDS = 10 * 60;
const MAX_GROUP_SIZE = 4;

const takenSeconds = (photo: TripPhoto): number | undefined =>
  photo.takenTime === undefined ? undefined : Number(photo.takenTime.seconds);

// CloseTogether returns whether two photos were taken close together.
const closeTogether = (first: TripPhoto, second: TripPhoto): boolean => {
  if (
    first.trackDistanceMeters !== undefined &&
    second.trackDistanceMeters !== undefined
  ) {
    return (
      Math.abs(first.trackDistanceMeters - second.trackDistanceMeters) <=
      GROUP_METERS
    );
  }
  const firstTime = takenSeconds(first);
  const secondTime = takenSeconds(second);
  return (
    firstTime !== undefined &&
    secondTime !== undefined &&
    Math.abs(firstTime - secondTime) <= GROUP_SECONDS
  );
};

/**
 * PhotoGroups splits a section's photos into groups shown side by side:
 * runs of photos each taken close to the one before, at most four at a time.
 */
export const photoGroups = (photos: TripPhoto[]): TripPhoto[][] => {
  const groups: TripPhoto[][] = [];
  let current: TripPhoto[] = [];
  for (const photo of photos) {
    const previous = current.at(-1);
    if (
      previous === undefined ||
      current.length >= MAX_GROUP_SIZE ||
      !closeTogether(previous, photo)
    ) {
      current = [];
      groups.push(current);
    }
    current.push(photo);
  }
  return groups;
};
