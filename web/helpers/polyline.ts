// DecodePolyline decodes a Google encoded polyline into [lat, lng] pairs.
// See https://developers.google.com/maps/documentation/utilities/polylinealgorithm.
export const decodePolyline = (encoded: string): [number, number][] => {
  const points: [number, number][] = [];
  let index = 0;
  let lat = 0;
  let lng = 0;

  const readValue = (): number => {
    let result = 0;
    let shift = 0;
    let byte = 0;
    do {
      byte = (encoded.codePointAt(index) ?? 63) - 63;
      index++;
      result |= (byte & 0x1f) << shift;
      shift += 5;
    } while (byte >= 0x20 && index < encoded.length);
    if ((result & 1) === 1) {
      return ~(result >> 1);
    }
    return result >> 1;
  };

  while (index < encoded.length) {
    lat += readValue();
    lng += readValue();
    points.push([lat / 1e5, lng / 1e5]);
  }
  return points;
};
