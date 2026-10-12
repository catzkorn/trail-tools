import ElevationProfile from "@components/charts/elevation-profile";
import Loading from "@components/loading";
import type { MapFocus } from "@components/map/layers";
import {
  Story,
  TripStatsRow,
  mapPhotos,
  profileMarkers,
  profilePoints,
  trackPositions,
  tripKicker,
} from "@components/trips/trip-bits";
import TripMap from "@components/trips/trip-map";
import Lightbox from "@components/ui/lightbox";
import QueryError from "@components/ui/query-error";
import { Code } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { MapPinIcon, PencilSquareIcon } from "@heroicons/react/24/outline";
import { getPublicTripReport } from "gen/trips/v1/trips-TripReportService_connectquery";
import type { TripPhoto } from "gen/trips/v1/trips_pb";
import type { LatLngTuple } from "leaflet";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router";

const NotAvailable: React.FC = () => (
  <div className="py-16 text-center">
    <h1 className="text-xl font-semibold">Trip report not found</h1>
    <p className="mt-2 text-gray-600">
      This trip report doesn&apos;t exist or isn&apos;t published.
    </p>
  </div>
);

interface PhotoFigureProps {
  photo: TripPhoto;
  // The photo's number, as on the map.
  number: number;
  selected: boolean;
  // Hero photos are cropped to a wide frame.
  hero?: boolean;
  onOpen: () => void;
  onHover: (id: string | null) => void;
  onShowOnMap: (photo: TripPhoto) => void;
}

// PhotoFigure shows a photo with its number and caption.
const PhotoFigure: React.FC<PhotoFigureProps> = ({
  photo,
  number,
  selected,
  hero = false,
  onOpen,
  onHover,
  onShowOnMap,
}) => (
  <figure
    id={`photo-${photo.id}`}
    className={`scroll-mt-4 rounded-2xl transition-shadow ${
      selected ? "ring-4 ring-amber-400 ring-offset-4 ring-offset-slate-50" : ""
    }`}
    onMouseEnter={() => {
      onHover(photo.id);
    }}
    onMouseLeave={() => {
      onHover(null);
    }}
  >
    <button
      type="button"
      className={`block cursor-zoom-in rounded-2xl focus:outline-none focus-visible:ring-4 focus-visible:ring-emerald-500 focus-visible:ring-offset-2 ${hero ? "w-full" : "mx-auto w-fit"}`}
      onClick={onOpen}
      aria-label={`Open photo ${number}`}
    >
      <img
        src={photo.url}
        alt={photo.caption}
        width={photo.width}
        height={photo.height}
        loading={hero ? "eager" : "lazy"}
        className={
          hero
            ? "max-h-[70vh] w-full rounded-2xl bg-gray-200 object-cover shadow-sm"
            : "mx-auto h-auto max-h-[85vh] w-auto max-w-full rounded-2xl bg-gray-200 shadow-sm"
        }
      />
    </button>
    <figcaption className="mt-2 flex items-start gap-2 px-1 text-sm text-gray-600">
      <span className="mt-0.5 inline-flex size-5 shrink-0 items-center justify-center rounded-full bg-emerald-700 text-[11px] font-bold text-white">
        {number}
      </span>
      <span className="grow">{photo.caption}</span>
      {photo.location !== undefined && (
        <button
          type="button"
          className="inline-flex shrink-0 items-center gap-1 text-emerald-700 hover:text-emerald-900"
          onClick={() => {
            onShowOnMap(photo);
          }}
        >
          <MapPinIcon className="size-4" />
          <span className="hidden sm:inline">Map</span>
        </button>
      )}
    </figcaption>
  </figure>
);

/**
 * PublicTripReportPage is the public, blog style page of a published trip
 * report, at /u/<handle>/<slug>. It works without signing in.
 */
