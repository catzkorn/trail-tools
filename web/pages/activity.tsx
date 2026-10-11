import ElevationProfile from "@components/charts/elevation-profile";
import type { ProfilePoint } from "@components/charts/elevation-profile";
import Loading from "@components/loading";
import {
  FitBounds,
  HoverMarker,
  PointMarker,
  PolylineLayer,
} from "@components/map/layers";
import MapView from "@components/map/map-view";
import Card from "@components/ui/card";
import { Badge, Stat } from "@components/ui/misc";
import QueryError, { ErrorText } from "@components/ui/query-error";
import { primaryButton, secondaryButton } from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  ArrowTopRightOnSquareIcon,
  MapIcon,
} from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import { getActivity } from "gen/activities/v1/activities-ActivityService_connectquery";
import type {
  Activity,
  ActivityStreams,
} from "gen/activities/v1/activities_pb";
import { createRoute } from "gen/routes/v1/routes-RouteService_connectquery";
import { RouteService, TrackConversion } from "gen/routes/v1/routes_pb";
import {
  formatDate,
  formatHoursMinutes,
  formatKm,
  formatMeters,
  formatSpeedAsPace,
  formatTimestamp,
} from "helpers/format";
import { sportLabel } from "helpers/labels";
import { decodePolyline } from "helpers/polyline";
import { invalidateServices } from "helpers/query";
import type { LatLngTuple } from "leaflet";
import React, { useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";

interface Track {
  positions: LatLngTuple[];
  // Profile points, with the index of the matching position.
  profile: ProfilePoint[];
  heartRate: number[];
}

/**
 * BuildTrack turns the activity streams into map positions and profile
 * points, falling back to the summary polyline without streams.
 */
const buildTrack = (
  activity: Activity,
  streams: ActivityStreams | undefined
): Track => {
  if (streams === undefined || streams.lat.length === 0) {
    return {
      heartRate: [],
      positions: decodePolyline(activity.summaryPolyline),
      profile: [],
    };
  }
  const positions: LatLngTuple[] = [];
  const profile: ProfilePoint[] = [];
  const heartRate: number[] = [];
  const hasAltitude = streams.altitudeMeters.length === streams.lat.length;
  const hasDistance = streams.distanceMeters.length === streams.lat.length;
  const hasHeartRate = streams.heartrateBpm.length === streams.lat.length;
  for (const [index, lat] of streams.lat.entries()) {
    positions.push([lat, streams.lng[index] ?? 0]);
    if (hasAltitude && hasDistance) {
      profile.push({
        distanceMeters: streams.distanceMeters[index] ?? 0,
        elevationMeters: streams.altitudeMeters[index] ?? 0,
      });
      heartRate.push(hasHeartRate ? (streams.heartrateBpm[index] ?? 0) : 0);
    }
  }
  return { heartRate, positions, profile };
};

const ActivityPage: React.FC = () => {
  const { id = "" } = useParams();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const { isPending, isError, data, error, refetch } = useQuery(getActivity, {
    id,
    includeStreams: true,
    maxStreamPoints: 2000,
  });
  const makeRoute = useMutation(createRoute, {
    onSuccess: async (response) => {
      await invalidateServices(queryClient, [RouteService]);
      if (response.route !== undefined) {
        void navigate(`/routes/${response.route.id}`);
      }
    },
  });
  const track = useMemo(
    () =>
      data?.activity === undefined
        ? undefined
        : buildTrack(data.activity, data.streams),
    [data]
  );

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
  const { activity } = data;
  if (activity === undefined || track === undefined) {
    return <p>Activity not found.</p>;
  }
  const [start] = track.positions;
  const finish = track.positions.at(-1);
  let hoverPosition: LatLngTuple | null = null;
  if (hoverIndex !== null) {
    hoverPosition = track.positions.at(hoverIndex) ?? null;
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <Link
            to="/activities"
            className="text-sm text-gray-500 hover:underline"
          >
            Activities
          </Link>
          <h1 className="text-2xl font-bold text-gray-900">{activity.name}</h1>
          <p className="flex flex-wrap items-center gap-2 text-sm text-gray-600">
            {formatDate(activity.startDateLocal)}
            <span className="text-gray-400">
              {formatTimestamp(activity.startTime)}
            </span>
            <Badge>{sportLabel(activity.sportType)}</Badge>
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {activity.stravaId !== "" && (
            <a
              href={`https://www.strava.com/activities/${activity.stravaId}`}
              target="_blank"
              rel="noreferrer"
              className={secondaryButton}
            >
              <ArrowTopRightOnSquareIcon className="size-4" />
              View on Strava
            </a>
          )}
          {track.positions.length > 1 && (
            <button
              type="button"
              className={primaryButton}
              disabled={makeRoute.isPending}
              onClick={() => {
                makeRoute.mutate({
                  conversion: TrackConversion.SNAP,
                  name: activity.name,
                  source: { case: "activityId", value: activity.id },
                });
              }}
            >
              <MapIcon className="size-4" />
              {makeRoute.isPending
                ? "Snapping to trails…"
                : "Create route from this activity"}
            </button>
          )}
        </div>
      </div>
      <ErrorText error={makeRoute.error} />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div className="flex min-w-0 flex-col gap-4">
          {track.positions.length > 1 ? (
            <div className="overflow-hidden rounded-lg border border-gray-200">
              <MapView className="h-[50vh] min-h-72">
                <PolylineLayer
                  positions={track.positions}
                  color="#ea580c"
                  weight={4}
                />
                {start !== undefined && (
                  <PointMarker
                    position={start}
                    color="#16a34a"
                    tooltip="Start"
                  />
                )}
                {finish !== undefined && (
                  <PointMarker
                    position={finish}
                    color="#dc2626"
                    tooltip="Finish"
                  />
                )}
                <HoverMarker position={hoverPosition} />
                <FitBounds positions={track.positions} fitKey={activity.id} />
              </MapView>
            </div>
          ) : (
            <p className="rounded-md bg-white p-4 text-sm text-gray-500 ring-1 ring-gray-200">
              This activity has no GPS track.
            </p>
          )}
          {track.profile.length > 1 ? (
            <Card title="Elevation and heart rate">
              <ElevationProfile
                points={track.profile}
                secondary={track.heartRate}
                secondaryLabel="bpm"
                onHover={setHoverIndex}
              />
            </Card>
          ) : (
            !activity.hasStreams && (
              <p className="text-sm text-gray-500">
                Detailed data for this activity has not been synced yet.
              </p>
            )
          )}
        </div>
        <Card title="Stats">
          <div className="grid grid-cols-2 gap-3">
            <Stat label="Distance" value={formatKm(activity.distanceMeters)} />
            <Stat
              label="Elevation gain"
              value={formatMeters(activity.elevationGainMeters)}
            />
            <Stat
              label="Moving time"
              value={formatHoursMinutes(activity.movingTimeSeconds)}
            />
            <Stat
              label="Elapsed time"
              value={formatHoursMinutes(activity.elapsedTimeSeconds)}
            />
            <Stat
              label="Average pace"
              value={formatSpeedAsPace(activity.averageSpeedMps)}
            />
            {activity.averageHeartrateBpm > 0 && (
              <Stat
                label="Heart rate"
                value={`${Math.round(activity.averageHeartrateBpm)} bpm`}
                sub={`max ${Math.round(activity.maxHeartrateBpm)} bpm`}
              />
            )}
            {activity.averageCadenceSpm > 0 && (
              <Stat
                label="Cadence"
                value={`${Math.round(activity.averageCadenceSpm)} spm`}
              />
            )}
            {activity.relativeEffort > 0 && (
              <Stat label="Relative effort" value={activity.relativeEffort} />
            )}
          </div>
          {activity.deviceName !== "" && (
            <p className="mt-3 text-xs text-gray-500">
              Recorded with {activity.deviceName}
            </p>
          )}
          {activity.description !== "" && (
            <p className="mt-3 text-sm whitespace-pre-line text-gray-700">
              {activity.description}
            </p>
          )}
        </Card>
      </div>
    </div>
  );
};

export default ActivityPage;
