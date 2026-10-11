import { useQuery } from "@connectrpc/connect-query";
import { keepPreviousData, skipToken } from "@tanstack/react-query";
import {
  findPointsOfInterest,
  findTrails,
  getPersonalHeatmap,
} from "gen/geo/v1/geo-GeoService_connectquery";
import { formatKm } from "helpers/format";
import { poiStyle, terrainStyle } from "helpers/labels";
import * as leaflet from "leaflet";
import React, { useEffect } from "react";

import type { Viewport } from "./layers";
import { textElement } from "./layers";
import { useMap } from "./map-view";

export const TRAILS_MIN_ZOOM = 13;
export const POI_MIN_ZOOM = 12;
const MAX_RADIUS_METERS = 15_000;
const METERS_PER_DEGREE = 111_320;

const round = (value: number, decimals: number): number => {
  const factor = 10 ** decimals;
  return Math.round(value * factor) / factor;
};

// SearchRadius is half the visible width, capped at the API maximum.
const searchRadius = (viewport: Viewport): number => {
  const { bounds } = viewport;
  const width = bounds.getNorthWest().distanceTo(bounds.getNorthEast());
  return Math.round(Math.min(MAX_RADIUS_METERS, width / 2));
};

const cellSizeForZoom = (zoom: number): number => {
  if (zoom >= 15) {
    return 25;
  }
  if (zoom >= 14) {
    return 50;
  }
  if (zoom >= 12) {
    return 100;
  }
  if (zoom >= 10) {
    return 250;
  }
  return 1000;
};

const HEAT_COLORS = ["#fde68a", "#fbbf24", "#f97316", "#dc2626", "#7f1d1d"];

const heatColor = (count: number, maxCount: number): string => {
  if (maxCount <= 1) {
    return "#f97316";
  }
  const intensity = Math.log(count) / Math.log(maxCount);
  const index = Math.min(
    HEAT_COLORS.length - 1,
    Math.floor(intensity * HEAT_COLORS.length)
  );
  return HEAT_COLORS[index] ?? "#f97316";
};

interface OverlayProps {
  viewport: Viewport;
}

// HeatmapOverlay draws the user's personal heatmap for the visible area.
export const HeatmapOverlay: React.FC<OverlayProps> = ({ viewport }) => {
  const map = useMap();
  const { bounds, zoom } = viewport;
  const cellSize = cellSizeForZoom(zoom);
  const { data } = useQuery(
    getPersonalHeatmap,
    {
      bounds: {
        east: round(bounds.getEast(), 3),
        north: round(bounds.getNorth(), 3),
        south: round(bounds.getSouth(), 3),
        west: round(bounds.getWest(), 3),
      },
      cellSizeMeters: cellSize,
      limit: 5000,
    },
    { placeholderData: keepPreviousData }
  );

  useEffect(() => {
    if (data === undefined) {
      return undefined;
    }
    const renderer = leaflet.canvas({ padding: 0.2 });
    const group = leaflet.layerGroup();
    let maxCount = 1;
    for (const cell of data.cells) {
      maxCount = Math.max(maxCount, cell.activityCount);
    }
    const half = data.cellSizeMeters / 2;
    for (const cell of data.cells) {
      if (cell.center === undefined) {
        continue;
      }
      const { lat, lng } = cell.center;
      const latOffset = half / METERS_PER_DEGREE;
      const lngOffset =
        half / (METERS_PER_DEGREE * Math.cos((lat * Math.PI) / 180));
      const color = heatColor(cell.activityCount, maxCount);
      group.addLayer(
        leaflet.rectangle(
          [
            [lat - latOffset, lng - lngOffset],
            [lat + latOffset, lng + lngOffset],
          ],
          {
            color,
            fillColor: color,
            fillOpacity: 0.55,
            interactive: false,
            renderer,
            weight: 0,
          }
        )
      );
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, data]);

  return null;
};

// TrailsOverlay draws OpenStreetMap trails around the map center.
export const TrailsOverlay: React.FC<OverlayProps> = ({ viewport }) => {
  const map = useMap();
  const { center, zoom } = viewport;
  const enabled = zoom >= TRAILS_MIN_ZOOM;
  const { data } = useQuery(
    findTrails,
    enabled
      ? {
          center: { lat: round(center.lat, 3), lng: round(center.lng, 3) },
          includeGeometry: true,
          limit: 300,
          radiusMeters: searchRadius(viewport),
        }
      : skipToken,
    { placeholderData: keepPreviousData }
  );

  useEffect(() => {
    if (data === undefined || !enabled) {
      return undefined;
    }
    const group = leaflet.layerGroup();
    for (const trail of data.trails) {
      if (trail.geometry.length < 2) {
        continue;
      }
      const style = terrainStyle(trail.terrain);
      const line = leaflet.polyline(
        trail.geometry.map((point): leaflet.LatLngTuple => [
          point.lat,
          point.lng,
        ]),
        {
          color: style.color,
          dashArray: "6 4",
          opacity: 0.9,
          weight: 2.5,
        }
      );
      const details = [trail.highway, trail.surface, trail.sacScale]
        .filter((detail) => detail !== "")
        .join(", ");
      const name = trail.name === "" ? "Unnamed trail" : trail.name;
      line.bindTooltip(
        textElement(
          `${name} · ${style.label} · ${formatKm(trail.lengthMeters)}${details === "" ? "" : ` (${details})`}`
        ),
        { sticky: true }
      );
      group.addLayer(line);
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, data, enabled]);

  return null;
};

const poiIcon = (symbol: string, color: string, name: string): HTMLElement => {
  const root = document.createElement("div");
  root.className = "poi-marker";
  const badge = document.createElement("span");
  badge.className = "poi-symbol";
  badge.style.backgroundColor = color;
  badge.textContent = symbol;
  root.append(badge);
  if (name !== "") {
    const label = document.createElement("span");
    label.className = "poi-label";
    label.textContent = name;
    root.append(label);
  }
  return root;
};

// PoiOverlay draws points of interest such as peaks, water and parking.
export const PoiOverlay: React.FC<OverlayProps> = ({ viewport }) => {
  const map = useMap();
  const { center, zoom } = viewport;
  const enabled = zoom >= POI_MIN_ZOOM;
  const { data } = useQuery(
    findPointsOfInterest,
    enabled
      ? {
          center: { lat: round(center.lat, 3), lng: round(center.lng, 3) },
          limit: 200,
          radiusMeters: searchRadius(viewport),
        }
      : skipToken,
    { placeholderData: keepPreviousData }
  );

  useEffect(() => {
    if (data === undefined || !enabled) {
      return undefined;
    }
    const group = leaflet.layerGroup();
    for (const point of data.points) {
      if (point.location === undefined) {
        continue;
      }
      const style = poiStyle(point.kind);
      const elevation =
        point.elevationMeters > 0
          ? ` (${Math.round(point.elevationMeters)} m)`
          : "";
      const marker = leaflet.marker([point.location.lat, point.location.lng], {
        icon: leaflet.divIcon({
          className: "",
          html: poiIcon(style.symbol, style.color, point.name),
          iconAnchor: [8, 8],
          iconSize: [16, 16],
        }),
        keyboard: false,
      });
      const name = point.name === "" ? style.label : point.name;
      marker.bindTooltip(textElement(`${name}${elevation} · ${style.label}`));
      group.addLayer(marker);
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, data, enabled]);

  return null;
};
