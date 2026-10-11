import type {
  LatLng,
  LatLngBounds,
  LatLngTuple,
  LeafletMouseEvent,
} from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useEffectEvent, useRef, useState } from "react";

import { useMap } from "./map-view";

/**
 * TextElement wraps text in an element, so that it can be passed to Leaflet
 * tooltips without being interpreted as HTML.
 */
export const textElement = (text: string): HTMLElement => {
  const element = document.createElement("span");
  element.textContent = text;
  return element;
};

interface PolylineLayerProps {
  positions: LatLngTuple[];
  color: string;
  weight?: number;
  opacity?: number;
  dashArray?: string;
  /**
   * Called with the clicked location when the line is clicked. Clicks on the
   * line do not reach the map's click handler.
   */
  onClick?: ((position: LatLng) => void) | undefined;
}

// PolylineLayer draws a line on the map.
export const PolylineLayer: React.FC<PolylineLayerProps> = ({
  positions,
  color,
  weight = 4,
  opacity = 0.9,
  dashArray,
  onClick,
}) => {
  const map = useMap();
  const clickable = onClick !== undefined;
  const handleClick = useEffectEvent((position: LatLng) => {
    if (onClick !== undefined) {
      onClick(position);
    }
  });

  // The line is recreated when anything changes, which is cheap.
  useEffect(() => {
    const line = leaflet.polyline(positions, {
      bubblingMouseEvents: !clickable,
      className: clickable ? "cursor-copy" : "",
      color,
      dashArray,
      interactive: clickable,
      opacity,
      weight,
    });
    line.on("click", (event: LeafletMouseEvent) => {
      handleClick(event.latlng);
    });
    line.addTo(map);
    return () => {
      line.remove();
    };
  }, [map, positions, color, weight, opacity, dashArray, clickable]);

  return null;
};

interface PointMarkerProps {
  position: LatLngTuple;
  color: string;
  radius?: number;
  tooltip?: string;
}

// PointMarker draws a small circle, e.g. for a start or finish.
export const PointMarker: React.FC<PointMarkerProps> = ({
  position,
  color,
  radius = 7,
  tooltip,
}) => {
  const map = useMap();
  const [lat, lng] = position;
  useEffect(() => {
    const marker = leaflet.circleMarker([lat, lng], {
      color: "#ffffff",
      fillColor: color,
      fillOpacity: 1,
      radius,
      weight: 2,
    });
    if (tooltip !== undefined) {
      marker.bindTooltip(textElement(tooltip));
    }
    marker.addTo(map);
    return () => {
      marker.remove();
    };
  }, [map, lat, lng, color, radius, tooltip]);
  return null;
};

interface HoverMarkerProps {
  position: LatLngTuple | null;
}

// HoverMarker shows the point currently hovered on an elevation profile.
export const HoverMarker: React.FC<HoverMarkerProps> = ({ position }) => {
  const map = useMap();
  const hasPosition = position !== null;
  const [lat, lng] = position ?? [0, 0];
  useEffect(() => {
    if (!hasPosition) {
      return undefined;
    }
    const marker = leaflet.circleMarker([lat, lng], {
      color: "#111827",
      fillColor: "#facc15",
      fillOpacity: 1,
      interactive: false,
      radius: 6,
      weight: 2,
    });
    marker.addTo(map);
    return () => {
      marker.remove();
    };
  }, [map, hasPosition, lat, lng]);
  return null;
};

interface FitBoundsProps {
  positions: LatLngTuple[];
  /**
   * The map is only fitted again when the key changes, so that it does not
   * jump around while the positions are being edited.
   */
  fitKey: string;
}

// FitBounds zooms the map to show all positions.
export const FitBounds: React.FC<FitBoundsProps> = ({ positions, fitKey }) => {
  const map = useMap();
  const fittedKey = useRef("");
  useEffect(() => {
    if (positions.length === 0 || fittedKey.current === fitKey) {
      return;
    }
    fittedKey.current = fitKey;
    map.fitBounds(leaflet.latLngBounds(positions), {
      maxZoom: 15,
      padding: [24, 24],
    });
  }, [map, positions, fitKey]);
  return null;
};

interface MapClickProps {
  onClick: (position: LatLng) => void;
}

// MapClick calls onClick with the location of clicks on the map.
export const MapClick: React.FC<MapClickProps> = ({ onClick }) => {
  const map = useMap();
  const handleClick = useEffectEvent((position: LatLng) => {
    onClick(position);
  });
  useEffect(() => {
    const handler = (event: LeafletMouseEvent) => {
      handleClick(event.latlng);
    };
    map.on("click", handler);
    const container = map.getContainer();
    container.style.cursor = "crosshair";
    return () => {
      map.off("click", handler);
      container.style.cursor = "";
    };
  }, [map]);
  return null;
};

export interface Viewport {
  bounds: LatLngBounds;
  center: LatLng;
  zoom: number;
}

const currentViewport = (map: leaflet.Map): Viewport => ({
  bounds: map.getBounds(),
  center: map.getCenter(),
  zoom: map.getZoom(),
});

// UseViewport returns the map's viewport, updated when the map stops moving.
export const useViewport = (): Viewport => {
  const map = useMap();
  const [viewport, setViewport] = useState<Viewport>(() =>
    currentViewport(map)
  );
  useEffect(() => {
    const handler = () => {
      setViewport(currentViewport(map));
    };
    map.on("moveend", handler);
    return () => {
      map.off("moveend", handler);
    };
  }, [map]);
  return viewport;
};
