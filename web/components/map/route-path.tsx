import type { RoutePoint } from "gen/routes/v1/routes_pb";
import { RoutingProfile } from "gen/routes/v1/routes_pb";
import type {
  LatLng,
  LatLngTuple,
  LeafletMouseEvent,
  Point as LeafletPoint,
} from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useEffectEvent, useMemo } from "react";

import { useMap } from "./map-view";

export interface PathWaypoint {
  lat: number;
  lng: number;
}

interface LegSection {
  positions: LatLngTuple[];
  straight: boolean;
}

/**
 * HoverDistancePixels is how close the mouse must be to the path to show the
 * insert handle.
 */
const HOVER_DISTANCE_PIXELS = 14;
const PATH_COLOR = "#047857";

const squaredDistance = (first: LatLngTuple, second: LatLng): number => {
  const scale = Math.cos((first[0] * Math.PI) / 180);
  const dLat = first[0] - second.lat;
  const dLng = (first[1] - second.lng) * scale;
  return dLat * dLat + dLng * dLng;
};

const nearestIndex = (positions: LatLngTuple[], target: LatLng): number => {
  let best = 0;
  let bestDistance = Infinity;
  for (const [index, position] of positions.entries()) {
    const distance = squaredDistance(position, target);
    if (distance < bestDistance) {
      bestDistance = distance;
      best = index;
    }
  }
  return best;
};

interface RoutePathProps {
  points: RoutePoint[];
  /**
   * The path index of each waypoint the path was computed for. The path
   * between consecutive waypoints is a leg.
   */
  waypointIndexes: number[];
  // The routing profile of each leg. Straight legs are drawn dashed.
  legProfiles: RoutingProfile[];
  // The current waypoints, for rubber banding while inserting one.
  waypoints: PathWaypoint[];
  /**
   * Called to insert a waypoint before the waypoint at index, when the path
   * is clicked or dragged. Without it, the path is not editable.
   */
  onInsert?: ((index: number, position: LatLng) => void) | undefined;
  // Called with the index of the path point under the mouse, or null.
  onHover?: ((index: number | null) => void) | undefined;
}

/**
 * InsertIndex returns the index of the waypoint to insert before, to add a
 * waypoint at the path point at pathIndex: the end of the leg containing it.
 */
const insertIndex = (pathIndex: number, waypointIndexes: number[]): number => {
  for (let leg = 0; leg < waypointIndexes.length - 1; leg++) {
    const end = waypointIndexes[leg + 1] ?? 0;
    if (pathIndex <= end) {
      return leg + 1;
    }
  }
  return Math.max(waypointIndexes.length - 1, 1);
};

/**
 * RoutePath draws a route's path, with dashed straight legs. When editable,
 * hovering the path shows a handle that can be clicked to insert a waypoint,
 * or dragged to insert and move one.
 */
