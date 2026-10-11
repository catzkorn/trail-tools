import type { RoutingProfile } from "gen/routes/v1/routes_pb";

import type { LatLngValue } from "./route-geometry";

/**
 * RouteShape is the editable geometry of a route: its waypoints and the
 * routing profile of each leg. legProfiles[i] is the profile of the leg from
 * waypoints[i] to waypoints[i+1], so there is one fewer than waypoints.
 */
export interface RouteShape {
  waypoints: LatLngValue[];
  legProfiles: RoutingProfile[];
}

export const EMPTY_SHAPE: RouteShape = { legProfiles: [], waypoints: [] };

const reversedCopy = <Item>(items: Item[]): Item[] => {
  const result: Item[] = [];
  for (const item of items) {
    result.unshift(item);
  }
  return result;
};

/**
 * NewShape creates a shape from waypoints and leg profiles, using the
 * default profile for legs without one.
 */
export const newShape = (
  waypoints: LatLngValue[],
  legProfiles: RoutingProfile[],
  defaultProfile: RoutingProfile
): RouteShape => {
  const profiles: RoutingProfile[] = [];
  for (let index = 0; index < waypoints.length - 1; index++) {
    profiles.push(legProfiles[index] ?? defaultProfile);
  }
  return { legProfiles: profiles, waypoints };
};

// AppendWaypoint adds a waypoint at the end, routed with the profile.
export const appendWaypoint = (
  shape: RouteShape,
  point: LatLngValue,
  profile: RoutingProfile
): RouteShape => ({
  legProfiles:
    shape.waypoints.length === 0 ? [] : [...shape.legProfiles, profile],
  waypoints: [...shape.waypoints, point],
});

/**
 * InsertWaypoint inserts a waypoint before the waypoint at index, splitting
 * the leg it is on into two legs routed with the profile.
 */
export const insertWaypoint = (
  shape: RouteShape,
  index: number,
  point: LatLngValue,
  profile: RoutingProfile
): RouteShape => {
  if (index <= 0 || index >= shape.waypoints.length) {
    return appendWaypoint(shape, point, profile);
  }
  const waypoints = [...shape.waypoints];
  waypoints.splice(index, 0, point);
  const legProfiles = [...shape.legProfiles];
  legProfiles.splice(index - 1, 1, profile, profile);
  return { legProfiles, waypoints };
};

/**
 * MoveWaypoint moves the waypoint at index. The legs to and from it are
 * routed again with the profile.
 */
export const moveWaypoint = (
  shape: RouteShape,
  index: number,
  point: LatLngValue,
  profile: RoutingProfile
): RouteShape => {
  const waypoints = [...shape.waypoints];
  waypoints[index] = point;
  const legProfiles = [...shape.legProfiles];
  if (index > 0) {
    legProfiles[index - 1] = profile;
  }
  if (index < legProfiles.length) {
    legProfiles[index] = profile;
  }
  return { legProfiles, waypoints };
};

/**
 * RemoveWaypoint removes the waypoint at index. The legs to and from a middle
 * waypoint are replaced by a single leg routed with the profile.
 */
export const removeWaypoint = (
  shape: RouteShape,
  index: number,
  profile: RoutingProfile
): RouteShape => {
  const waypoints = shape.waypoints.filter(
    (_point, position) => position !== index
  );
  const legProfiles = [...shape.legProfiles];
  if (index === 0) {
    legProfiles.splice(0, 1);
  } else if (index === shape.waypoints.length - 1) {
    legProfiles.splice(index - 1, 1);
  } else {
    legProfiles.splice(index - 1, 2, profile);
  }
  return { legProfiles, waypoints };
};

// ReverseRoute swaps the start and finish.
export const reverseRoute = (shape: RouteShape): RouteShape => ({
  legProfiles: reversedCopy(shape.legProfiles),
  waypoints: reversedCopy(shape.waypoints),
});

const samePoint = (first: LatLngValue, second: LatLngValue): boolean =>
  first.lat === second.lat && first.lng === second.lng;

// IsLoop reports whether the route finishes where it starts.
export const isLoop = (shape: RouteShape): boolean => {
  const [start] = shape.waypoints;
  const finish = shape.waypoints.at(-1);
  return (
    shape.waypoints.length > 2 &&
    start !== undefined &&
    finish !== undefined &&
    samePoint(start, finish)
  );
};

/**
 * ReturnToStart adds a leg from the finish back to the start, routed with
 * the profile, to close the loop.
 */
export const returnToStart = (
  shape: RouteShape,
  profile: RoutingProfile
): RouteShape => {
  const [start] = shape.waypoints;
  if (start === undefined || shape.waypoints.length < 2 || isLoop(shape)) {
    return shape;
  }
  return appendWaypoint(shape, start, profile);
};

/**
 * OutAndBack mirrors the route, so that it returns to the start the same way
 * it went out.
 */
export const outAndBack = (shape: RouteShape): RouteShape => {
  if (shape.waypoints.length < 2) {
    return shape;
  }
  const back = reversedCopy(shape.waypoints.slice(0, -1));
  return {
    legProfiles: [...shape.legProfiles, ...reversedCopy(shape.legProfiles)],
    waypoints: [...shape.waypoints, ...back],
  };
};

const MAX_HISTORY = 100;

/**
 * EditHistory is the current shape with the shapes before it, for undo, and
 * the shapes undone, for redo.
 */
export interface EditHistory {
  past: RouteShape[];
  present: RouteShape;
  future: RouteShape[];
}

export const newHistory = (present: RouteShape): EditHistory => ({
  future: [],
  past: [],
  present,
});

// Push records a new shape, clearing the shapes that can be redone.
export const push = (history: EditHistory, next: RouteShape): EditHistory => {
  if (next === history.present) {
    return history;
  }
  return {
    future: [],
    past: [...history.past.slice(-(MAX_HISTORY - 1)), history.present],
    present: next,
  };
};

export const undo = (history: EditHistory): EditHistory => {
  const previous = history.past.at(-1);
  if (previous === undefined) {
    return history;
  }
  return {
    future: [history.present, ...history.future],
    past: history.past.slice(0, -1),
    present: previous,
  };
};

export const redo = (history: EditHistory): EditHistory => {
  const [next, ...rest] = history.future;
  if (next === undefined) {
    return history;
  }
  return {
    future: rest,
    past: [...history.past, history.present],
    present: next,
  };
};
