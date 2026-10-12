import ElevationProfile from "@components/charts/elevation-profile";
import Loading from "@components/loading";
import PhotoUploader from "@components/trips/photo-uploader";
import StravaImport from "@components/trips/strava-import";
import {
  TripStatsRow,
  locationSourceLabel,
  mapPhotos,
  profileMarkers,
  profilePoints,
  trackPositions,
} from "@components/trips/trip-bits";
import TripMap from "@components/trips/trip-map";
import Card from "@components/ui/card";
import ConfirmButton from "@components/ui/confirm-button";
import { SelectField, TextField } from "@components/ui/fields";
import type { SelectOption } from "@components/ui/fields";
import { Badge } from "@components/ui/misc";
import QueryError, { ErrorText } from "@components/ui/query-error";
import {
  ghostButton,
  iconButton,
  inputClass,
  labelClass,
  linkClass,
  primaryButton,
  secondaryButton,
  secondaryDangerButton,
} from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  ArrowDownIcon,
  ArrowTopRightOnSquareIcon,
  ArrowUpIcon,
  EyeSlashIcon,
  GlobeAltIcon,
  MapPinIcon,
  TrashIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  deleteTripPhoto,
  deleteTripReport,
  getProfile,
  getTripReport,
  publishTripReport,
  reorderTripPhotos,
  unpublishTripReport,
  updateTripPhoto,
  updateTripReport,
} from "gen/trips/v1/trips-TripReportService_connectquery";
import type { TripPhoto, TripReport } from "gen/trips/v1/trips_pb";
import { LocationSource, TripReportService } from "gen/trips/v1/trips_pb";
import { invalidateServices } from "helpers/query";
import { isValidSlug, slugify } from "helpers/slug";
import type { LatLng, LatLngTuple } from "leaflet";
import React, { useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";

// How much of the track around its start and finish to hide publicly.
const HIDDEN_ENDS_OPTIONS: SelectOption[] = [
  { label: "Nothing: show the whole track", value: "0" },
  { label: "250 m around the start and finish", value: "250" },
  { label: "500 m around the start and finish", value: "500" },
  { label: "1 km around the start and finish", value: "1000" },
  { label: "2 km around the start and finish", value: "2000" },
];

interface ReportFormProps {
  report: TripReport;
  handle: string | undefined;
  onSaved: () => Promise<void>;
}

/**
 * ReportForm edits the title, address, date and story of a report. It is
 * keyed by the report, so its state starts from the saved report.
 */
const ReportForm: React.FC<ReportFormProps> = ({ report, handle, onSaved }) => {
  const [title, setTitle] = useState(report.title);
  const [slug, setSlug] = useState(report.slug);
  // The slug follows the title until it is edited by hand.
  const [slugEdited, setSlugEdited] = useState(
    report.slug !== slugify(report.title)
  );
  const [date, setDate] = useState(report.date);
  const [body, setBody] = useState(report.body);
  const [hiddenEnds, setHiddenEnds] = useState(String(report.hiddenEndsMeters));
  const update = useMutation(updateTripReport, { onSuccess: onSaved });

  const trimmedTitle = title.trim();
  const dirty =
    trimmedTitle !== report.title ||
    slug !== report.slug ||
    date !== report.date ||
    body !== report.body ||
    Number(hiddenEnds) !== report.hiddenEndsMeters;
  const slugValid = isValidSlug(slug);
  const save = () => {
    update.mutate({
      body: body === report.body ? undefined : body,
      date: date === report.date ? undefined : date,
      hiddenEndsMeters:
        Number(hiddenEnds) === report.hiddenEndsMeters
          ? undefined
          : Number(hiddenEnds),
      id: report.id,
      slug: slug === report.slug ? undefined : slug,
      title: trimmedTitle === report.title ? undefined : trimmedTitle,
    });
  };

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        save();
      }}
    >
      <TextField
        label="Title"
        value={title}
        required
        onChange={(value) => {
          setTitle(value);
          if (!slugEdited && !report.published) {
            const next = slugify(value);
            if (next !== "") {
              setSlug(next);
            }
          }
        }}
      />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-[minmax(0,1fr)_10rem]">
        <label className="flex min-w-0 flex-col gap-1">
          <span className={labelClass}>Address</span>
          <div className="flex items-center rounded-md border border-gray-300 bg-white shadow-sm focus-within:border-emerald-600 focus-within:ring-1 focus-within:ring-emerald-600">
            <span className="shrink-0 truncate pl-2.5 text-sm text-gray-500">
              /u/{handle ?? "<handle>"}/
            </span>
            <input
              value={slug}
              disabled={report.published}
              className="block w-full min-w-0 rounded-md border-0 bg-transparent py-1.5 pr-2.5 pl-0.5 text-sm text-gray-900 focus:outline-none disabled:text-gray-500"
              onChange={(event) => {
                setSlug(event.target.value.toLowerCase());
                setSlugEdited(true);
              }}
            />
          </div>
          {report.published ? (
            <span className="text-xs text-gray-500">
              The address of a published report can&apos;t change, so links to
              it keep working. Unpublish the report to change it.
            </span>
          ) : (
            !slugValid && (
              <span className="text-xs text-red-700">
                Use lowercase letters and digits separated by single dashes.
              </span>
            )
          )}
        </label>
        <TextField
          label="Date"
          type="date"
          value={date}
          required
          onChange={setDate}
        />
      </div>
      <label className="flex flex-col gap-1">
        <span className={labelClass}>Story</span>
        <textarea
          value={body}
          rows={14}
          placeholder="How did it go? Separate paragraphs with a blank line."
          className={`${inputClass} text-base leading-7`}
          onChange={(event) => {
            setBody(event.target.value);
          }}
        />
      </label>
      {report.hasTrack && (
        <SelectField
          label="Hide on the public map"
          value={hiddenEnds}
          onChange={setHiddenEnds}
          options={HIDDEN_ENDS_OPTIONS}
          className="sm:max-w-sm"
        />
      )}
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="submit"
          className={primaryButton}
          disabled={
            !dirty ||
            update.isPending ||
            trimmedTitle === "" ||
            !slugValid ||
            date === ""
          }
        >
          {update.isPending ? "Saving…" : "Save"}
        </button>
        <span className="text-sm text-gray-500">
          {dirty ? "Unsaved changes" : "All changes saved"}
        </span>
      </div>
      <ErrorText error={update.error} />
    </form>
  );
};

