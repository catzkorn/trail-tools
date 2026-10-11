import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useMap } from "@components/map/map-view";
import type { ForecastPoint } from "gen/routes/v1/routes_pb";
import { weatherColor, weatherSymbol } from "helpers/weather";
import * as leaflet from "leaflet";
import React, { useEffect } from "react";

import { forecastKindLabel, formatClock } from "./forecast-utils";

const popupContent = (point: ForecastPoint): HTMLElement => {
  const root = document.createElement("div");
  root.className = "text-xs";
  const lines = [
    `${formatClock(point.time)} · km ${(point.distanceMeters / 1000).toFixed(1)} · ${Math.round(point.elevationMeters)} m`,
    `${weatherSymbol(point.weatherCode, point.isDay)} ${point.weatherDescription}`,
    `${point.temperatureCelsius.toFixed(0)}°C (feels ${point.apparentTemperatureCelsius.toFixed(0)}°C)`,
    `Precipitation ${point.precipitationProbabilityPercent}% · ${point.precipitationMm.toFixed(1)} mm`,
    `Wind ${Math.round(point.windSpeedKmh)} km/h, gusts ${Math.round(point.windGustsKmh)} km/h`,
  ];
  if (point.freezingLevelMeters > 0) {
    lines.push(`Freezing level ${Math.round(point.freezingLevelMeters)} m`);
  }
  const kind = forecastKindLabel(point.kind);
  if (kind !== "") {
    const title = document.createElement("strong");
    title.textContent = kind;
    root.append(title);
  }
  for (const line of lines) {
    const row = document.createElement("div");
    row.textContent = line;
    root.append(row);
  }
  return root;
};

const markerIcon = (point: ForecastPoint): HTMLElement => {
  const element = document.createElement("div");
  element.className = "forecast-marker";
  element.style.backgroundColor = weatherColor(point.weatherCode, point.isDay);
  element.textContent = weatherSymbol(point.weatherCode, point.isDay);
  return element;
};

interface ForecastMarkersProps {
  points: ForecastPoint[];
}

// ForecastMarkers draws a weather marker at each forecast point.
const ForecastMarkers: React.FC<ForecastMarkersProps> = ({ points }) => {
  const map = useMap();
  useEffect(() => {
    const group = leaflet.layerGroup();
    for (const point of points) {
      if (point.location === undefined) {
        continue;
      }
      const marker = leaflet.marker([point.location.lat, point.location.lng], {
        icon: leaflet.divIcon({
          className: "",
          html: markerIcon(point),
          iconAnchor: [13, 13],
          iconSize: [26, 26],
        }),
        title:
          point.time === undefined
            ? point.weatherDescription
            : timestampDate(point.time).toLocaleTimeString(),
      });
      marker.bindPopup(popupContent(point));
      group.addLayer(marker);
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, points]);
  return null;
};

export default ForecastMarkers;
