import type { LatLng } from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useEffectEvent } from "react";

import { useMap } from "./map-view";

export interface Waypoint {
  lat: number;
  lng: number;
}

interface WaypointMarkersProps {
  waypoints: Waypoint[];
  onMove: (index: number, position: LatLng) => void;
  onRemove: (index: number) => void;
}

const markerClass = (index: number, count: number): string => {
  if (index === 0) {
    return "waypoint-marker waypoint-start";
  }
  if (index === count - 1) {
    return "waypoint-marker waypoint-finish";
  }
  return "waypoint-marker";
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
 * WaypointMarkers draws draggable, numbered waypoint markers. Dragging a
 * marker moves the waypoint and clicking it removes the waypoint.
 */
const WaypointMarkers: React.FC<WaypointMarkersProps> = ({
  waypoints,
  onMove,
  onRemove,
}) => {
  const map = useMap();
  const handleMove = useEffectEvent((index: number, position: LatLng) => {
    onMove(index, position);
  });
  const handleRemove = useEffectEvent((index: number) => {
    onRemove(index);
  });

  useEffect(() => {
    const group = leaflet.layerGroup();
    const count = waypoints.length;
    for (const [index, waypoint] of waypoints.entries()) {
      const marker = leaflet.marker([waypoint.lat, waypoint.lng], {
        draggable: true,
        icon: leaflet.divIcon({
          className: "",
          html: `<div class="${markerClass(index, count)}">${markerLabel(index, count)}</div>`,
          iconAnchor: [11, 11],
          iconSize: [22, 22],
        }),
        title: `Waypoint ${index + 1}: drag to move, click to remove`,
      });
      marker.on("dragend", () => {
        handleMove(index, marker.getLatLng());
      });
      marker.on("click", () => {
        handleRemove(index);
      });
      group.addLayer(marker);
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, waypoints]);

  return null;
};

export default WaypointMarkers;
