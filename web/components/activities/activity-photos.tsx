import { timestampMs } from "@bufbuild/protobuf/wkt";
import PhotoMarkers from "@components/map/photo-markers";
import type { MapPhoto } from "@components/map/photo-markers";
import Card from "@components/ui/card";
import Lightbox from "@components/ui/lightbox";
import type {
  Activity,
  ActivityPhoto,
  ActivityStreams,
} from "gen/activities/v1/activities_pb";
import type { LatLngTuple } from "leaflet";
import React, { useMemo } from "react";

// PlacedPhoto is a photo with where it was taken, if known.
export interface PlacedPhoto {
  photo: ActivityPhoto;
  position: LatLngTuple | undefined;
}

/**
 * PositionAt returns the position on the track at a time since the start,
 * interpolated between the recorded points.
 */
const positionAt = (
  streams: ActivityStreams,
  seconds: number
): LatLngTuple | undefined => {
  const times = streams.timeSeconds;
  if (times.length !== streams.lat.length || times.length === 0) {
    return undefined;
  }
  const next = times.findIndex((time) => time >= seconds);
  if (next < 0 || (next === 0 && (times[0] ?? 0) > seconds)) {
    return undefined;
  }
  const after: LatLngTuple = [streams.lat[next] ?? 0, streams.lng[next] ?? 0];
  if (next === 0) {
    return after;
  }
  const before: LatLngTuple = [
    streams.lat[next - 1] ?? 0,
    streams.lng[next - 1] ?? 0,
  ];
  const start = times[next - 1] ?? 0;
  const span = (times[next] ?? 0) - start;
  const fraction = span > 0 ? (seconds - start) / span : 0;
  return [
    before[0] + (after[0] - before[0]) * fraction,
    before[1] + (after[1] - before[1]) * fraction,
  ];
};

/**
 * PlacePhotos places photos at their location, or else on the track at the
 * time they were taken, like trip reports do.
 */
export const placePhotos = (
  photos: ActivityPhoto[],
  activity: Activity | undefined,
  streams: ActivityStreams | undefined
): PlacedPhoto[] =>
  photos.map((photo) => {
    if (photo.location !== undefined) {
      return {
        photo,
        position: [photo.location.lat, photo.location.lng],
      };
    }
    if (
      photo.takenTime === undefined ||
      activity?.startTime === undefined ||
      streams === undefined
    ) {
      return { photo, position: undefined };
    }
    const seconds =
      (timestampMs(photo.takenTime) - timestampMs(activity.startTime)) / 1000;
    return { photo, position: positionAt(streams, seconds) };
  });

interface ActivityPhotoMarkersProps {
  photos: PlacedPhoto[];
  // The index of the photo shown in the lightbox, or null.
  openIndex: number | null;
  onOpen: (index: number) => void;
}

/**
 * ActivityPhotoMarkers marks the photos on the activity's map. Clicking one
 * opens it.
 */
export const ActivityPhotoMarkers: React.FC<ActivityPhotoMarkersProps> = ({
  photos,
  openIndex,
  onOpen,
}) => {
  const markers = useMemo(() => {
    const result: MapPhoto[] = [];
    for (const [index, { photo, position }] of photos.entries()) {
      if (position !== undefined) {
        result.push({
          caption: photo.caption,
          id: photo.id,
          number: index + 1,
          position,
          thumbnailUrl: photo.url,
        });
      }
    }
    return result;
  }, [photos]);
  const selectedId =
    openIndex === null ? null : (photos[openIndex]?.photo.id ?? null);
  return (
    <PhotoMarkers
      photos={markers}
      selectedId={selectedId}
      onSelect={(id) => {
        onOpen(photos.findIndex(({ photo }) => photo.id === id));
      }}
    />
  );
};

interface ActivityPhotosProps {
  photos: PlacedPhoto[];
  // The index of the photo shown in the lightbox, or null when closed.
  openIndex: number | null;
  onOpen: (index: number | null) => void;
}

/**
 * ActivityPhotos shows the photos added to the activity on Strava, with
 * their captions, and opens them in a lightbox. The images are served by
 * Strava.
 */
const ActivityPhotos: React.FC<ActivityPhotosProps> = ({
  photos,
  openIndex,
  onOpen,
}) => {
  const lightboxPhotos = useMemo(
    () => photos.map(({ photo }) => photo),
    [photos]
  );
  return (
    <Card title={`Photos (${photos.length})`}>
      <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {photos.map(({ photo }, index) => (
          <li key={photo.id} className="flex min-w-0 flex-col gap-1">
            <button
              type="button"
              className="relative block overflow-hidden rounded-md bg-gray-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
              aria-label={`Open photo ${index + 1}`}
              onClick={() => {
                onOpen(index);
              }}
            >
              <img
                src={photo.url}
                alt={photo.caption}
                loading="lazy"
                decoding="async"
                className="aspect-[4/3] w-full object-cover transition-transform hover:scale-105"
              />
              <span className="absolute top-1 left-1 inline-flex size-5 items-center justify-center rounded-full border-2 border-white bg-emerald-700 text-[10px] font-bold text-white">
                {index + 1}
              </span>
            </button>
            {photo.caption !== "" && (
              <p className="line-clamp-2 text-xs text-gray-600">
                {photo.caption}
              </p>
            )}
          </li>
        ))}
      </ul>
      <Lightbox photos={lightboxPhotos} index={openIndex} onChange={onOpen} />
    </Card>
  );
};

export default ActivityPhotos;