interface PhotoRowProps {
  photo: TripPhoto;
  number: number;
  count: number;
  selected: boolean;
  placing: boolean;
  onSelect: () => void;
  onPlace: () => void;
  onMove: (delta: number) => void;
  onChanged: () => Promise<void>;
}

// PhotoRow edits a photo: its caption, place on the map and order.
const PhotoRow: React.FC<PhotoRowProps> = ({
  photo,
  number,
  count,
  selected,
  placing,
  onSelect,
  onPlace,
  onMove,
  onChanged,
}) => {
  const [caption, setCaption] = useState(photo.caption);
  const update = useMutation(updateTripPhoto, { onSuccess: onChanged });
  const remove = useMutation(deleteTripPhoto, { onSuccess: onChanged });
  const located = photo.location !== undefined;
  let placeLabel = "Place on map";
  if (placing) {
    placeLabel = "Click the map…";
  } else if (located) {
    placeLabel = "Move";
  }

  return (
    <li
      className={`flex gap-3 rounded-lg p-2 ${selected ? "bg-amber-50 ring-2 ring-amber-300" : ""}`}
    >
      <button
        type="button"
        className="relative shrink-0 self-start"
        onClick={onSelect}
        aria-label={`Select photo ${number}`}
      >
        <img
          src={photo.thumbnailUrl}
          alt=""
          className="size-24 rounded-md object-cover sm:size-28"
        />
        <span className="absolute -top-1.5 -left-1.5 inline-flex size-5 items-center justify-center rounded-full border-2 border-white bg-emerald-700 text-[10px] font-bold text-white">
          {number}
        </span>
      </button>
      <div className="flex min-w-0 grow flex-col gap-1.5">
        <textarea
          value={caption}
          rows={2}
          placeholder="Caption"
          className={`${inputClass} resize-none`}
          onFocus={onSelect}
          onChange={(event) => {
            setCaption(event.target.value);
          }}
          onBlur={() => {
            if (caption.trim() !== photo.caption) {
              update.mutate({ caption, id: photo.id });
            }
          }}
        />
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span
            className={`inline-flex items-center gap-1 text-xs ${located ? "text-gray-500" : "text-amber-700"}`}
          >
            <MapPinIcon className="size-3.5" />
            {locationSourceLabel(photo.locationSource)}
          </span>
          <span className="ml-auto flex items-center">
            <button
              type="button"
              className={`${ghostButton} text-xs`}
              onClick={onPlace}
            >
              {placeLabel}
            </button>
            {located && (
              <button
                type="button"
                className={iconButton}
                title="Remove from the map"
                aria-label="Remove from the map"
                disabled={update.isPending}
                onClick={() => {
                  update.mutate({ clearLocation: true, id: photo.id });
                }}
              >
                <XMarkIcon className="size-4" />
              </button>
            )}
            <button
              type="button"
              className={iconButton}
              aria-label="Move up"
              disabled={number === 1}
              onClick={() => {
                onMove(-1);
              }}
            >
              <ArrowUpIcon className="size-4" />
            </button>
            <button
              type="button"
              className={iconButton}
              aria-label="Move down"
              disabled={number === count}
              onClick={() => {
                onMove(1);
              }}
            >
              <ArrowDownIcon className="size-4" />
            </button>
            <ConfirmButton
              className={`${iconButton} hover:text-red-600`}
              title="Delete photo?"
              message="The photo is removed from the report."
              disabled={remove.isPending}
              onConfirm={() => {
                remove.mutate({ id: photo.id });
              }}
            >
              <TrashIcon className="size-4" aria-label="Delete photo" />
            </ConfirmButton>
          </span>
        </div>
        <ErrorText error={update.error ?? remove.error} />
      </div>
    </li>
  );
};

