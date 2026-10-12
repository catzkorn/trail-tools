import {
  FitBounds,
  FlyTo,
  HoverMarker,
  MapClick,
  PointMarker,
  PolylineLayer,
} from "@components/map/layers";
import type { MapFocus } from "@components/map/layers";
import MapView, { useMap } from "@components/map/map-view";
import PhotoMarkers from "@components/map/photo-markers";
import type { MapPhoto } from "@components/map/photo-markers";
import type { LatLng, LatLngTuple } from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useRef } from "react";

interface TripMapProps {
  track: LatLngTuple[];
  photos: MapPhoto[];
  selectedId: string | null;
  // The position hovered on the elevation profile.
  hoverPosition?: LatLngTuple | null;
  onSelectPhoto?: (id: string) => void;
  // Makes the photo markers draggable.
  onMovePhoto?: (id: string, position: LatLng) => void;
  // Shows the areas around the start and finish hidden on the public page.
  hiddenEndsMeters?: number;
  // Called with clicks on the map, e.g. to place a photo.
  onMapClick?: ((position: LatLng) => void) | undefined;
  // Zooms the map to a position, e.g. of a photo shown on the map.
  focus?: MapFocus | null;
  // A stretch of the track to highlight, e.g. the section being read.
  highlight?: LatLngTuple[];
  /**
   * The map moves to fit the highlight, or the whole track without one, when
   * the key changes.
   */
  highlightKey?: string;
  fitKey: string;
  className?: string;
}

interface HiddenAreaProps {
  center: LatLngTuple;
  radiusMeters: number;
}

// HiddenArea shades an area of the track hidden on the public page.
const HiddenArea: React.FC<HiddenAreaProps> = ({ center, radiusMeters }) => {
  const map = useMap();
  const [lat, lng] = center;
  useEffect(() => {
    const circle = leaflet
      .circle([lat, lng], {
        color: "#4b5563",
        dashArray: "4 4",
        fillOpacity: 0.15,
        interactive: false,
        radius: radiusMeters,
        weight: 1.5,
      })
      .addTo(map);
    return () => {
      circle.remove();
    };
  }, [map, lat, lng, radiusMeters]);
  return null;
};

interface FitHighlightProps {
  positions: LatLngTuple[];
  highlightKey: string;
}

// FitHighlight moves the map to fit positions when the key changes.
const FitHighlight: React.FC<FitHighlightProps> = ({
  positions,
  highlightKey,
}) => {
  const map = useMap();
  const fittedKey = useRef(highlightKey);
  useEffect(() => {
    if (fittedKey.current === highlightKey || positions.length < 2) {
      return;
    }
    fittedKey.current = highlightKey;
    const bounds = leaflet.latLngBounds(positions);
    const options = { maxZoom: 15, padding: [32, 32] as [number, number] };
    if (globalThis.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      map.fitBounds(bounds, options);
    } else {
      map.flyToBounds(bounds, { ...options, duration: 0.8 });
    }
  }, [map, positions, highlightKey]);
  return null;
};

const NO_HIGHLIGHT: LatLngTuple[] = [];

/**
 * TripMap shows a trip's track with its photos. Without a track, it fits the
 * photos.
 */
const TripMap: React.FC<TripMapProps> = ({
  track,
  photos,
  selectedId,
  hoverPosition = null,
  onSelectPhoto,
  onMovePhoto,
  onMapClick,
  hiddenEndsMeters = 0,
  focus = null,
  highlight = NO_HIGHLIGHT,
  highlightKey,
  fitKey,
  className = "h-96",
}) => {
  const [start] = track;
  const finish = track.at(-1);
  const fitPositions =
    track.length > 1 ? track : photos.map((photo) => photo.position);
  return (
    <MapView className={className}>
      {track.length > 1 && (
        <PolylineLayer
          positions={track}
          color="#ea580c"
          weight={4}
          opacity={highlight.length > 1 ? 0.35 : 0.9}
        />
      )}
      {highlight.length > 1 && (
        <PolylineLayer
          positions={highlight}
          color="#c2410c"
          weight={6}
          opacity={1}
        />
      )}
      {start !== undefined && (
        <PointMarker position={start} color="#16a34a" tooltip="Start" />
      )}
      {finish !== undefined && track.length > 1 && (
        <PointMarker position={finish} color="#dc2626" tooltip="Finish" />
      )}
      {hiddenEndsMeters > 0 && start !== undefined && (
        <HiddenArea center={start} radiusMeters={hiddenEndsMeters} />
      )}
      {hiddenEndsMeters > 0 && finish !== undefined && track.length > 1 && (
        <HiddenArea center={finish} radiusMeters={hiddenEndsMeters} />
      )}
      <HoverMarker position={hoverPosition} />
      <PhotoMarkers
        photos={photos}
        selectedId={selectedId}
        onSelect={onSelectPhoto}
        onMove={onMovePhoto}
      />
      {onMapClick !== undefined && <MapClick onClick={onMapClick} />}
      <FitBounds positions={fitPositions} fitKey={fitKey} />
      <FlyTo focus={focus} />
      {highlightKey !== undefined && (
        <FitHighlight
          positions={highlight.length > 1 ? highlight : fitPositions}
          highlightKey={highlightKey}
        />
      )}
    </MapView>
  );
};

export default TripMap;
