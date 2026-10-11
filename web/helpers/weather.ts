/**
 * Helpers for displaying weather forecasts. Weather codes are WMO codes as
 * used by Open-Meteo.
 */

// WeatherSymbol returns an emoji for a WMO weather code.
export const weatherSymbol = (code: number, isDay: boolean): string => {
  if (code >= 95) {
    return "⛈️";
  }
  if (code === 85 || code === 86 || (code >= 71 && code <= 77)) {
    return "🌨️";
  }
  if (code >= 80 && code <= 82) {
    return "🌦️";
  }
  if (code >= 61 && code <= 67) {
    return "🌧️";
  }
  if (code >= 51 && code <= 57) {
    return "🌦️";
  }
  if (code === 45 || code === 48) {
    return "🌫️";
  }
  if (code === 3) {
    return "☁️";
  }
  if (code === 1 || code === 2) {
    return isDay ? "🌤️" : "☁️";
  }
  return isDay ? "☀️" : "🌙";
};

// WeatherColor returns a hex background color for a forecast marker.
export const weatherColor = (code: number, isDay: boolean): string => {
  if (code >= 95) {
    return "#7c3aed";
  }
  if (code === 85 || code === 86 || (code >= 71 && code <= 77)) {
    return "#67e8f9";
  }
  if (code >= 51 && code <= 82) {
    return "#3b82f6";
  }
  if (code === 45 || code === 48) {
    return "#9ca3af";
  }
  if (code === 3) {
    return "#94a3b8";
  }
  return isDay ? "#fcd34d" : "#334155";
};

const COMPASS = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"];

// CompassDirection returns e.g. "SW" for a direction in degrees.
export const compassDirection = (degrees: number): string => {
  const index = Math.round((((degrees % 360) + 360) % 360) / 45) % 8;
  return COMPASS[index] ?? "";
};

/**
 * TemperatureClass returns Tailwind text classes for a temperature in
 * degrees Celsius.
 */
export const temperatureClass = (celsius: number): string => {
  if (celsius <= 0) {
    return "text-sky-700";
  }
  if (celsius >= 28) {
    return "text-red-700";
  }
  return "text-gray-900";
};