// TripReportPage edits a trip report: its story, photos and publication.
const TripReportPage: React.FC = () => {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { isPending, isError, data, error, refetch } = useQuery(getTripReport, {
    id,
    includeTrack: true,
  });
  const profileQuery = useQuery(getProfile);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [placingId, setPlacingId] = useState<string | null>(null);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);

  const invalidate = async () => {
    await invalidateServices(queryClient, [TripReportService]);
  };
  const publish = useMutation(publishTripReport, { onSuccess: invalidate });
  const unpublish = useMutation(unpublishTripReport, { onSuccess: invalidate });
  const remove = useMutation(deleteTripReport, {
    onSuccess: async () => {
      await navigate("/trip-reports");
      await invalidate();
    },
  });
  const movePhoto = useMutation(updateTripPhoto, { onSuccess: invalidate });
  const reorder = useMutation(reorderTripPhotos, { onSuccess: invalidate });

  /**
   * Derived data is memoized, so the map isn't redrawn while hovering the
   * elevation profile.
   */
  const photos = useMemo(
    () => data?.report?.photos ?? [],
    [data?.report?.photos]
  );
  const track = useMemo(() => trackPositions(data?.track ?? []), [data?.track]);
  const profile = useMemo(
    () => profilePoints(data?.track ?? []),
    [data?.track]
  );
  const markers = useMemo(() => mapPhotos(photos), [photos]);
  const photoMarkers = useMemo(() => profileMarkers(photos), [photos]);

  if (isPending) {
    return <Loading />;
  }
  if (isError) {
    return (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  }
  const { report } = data;
  if (report === undefined) {
    return <p>Trip report not found.</p>;
  }
  const handle = profileQuery.data?.profile?.handle;
  const placing = photos.find((photo) => photo.id === placingId);
  const placingNumber = placing === undefined ? 0 : photos.indexOf(placing) + 1;
  let hoverPosition: LatLngTuple | null = null;
  if (hoverIndex !== null) {
    hoverPosition = track.at(hoverIndex) ?? null;
  }
  const publicPath = handle === undefined ? "" : `/u/${handle}/${report.slug}`;

  const place = (photoId: string, position: LatLng) => {
    movePhoto.mutate({
      id: photoId,
      location: { lat: position.lat, lng: position.wrap().lng },
    });
  };
  const move = (index: number, delta: number) => {
    const ids = photos.map((photo) => photo.id);
    const target = index + delta;
    const [moved] = ids.splice(index, 1);
    if (moved === undefined || target < 0 || target > ids.length) {
      return;
    }
    ids.splice(target, 0, moved);
    reorder.mutate({ photoIds: ids, reportId: report.id });
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <Link
            to="/trip-reports"
            className="text-sm text-gray-500 hover:underline"
          >
            Trip reports
          </Link>
          <h1 className="flex flex-wrap items-center gap-2 text-2xl font-bold text-gray-900">
            {report.title}
            {report.published ? (
              <Badge className="bg-emerald-50 text-emerald-800 ring-emerald-300">
                Published
              </Badge>
            ) : (
              <Badge>Draft</Badge>
            )}
          </h1>
          {report.activityId !== "" && (
            <Link
              to={`/activities/${report.activityId}`}
              className="text-sm text-gray-500 hover:underline"
            >
              View the activity
            </Link>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {publicPath !== "" && (
            <Link to={publicPath} className={secondaryButton}>
              <ArrowTopRightOnSquareIcon className="size-4" />
              {report.published ? "View" : "Preview"}
            </Link>
          )}
          {report.published ? (
            <button
              type="button"
              className={secondaryButton}
              disabled={unpublish.isPending}
              onClick={() => {
                unpublish.mutate({ id: report.id });
              }}
            >
              <EyeSlashIcon className="size-4" />
              Unpublish
            </button>
          ) : (
            <button
              type="button"
              className={primaryButton}
              disabled={handle === undefined || publish.isPending}
              onClick={() => {
                publish.mutate({ id: report.id });
              }}
            >
              <GlobeAltIcon className="size-4" />
              Publish
            </button>
          )}
          <ConfirmButton
            className={secondaryDangerButton}
            title="Delete trip report?"
            message="The report and its photos are deleted. This can't be undone."
            disabled={remove.isPending}
            onConfirm={() => {
              remove.mutate({ id: report.id });
            }}
          >
            <TrashIcon className="size-4" />
            Delete
          </ConfirmButton>
        </div>
      </div>
      {profileQuery.data !== undefined && handle === undefined && (
        <p className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-900">
          To publish, first choose a public handle for your reports' address in{" "}
          <Link to="/settings" className={linkClass}>
            Settings
          </Link>
          .
        </p>
      )}
      {report.published && report.url !== "" && (
        <p className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">
          Published at{" "}
          <a href={report.url} className="font-medium break-all underline">
            {report.url}
          </a>
          . Anyone with the link can read it, and search engines can find it.
        </p>
      )}
      <ErrorText
        error={
          publish.error ?? unpublish.error ?? remove.error ?? reorder.error
        }
      />
      <TripStatsRow stats={report.stats} />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,28rem)]">
        <div className="flex min-w-0 flex-col gap-4">
          <Card title="Story">
            <ReportForm
              key={report.id}
              report={report}
              handle={handle}
              onSaved={invalidate}
            />
          </Card>
          <Card title={`Photos (${photos.length})`}>
            <div className="flex flex-col gap-4">
              <PhotoUploader reportId={report.id} onUploaded={invalidate} />
              {report.activityId !== "" && (
                <StravaImport reportId={report.id} onImported={invalidate} />
              )}
              {photos.length > 0 && (
                <ul className="flex flex-col gap-2">
                  {photos.map((photo, index) => (
                    <PhotoRow
                      key={photo.id}
                      photo={photo}
                      number={index + 1}
                      count={photos.length}
                      selected={photo.id === selectedId}
                      placing={photo.id === placingId}
                      onSelect={() => {
                        setSelectedId(photo.id);
                      }}
                      onPlace={() => {
                        setSelectedId(photo.id);
                        setPlacingId(photo.id === placingId ? null : photo.id);
                      }}
                      onMove={(delta) => {
                        move(index, delta);
                      }}
                      onChanged={invalidate}
                    />
                  ))}
                </ul>
              )}
            </div>
          </Card>
        </div>
        <div className="flex flex-col gap-3 lg:sticky lg:top-4 lg:self-start">
          <div className="overflow-hidden rounded-lg border border-gray-200">
            <TripMap
              track={track}
              photos={markers}
              selectedId={selectedId}
              hoverPosition={hoverPosition}
              onSelectPhoto={setSelectedId}
              onMovePhoto={place}
              hiddenEndsMeters={report.hiddenEndsMeters}
              onMapClick={
                placing === undefined
                  ? undefined
                  : (position) => {
                      place(placing.id, position);
                      setPlacingId(null);
                    }
              }
              fitKey={report.id}
              className="h-[45vh] min-h-72"
            />
          </div>
          {placing === undefined ? (
            <p className="text-xs text-gray-500">
              Drag a photo on the map to move it. Photos that aren&apos;t on the
              map yet can be placed from the list.
            </p>
          ) : (
            <p className="flex items-center justify-between gap-2 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900">
              Click the map where photo {placingNumber} was taken.
              <button
                type="button"
                className={ghostButton}
                onClick={() => {
                  setPlacingId(null);
                }}
              >
                Cancel
              </button>
            </p>
          )}
          <ErrorText error={movePhoto.error} />
          {profile.length > 1 && (
            <Card>
              <ElevationProfile
                points={profile}
                markers={photoMarkers}
                height={150}
                onHover={setHoverIndex}
              />
            </Card>
          )}
          {photos.some(
            (photo) => photo.locationSource === LocationSource.UNSPECIFIED
          ) && (
            <p className="text-xs text-amber-800">
              Some photos aren&apos;t on the map yet.
            </p>
          )}
        </div>
      </div>
    </div>
  );
};

export default TripReportPage;
