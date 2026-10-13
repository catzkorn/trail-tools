import type { LatLng, LatLngTuple } from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useEffectEvent } from "react";

import { useMap } from "./map-view";

export interface Waypoint {
  lat: number;
  lng: number;
}

interface WaypointMarkersProps {
  waypoints: Waypoint[];
  // The index of the selected waypoint, if any.
  selected: number | null;
  onMove: (index: number, position: LatLng) => void;
  onSelect: (index: number) => void;
}

const markerClass = (
  index: number,
  count: number,
  selected: boolean
): string => {
  const classes = ["waypoint-marker"];
  if (index === 0) {
    classes.push("waypoint-start");
  } else if (index === count - 1) {
    classes.push("waypoint-finish");
  }
  if (selected) {
    classes.push("waypoint-selected");
  }
  return classes.join(" ");
};

const markerLabel = (index: number, count: number): string => {
  if (index === 0) {
    return "S";
  }
  if (index === count - 1 && count > 1) {
    return "F";
  }
  return String(index + 1);
};

/**
 * MarkerZIndex keeps the selected marker on top, and the start above the
 * finish, as they overlap in a loop.
 */
const markerZIndex = (index: number, selected: number | null): number => {
  if (index === selected) {
    return 1000;
  }
  return index === 0 ? 500 : 0;
};

const toTuple = (waypoint: Waypoint | undefined): LatLngTuple | null =>
  waypoint === undefined ? null : [waypoint.lat, waypoint.lng];

/**
 * WaypointMarkers draws draggable, numbered waypoint markers. Dragging a
 * marker moves the waypoint, with dashed lines to its neighbors while
 * dragging, and clicking it selects the waypoint.
 */
const WaypointMarkers: React.FC<WaypointMarkersProps> = ({
  waypoints,
  selected,
  onMove,
  onSelect,
}) => {
  const map = useMap();
  const handleMove = useEffectEvent((index: number, position: LatLng) => {
    onMove(index, position);
  });
  const handleSelect = useEffectEvent((index: number) => {
    onSelect(index);
  });

  useEffect(() => {
    const group = leaflet.layerGroup();
    const rubberBand = leaflet.polyline([], {
      color: "#d2601a",
      dashArray: "4 6",
      interactive: false,
      weight: 3,
    });
    const count = waypoints.length;
    for (const [index, waypoint] of waypoints.entries()) {
      const label = markerLabel(index, count);
      const marker = leaflet.marker([waypoint.lat, waypoint.lng], {
        draggable: true,
        icon: leaflet.divIcon({
          className: "",
          html: `<div class="${markerClass(index, count, index === selected)}">${label}</div>`,
          iconAnchor: [11, 11],
          iconSize: [22, 22],
        }),
        title: `Waypoint ${index + 1}: drag to move, click to select`,
        zIndexOffset: markerZIndex(index, selected),
      });
      const before = toTuple(waypoints[index - 1]);
      const after = toTuple(waypoints[index + 1]);
      marker.on("dragstart", () => {
        rubberBand.addTo(map);
      });
      marker.on("drag", () => {
        const position = marker.getLatLng();
        const latLngs: LatLngTuple[] = [];
        if (before !== null) {
          latLngs.push(before);
        }
        latLngs.push([position.lat, position.lng]);
        if (after !== null) {
          latLngs.push(after);
        }
        rubberBand.setLatLngs(latLngs);
      });
      marker.on("dragend", () => {
        rubberBand.remove();
        handleMove(index, marker.getLatLng());
      });
      marker.on("click", () => {
        handleSelect(index);
      });
      group.addLayer(marker);
    }
    group.addTo(map);
    return () => {
      group.remove();
      rubberBand.remove();
    };
  }, [map, waypoints, selected]);

  return null;
};

export default WaypointMarkers;
