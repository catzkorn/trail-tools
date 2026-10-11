import { PointOfInterestKind, Terrain } from "gen/geo/v1/geo_pb";
import { RoutingProfile, RouteSource } from "gen/routes/v1/routes_pb";

// Human readable labels and colors for API enums.

export interface TerrainStyle {
  terrain: Terrain;
  label: string;
  // A hex color, used on the map and in charts.
  color: string;
}

export const TERRAIN_STYLES: TerrainStyle[] = [
  { color: "#64748b", label: "Paved", terrain: Terrain.PAVED },
  { color: "#d97706", label: "Gravel", terrain: Terrain.GRAVEL },
  { color: "#059669", label: "Trail", terrain: Terrain.TRAIL },
  { color: "#7c3aed", label: "Technical", terrain: Terrain.TECHNICAL },
  { color: "#cbd5e1", label: "Unknown", terrain: Terrain.UNSPECIFIED },
];

export const terrainStyle = (terrain: Terrain): TerrainStyle => {
  for (const style of TERRAIN_STYLES) {
    if (style.terrain === terrain) {
      return style;
    }
  }
  return { color: "#cbd5e1", label: "Unknown", terrain };
};

export interface ProfileOption {
  value: RoutingProfile;
  label: string;
  description: string;
}

// PROFILE_OPTIONS are the route editor's routing modes.
export const PROFILE_OPTIONS: ProfileOption[] = [
  {
    description: "Snap to trails and paths, preferring unpaved ones",
    label: "Trail",
    value: RoutingProfile.TRAIL,
  },
  {
    description: "Snap to the shortest path on any way, including roads",
    label: "Road",
    value: RoutingProfile.SHORTEST,
  },
  {
    description: "Draw straight lines, for off-trail sections",
    label: "Manual",
    value: RoutingProfile.STRAIGHT,
  },
  {
    description:
      "Snap to trails, preferring the paths you have run before (your heatmap)",
    label: "My routes",
    value: RoutingProfile.HEATMAP,
  },
];

export const routeSourceLabel = (source: RouteSource): string => {
  switch (source) {
    case RouteSource.WAYPOINTS: {
      return "Designed";
    }
    case RouteSource.GPX: {
      return "GPX import";
    }
    case RouteSource.ACTIVITY: {
      return "From activity";
    }
    default: {
      return "";
    }
  }
};

export interface PoiStyle {
  kind: PointOfInterestKind;
  label: string;
  symbol: string;
  color: string;
}

export const POI_STYLES: PoiStyle[] = [
  {
    color: "#7c2d12",
    kind: PointOfInterestKind.PEAK,
    label: "Peak",
    symbol: "▲",
  },
  {
    color: "#0284c7",
    kind: PointOfInterestKind.WATER,
    label: "Water",
    symbol: "W",
  },
  {
    color: "#1d4ed8",
    kind: PointOfInterestKind.PARKING,
    label: "Parking",
    symbol: "P",
  },
  {
    color: "#0f766e",
    kind: PointOfInterestKind.VIEWPOINT,
    label: "Viewpoint",
    symbol: "V",
  },
  {
    color: "#92400e",
    kind: PointOfInterestKind.SHELTER,
    label: "Shelter",
    symbol: "H",
  },
  {
    color: "#475569",
    kind: PointOfInterestKind.TOILETS,
    label: "Toilets",
    symbol: "T",
  },
  {
    color: "#6d28d9",
    kind: PointOfInterestKind.SADDLE,
    label: "Saddle",
    symbol: "◆",
  },
  {
    color: "#15803d",
    kind: PointOfInterestKind.TRAILHEAD,
    label: "Trailhead",
    symbol: "S",
  },
];

export const poiStyle = (kind: PointOfInterestKind): PoiStyle => {
  for (const style of POI_STYLES) {
    if (style.kind === kind) {
      return style;
    }
  }
  return { color: "#475569", kind, label: "Point", symbol: "•" };
};

// Sport types counted as running volume (Strava sport types).
export const FOOT_SPORT_TYPES = ["Run", "TrailRun", "VirtualRun", "Hike"];

export const SPORT_TYPE_OPTIONS = [
  "Run",
  "TrailRun",
  "Hike",
  "Walk",
  "VirtualRun",
  "Ride",
  "GravelRide",
  "MountainBikeRide",
  "Swim",
  "NordicSki",
  "BackcountrySki",
  "WeightTraining",
  "Workout",
  "Yoga",
];

// SportLabel splits a Strava sport type into words, e.g. "Trail Run".
export const sportLabel = (sportType: string): string =>
  sportType.replaceAll(/([a-z])([A-Z])/gu, "$1 $2");
