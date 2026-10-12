import {
  FitBounds,
  FlyTo,
  HoverMarker,
  MapClick,
  PointMarker,
  PolylineLayer,
} from "@components/map/layers";
import type { MapFocus } from "@components/map/layers";
import MapView from "@components/map/map-view";
import PhotoMarkers from "@components/map/photo-markers";
import type { MapPhoto } from "@components/map/photo-markers";
import type { LatLng, LatLngTuple } from "leaflet";
import React from "react";

interface TripMapProps {
  track: LatLngTuple[];
  photos: MapPhoto[];
  selectedId: string | null;
  // The position hovered on the elevation profile.
  hoverPosition?: LatLngTuple | null;
  onSelectPhoto?: (id: string) => void;
  // Makes the photo markers draggable.
  onMovePhoto?: (id: string, position: LatLng) => void;
  // Called with clicks on the map, e.g. to place a photo.
  onMapClick?: ((position: LatLng) => void) | undefined;
  // Zooms the map to a position, e.g. of a photo shown on the map.
  focus?: MapFocus | null;
  fitKey: string;
  className?: string;
}

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
  focus = null,
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
        <PolylineLayer positions={track} color="#ea580c" weight={4} />
      )}
      {start !== undefined && (
        <PointMarker position={start} color="#16a34a" tooltip="Start" />
      )}
      {finish !== undefined && track.length > 1 && (
        <PointMarker position={finish} color="#dc2626" tooltip="Finish" />
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
    </MapView>
  );
};

export default TripMap;