const PublicTripReportPage: React.FC = () => {
  const { handle = "", slug = "" } = useParams();
  const { isPending, isError, data, error, refetch } = useQuery(
    getPublicTripReport,
    { handle, slug },
    {
      retry: (failureCount, err) =>
        err.code !== Code.NotFound && failureCount < 3,
    }
  );
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [mapFocus, setMapFocus] = useState<MapFocus | null>(null);
  const mapRef = useRef<HTMLDivElement>(null);

  const report = data?.report;
  const track = useMemo(() => trackPositions(data?.track ?? []), [data?.track]);
  const profile = useMemo(
    () => profilePoints(data?.track ?? []),
    [data?.track]
  );
  const photos = useMemo(() => report?.photos ?? [], [report?.photos]);
  const markers = useMemo(() => mapPhotos(photos), [photos]);
  const photoMarkers = useMemo(() => profileMarkers(photos), [photos]);

  useEffect(() => {
    if (report === undefined || data?.author === undefined) {
      return undefined;
    }
    const previous = document.title;
    document.title = `${report.title} · ${data.author.displayName}`;
    return () => {
      document.title = previous;
    };
  }, [report, data?.author]);

  if (isPending) {
    return <Loading />;
  }
  if (isError) {
    if (error.code === Code.NotFound) {
      return <NotAvailable />;
    }
    return (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  }
  const { author } = data;
  if (report === undefined || author === undefined) {
    return <NotAvailable />;
  }

  // Selecting a marker scrolls to its photo.
  const selectFromMap = (id: string) => {
    setSelectedId(id);
    document
      .querySelector(`#photo-${CSS.escape(id)}`)
      ?.scrollIntoView({ behavior: "smooth", block: "center" });
  };
  const showOnMap = (photo: TripPhoto) => {
    setLightboxIndex(null);
    setSelectedId(photo.id);
    if (photo.location !== undefined) {
      setMapFocus({
        key: Date.now(),
        position: [photo.location.lat, photo.location.lng],
      });
    }
    // On narrow screens the map is above the story.
    const rect = mapRef.current?.getBoundingClientRect();
    if (
      rect !== undefined &&
      (rect.bottom < 0 || rect.top > globalThis.innerHeight)
    ) {
      mapRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };
  const openPhoto = (index: number) => {
    const photo = photos[index];
    if (photo !== undefined) {
      setSelectedId(photo.id);
    }
    setLightboxIndex(index);
  };
  let hoverPosition: LatLngTuple | null = null;
  if (hoverIndex !== null) {
    hoverPosition = track.at(hoverIndex) ?? null;
  }
  const hasMap = track.length > 1 || markers.length > 0;
  const [cover] = photos;

  return (
    <article className="mx-auto flex w-full max-w-6xl flex-col gap-8 pb-16">
      {!report.published && (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <span>
            This is a preview of a draft. Only you can see it until you publish
            it.
          </span>
          <Link
            to={`/trip-reports/${report.id}`}
            className="inline-flex items-center gap-1 font-medium hover:underline"
          >
            <PencilSquareIcon className="size-4" />
            Edit
          </Link>
        </div>
      )}
      <header className="mx-auto flex max-w-3xl flex-col items-center gap-3 pt-4 text-center sm:pt-8">
        <p className="text-sm font-semibold tracking-wide text-emerald-700 uppercase">
          {tripKicker(report)}
        </p>
        <h1 className="text-3xl font-bold tracking-tight text-balance text-gray-900 sm:text-5xl">
          {report.title}
        </h1>
        <p className="text-gray-600">
          by{" "}
          <Link
            to={`/u/${author.handle}`}
            className="font-medium text-gray-900 hover:text-emerald-800 hover:underline"
          >
            {author.displayName}
          </Link>
        </p>
      </header>

      {cover !== undefined && (
        <PhotoFigure
          photo={cover}
          number={1}
          selected={cover.id === selectedId}
          hero
          onOpen={() => {
            openPhoto(0);
          }}
          onHover={setHoveredId}
          onShowOnMap={showOnMap}
        />
      )}

      <TripStatsRow stats={report.stats} />

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,26rem)]">
        {hasMap && (
          <aside
            ref={mapRef}
            className="scroll-mt-4 lg:col-start-2 lg:row-start-1"
          >
            <div className="flex flex-col gap-3 lg:sticky lg:top-4">
              <div className="overflow-hidden rounded-xl shadow-sm ring-1 ring-gray-200">
                <TripMap
                  track={track}
                  photos={markers}
                  selectedId={hoveredId ?? selectedId}
                  hoverPosition={hoverPosition}
                  onSelectPhoto={selectFromMap}
                  focus={mapFocus}
                  fitKey={report.id}
                  className="h-80 sm:h-96 lg:h-[50vh]"
                />
              </div>
              {profile.length > 1 && (
                <div className="rounded-xl bg-white p-3 shadow-sm ring-1 ring-gray-200">
                  <ElevationProfile
                    points={profile}
                    markers={photoMarkers}
                    height={160}
                    onHover={setHoverIndex}
                  />
                </div>
              )}
            </div>
          </aside>
        )}
        <div className="flex min-w-0 flex-col gap-10 lg:col-start-1 lg:row-start-1">
          <Story body={report.body} />
          {photos.length > 1 && (
            <div className="flex flex-col gap-8">
              {photos.slice(1).map((photo, index) => (
                <PhotoFigure
                  key={photo.id}
                  photo={photo}
                  number={index + 2}
                  selected={photo.id === selectedId}
                  onOpen={() => {
                    openPhoto(index + 1);
                  }}
                  onHover={setHoveredId}
                  onShowOnMap={showOnMap}
                />
              ))}
            </div>
          )}
        </div>
      </div>
      <footer className="border-t border-gray-200 pt-6 text-center text-sm text-gray-500">
        More trip reports by{" "}
        <Link
          to={`/u/${author.handle}`}
          className="font-medium text-emerald-700 hover:underline"
        >
          {author.displayName}
        </Link>
      </footer>
      <Lightbox
        photos={photos}
        index={lightboxIndex}
        onChange={setLightboxIndex}
        onShowOnMap={hasMap ? showOnMap : undefined}
      />
    </article>
  );
};

export default PublicTripReportPage;
