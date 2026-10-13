import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import { listWorkoutWeather } from "gen/plans/v1/plans-TrainingPlanService_connectquery";
import type {
  DayWeather,
  HourWeather,
  WorkoutWeather,
} from "gen/plans/v1/plans_pb";
import { WeatherLocationSource } from "gen/plans/v1/plans_pb";
import {
  compassDirection,
  temperatureClass,
  weatherSymbol,
} from "helpers/weather";
import React, { useMemo } from "react";

// Forecasts change slowly, so they're kept for half an hour.
const WEATHER_STALE_MS = 30 * 60 * 1000;

/**
 * UseWorkoutWeather returns the forecasts of the workouts in a date range by
 * workout ID. Forecasts are an extra, so without one the map is empty.
 */
export const useWorkoutWeather = (
  startDate: string,
  endDate: string
): Map<string, WorkoutWeather> => {
  const { data } = useQuery(
    listWorkoutWeather,
    { endDate, startDate },
    { retry: false, staleTime: WEATHER_STALE_MS }
  );
  return useMemo(
    () => new Map((data?.weather ?? []).map((item) => [item.workoutId, item])),
    [data]
  );
};

const degrees = (celsius: number): string => `${Math.round(celsius)}°`;

// RAIN_SHOWN is the chance of precipitation from which it's mentioned.
const RAIN_SHOWN = 40;

// DaySummary describes a day's weather in words.
const daySummary = (day: DayWeather): string => {
  const parts = [
    day.description,
    `${degrees(day.minTemperatureCelsius)} to ${degrees(day.maxTemperatureCelsius)}C`,
  ];
  if (day.maxPrecipitationProbability > 0) {
    parts.push(`${day.maxPrecipitationProbability}% chance of precipitation`);
  }
  return parts.join(", ");
};

// WeatherBadge is a one line summary of a day's weather, for workout cards.
export const WeatherBadge: React.FC<{ day: DayWeather }> = ({ day }) => (
  <span
    className="flex min-w-0 flex-wrap items-center gap-x-1 tabular-nums"
    title={daySummary(day)}
  >
    <span aria-hidden="true">{weatherSymbol(day.weatherCode, true)}</span>
    <span className="sr-only">{daySummary(day)}</span>
    <span aria-hidden="true">
      {degrees(day.maxTemperatureCelsius)}
      <span className="opacity-60">/{degrees(day.minTemperatureCelsius)}</span>
    </span>
    {day.maxPrecipitationProbability >= RAIN_SHOWN && (
      <span aria-hidden="true" className="text-sky-700">
        {day.maxPrecipitationProbability}%
      </span>
    )}
  </span>
);

const locationLabel = (source: WeatherLocationSource): string =>
  source === WeatherLocationSource.ROUTE_START
    ? "At the start of the route."
    : "Where your latest activity started. Add a route for the forecast where you'll run.";

// FormatHour formats the hour of a time, e.g. "7 AM" or "07".
const formatHour = (time: Timestamp | undefined): string =>
  time === undefined
    ? ""
    : timestampDate(time).toLocaleTimeString(undefined, { hour: "numeric" });

const HourCell: React.FC<{ hour: HourWeather }> = ({ hour }) => (
  <li className="flex w-14 shrink-0 flex-col items-center gap-0.5 rounded-md py-2 text-xs tabular-nums">
    <span className="text-gray-500">{formatHour(hour.time)}</span>
    <span className="text-xl" aria-hidden="true">
      {weatherSymbol(hour.weatherCode, hour.isDay)}
    </span>
    <span
      className={`font-semibold ${temperatureClass(hour.temperatureCelsius)}`}
    >
      {degrees(hour.temperatureCelsius)}
    </span>
    <span
      className={
        hour.precipitationProbability >= RAIN_SHOWN
          ? "text-sky-700"
          : "text-gray-400"
      }
    >
      {hour.precipitationProbability}%
    </span>
    <span className="text-gray-500">
      {Math.round(hour.windSpeedKmh)}{" "}
      {compassDirection(hour.windDirectionDegrees)}
    </span>
  </li>
);

// WeatherForecast shows the forecast of a workout's day, hour by hour.
export const WeatherForecast: React.FC<{ weather: WorkoutWeather }> = ({
  weather,
}) => {
  const { day } = weather;
  if (day === undefined) {
    return null;
  }
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
        <div className="flex items-center gap-3">
          <span className="text-4xl" aria-hidden="true">
            {weatherSymbol(day.weatherCode, true)}
          </span>
          <div>
            <p className="font-semibold text-gray-900">{day.description}</p>
            <p className="text-sm text-gray-600 tabular-nums">
              <span className={temperatureClass(day.maxTemperatureCelsius)}>
                {degrees(day.maxTemperatureCelsius)}
              </span>{" "}
              /{" "}
              <span className={temperatureClass(day.minTemperatureCelsius)}>
                {degrees(day.minTemperatureCelsius)}
              </span>
            </p>
          </div>
        </div>
        <dl className="flex flex-wrap gap-x-6 gap-y-1 text-sm tabular-nums">
          <div>
            <dt className="text-xs text-gray-500">Precipitation</dt>
            <dd className="font-medium text-gray-900">
              {day.maxPrecipitationProbability}% ·{" "}
              {day.precipitationMm.toFixed(1)} mm
              {day.snowfallCm > 0 && ` · ${day.snowfallCm.toFixed(1)} cm snow`}
            </dd>
          </div>
          <div>
            <dt className="text-xs text-gray-500">Wind</dt>
            <dd className="font-medium text-gray-900">
              {Math.round(day.maxWindSpeedKmh)} km/h, gusts{" "}
              {Math.round(day.maxWindGustsKmh)}
            </dd>
          </div>
        </dl>
      </div>
      {weather.hours.length > 0 && (
        <ol
          className="-mx-1 flex overflow-x-auto px-1"
          aria-label="Hourly forecast"
        >
          {weather.hours.map((hour) => (
            <HourCell key={String(hour.time?.seconds)} hour={hour} />
          ))}
        </ol>
      )}
      <p className="text-xs text-gray-500">
        {locationLabel(weather.locationSource)}
      </p>
    </div>
  );
};