const RoutePath: React.FC<RoutePathProps> = ({
  points,
  waypointIndexes,
  legProfiles,
  waypoints,
  onInsert,
  onHover,
}) => {
  const map = useMap();
  const editable = onInsert !== undefined;
  const hoverable = editable || onHover !== undefined;
  const positions = useMemo(
    () => points.map((point): LatLngTuple => [point.lat, point.lng]),
    [points]
  );
  const sections = useMemo(() => {
    const result: LegSection[] = [];
    for (let leg = 0; leg < waypointIndexes.length - 1; leg++) {
      const start = waypointIndexes[leg] ?? 0;
      const end = waypointIndexes[leg + 1] ?? start;
      result.push({
        positions: positions.slice(start, end + 1),
        straight: legProfiles[leg] === RoutingProfile.STRAIGHT,
      });
    }
    if (result.length === 0) {
      result.push({ positions, straight: false });
    }
    return result;
  }, [positions, waypointIndexes, legProfiles]);

  // The visible path: a white casing with a line per leg on top.
  useEffect(() => {
    if (positions.length < 2) {
      return undefined;
    }
    const group = leaflet.layerGroup();
    group.addLayer(
      leaflet.polyline(positions, {
        color: "#ffffff",
        interactive: false,
        opacity: 0.8,
        weight: 8,
      })
    );
    for (const section of sections) {
      group.addLayer(
        leaflet.polyline(section.positions, {
          color: PATH_COLOR,
          dashArray: section.straight ? "8 8" : undefined,
          interactive: false,
          opacity: 0.95,
          weight: 5,
        })
      );
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, positions, sections]);

  const handleInsert = useEffectEvent((pathIndex: number, position: LatLng) => {
    onInsert?.(insertIndex(pathIndex, waypointIndexes), position);
  });
  const handleHover = useEffectEvent((index: number | null) => {
    onHover?.(index);
  });
  const neighbors = useEffectEvent((pathIndex: number): LatLngTuple[] => {
    const index = insertIndex(pathIndex, waypointIndexes);
    const result: LatLngTuple[] = [];
    const before = waypoints[index - 1];
    const after = waypoints[index];
    if (before !== undefined) {
      result.push([before.lat, before.lng]);
    }
    if (after !== undefined) {
      result.push([after.lat, after.lng]);
    }
    return result;
  });
  const nearWaypoint = useEffectEvent((layerPoint: LeafletPoint): boolean =>
    waypoints.some(
      (waypoint) =>
        map
          .latLngToLayerPoint([waypoint.lat, waypoint.lng])
          .distanceTo(layerPoint) < HOVER_DISTANCE_PIXELS
    )
  );

  /**
   * The interactive parts: an invisible wide line to find the closest point
   * on the path, and the insert handle.
   */
  useEffect(() => {
    if (positions.length < 2 || !hoverable) {
      return undefined;
    }
    const hitLine = leaflet.polyline(positions, {
      bubblingMouseEvents: false,
      className: editable ? "cursor-copy" : "",
      interactive: editable,
      opacity: 0,
      weight: 2 * HOVER_DISTANCE_PIXELS,
    });
    hitLine.addTo(map);
    const handle = leaflet.marker([0, 0], {
      draggable: true,
      icon: leaflet.divIcon({
        className: "",
        html: '<div class="route-handle"></div>',
        iconAnchor: [7, 7],
        iconSize: [14, 14],
      }),
      keyboard: false,
      title: "Click to add a waypoint, or drag to move the route",
      zIndexOffset: -1000,
    });
    const rubberBand = leaflet.polyline([], {
      color: PATH_COLOR,
      dashArray: "4 6",
      interactive: false,
      weight: 3,
    });
    let handleIndex = 0;
    /**
     * Whether the handle is pressed, so it is not moved or hidden while the
     * mouse moves away from the path to drag it.
     */
    let pressed = false;

    const hide = () => {
      handle.remove();
      handleHover(null);
    };
    const track = (layerPoint: LeafletPoint) => {
      if (pressed) {
        return;
      }
      const closest = hitLine.closestLayerPoint(layerPoint);
      if (
        closest === null ||
        closest.distanceTo(layerPoint) > HOVER_DISTANCE_PIXELS
      ) {
        hide();
        return;
      }
      const position = map.layerPointToLatLng(closest);
      handleIndex = nearestIndex(positions, position);
      handleHover(handleIndex);
      if (!editable || nearWaypoint(closest)) {
        handle.remove();
        return;
      }
      handle.setLatLng(position);
      handle.addTo(map);
    };
    const onMouseMove = (event: LeafletMouseEvent) => {
      track(event.layerPoint);
    };
    const onMouseOut = () => {
      if (!pressed) {
        hide();
      }
    };
    const release = () => {
      pressed = false;
    };
    map.on("mousemove", onMouseMove);
    map.on("mouseout", onMouseOut);
    map.on("mouseup", release);
    handle.on("mousemove", onMouseMove);
    handle.on("mousedown", () => {
      pressed = true;
    });
    handle.on("click", () => {
      pressed = false;
      handleInsert(handleIndex, handle.getLatLng());
      hide();
    });
    handle.on("dragstart", () => {
      rubberBand.addTo(map);
    });
    handle.on("drag", () => {
      const [before, after] = neighbors(handleIndex);
      const latLngs: LatLngTuple[] = [];
      if (before !== undefined) {
        latLngs.push(before);
      }
      const position = handle.getLatLng();
      latLngs.push([position.lat, position.lng]);
      if (after !== undefined) {
        latLngs.push(after);
      }
      rubberBand.setLatLngs(latLngs);
    });
    handle.on("dragend", () => {
      pressed = false;
      rubberBand.remove();
      rubberBand.setLatLngs([]);
      handleInsert(handleIndex, handle.getLatLng());
      hide();
    });
    // Touch devices have no hover: tapping the path inserts a waypoint.
    hitLine.on("click", (event: LeafletMouseEvent) => {
      const closest = hitLine.closestLayerPoint(event.layerPoint);
      const position =
        closest === null ? event.latlng : map.layerPointToLatLng(closest);
      handleInsert(nearestIndex(positions, position), position);
    });
    return () => {
      map.off("mousemove", onMouseMove);
      map.off("mouseout", onMouseOut);
      map.off("mouseup", release);
      hitLine.remove();
      handle.remove();
      rubberBand.remove();
    };
  }, [map, positions, editable, hoverable]);

  return null;
};

export default RoutePath;
