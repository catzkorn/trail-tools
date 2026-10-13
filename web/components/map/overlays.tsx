import type { ConnectError } from "@connectrpc/connect";
import { Code } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import type { UseQueryResult } from "@tanstack/react-query";
import { keepPreviousData, skipToken } from "@tanstack/react-query";
import {
  findTrails,
  listHeatmapTracks,
} from "gen/geo/v1/geo-GeoService_connectquery";
import type { FindTrailsResponse } from "gen/geo/v1/geo_pb";
import { formatKm } from "helpers/format";
import { terrainStyle } from "helpers/labels";
import { decodePolyline } from "helpers/polyline";
import * as leaflet from "leaflet";
import React, { useEffect } from "react";

import type { Viewport } from "./layers";
import { textElement } from "./layers";
import { useMap } from "./map-view";

export const TRAILS_MIN_ZOOM = 13;
const MAX_RADIUS_METERS = 15_000;

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

// HEAT_COLOR is the color of the heatmap's tracks, darker where they overlap.
const HEAT_COLOR = "#ea580c";

// MetersPerPixel is the size of a screen pixel on the map at a latitude.
const metersPerPixel = (lat: number, zoom: number): number =>
  (40_075_016 * Math.cos((lat * Math.PI) / 180)) / 2 ** (zoom + 8);

interface OverlayProps {
  viewport: Viewport;
}

/**
 * NotImported reports whether finding trails failed
 * because no map data has been imported for the area.
 */
export const notImported = (error: ConnectError | null): boolean =>
  error?.code === Code.FailedPrecondition;

// Retry retries failed queries, except where no map data has been imported.
const retry = (failureCount: number, error: ConnectError): boolean =>
  !notImported(error) && failureCount < 3;

/**
 * UseTrails finds the trails around the map center, if shown and zoomed in
 * enough. Calls with the same viewport share the query.
 */
export const useTrails = (
  viewport: Viewport,
  shown: boolean
): UseQueryResult<FindTrailsResponse, ConnectError> => {
  const { center, zoom } = viewport;
  return useQuery(
    findTrails,
    shown && zoom >= TRAILS_MIN_ZOOM
      ? {
          center: { lat: round(center.lat, 3), lng: round(center.lng, 3) },
          includeGeometry: true,
          limit: 300,
          radiusMeters: searchRadius(viewport),
        }
      : skipToken,
    { placeholderData: keepPreviousData, retry }
  );
};

/**
 * HeatmapOverlay draws the user's activity tracks in the visible area as thin
 * translucent lines, darker where the user goes often.
 */
export const HeatmapOverlay: React.FC<OverlayProps> = ({ viewport }) => {
  const map = useMap();
  const { bounds, zoom } = viewport;
  // A little beyond the edges, so tracks don't stop short while panning.
  const area = bounds.pad(0.1);
  const { data } = useQuery(
    listHeatmapTracks,
    {
      bounds: {
        east: round(area.getEast(), 3),
        north: round(area.getNorth(), 3),
        south: round(area.getSouth(), 3),
        west: round(area.getWest(), 3),
      },
      // Simplified to about a pixel.
      toleranceMeters: Math.max(
        1,
        Math.round(metersPerPixel(bounds.getCenter().lat, zoom))
      ),
    },
    { placeholderData: keepPreviousData }
  );

  useEffect(() => {
    if (data === undefined) {
      return undefined;
    }
    const renderer = leaflet.canvas({ padding: 0.2 });
    const group = leaflet.layerGroup();
    for (const polyline of data.polylines) {
      group.addLayer(
        leaflet.polyline(decodePolyline(polyline), {
          color: HEAT_COLOR,
          interactive: false,
          lineCap: "round",
          lineJoin: "round",
          opacity: 0.35,
          renderer,
          weight: 2,
        })
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
  const enabled = viewport.zoom >= TRAILS_MIN_ZOOM;
  const { data } = useTrails(viewport, true);

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
