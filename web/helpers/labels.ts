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
