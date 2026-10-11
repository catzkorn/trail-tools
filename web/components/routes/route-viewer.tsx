import ElevationProfile from "@components/charts/elevation-profile";
import ExploreControls from "@components/map/explore-controls";
import {
  FitBounds,
  HoverMarker,
  PointMarker,
  PolylineLayer,
} from "@components/map/layers";
import MapView from "@components/map/map-view";
import Card from "@components/ui/card";
import ConfirmButton from "@components/ui/confirm-button";
import { ErrorText } from "@components/ui/query-error";
import { primaryButton, secondaryButton } from "@components/ui/styles";
import { useMutation } from "@connectrpc/connect-query";
import {
  ArrowDownTrayIcon,
  CloudIcon,
  PencilSquareIcon,
} from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import { deleteRoute } from "gen/routes/v1/routes-RouteService_connectquery";
import type { Route } from "gen/routes/v1/routes_pb";
import { RouteService } from "gen/routes/v1/routes_pb";
import { formatTimestamp } from "helpers/format";
import { routeSourceLabel } from "helpers/labels";
import { invalidateServices } from "helpers/query";
import type { LatLngTuple } from "leaflet";
import React, { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";

import { toTuples } from "./route-geometry";
import { RouteStatsPanel } from "./route-stats";

interface RouteViewerProps {
  route: Route;
  onEdit: () => void;
}

// RouteViewer shows a saved route with its map, profile and statistics.
const RouteViewer: React.FC<RouteViewerProps> = ({ route, onEdit }) => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const remove = useMutation(deleteRoute, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [RouteService]);
      void navigate("/routes");
    },
  });

  const { points } = route;
  const tuples = useMemo(() => toTuples(points), [points]);
  const [start] = tuples;
  const finish = tuples.at(-1);
  let hoverPosition: LatLngTuple | null = null;
  const hoverPoint = hoverIndex === null ? undefined : points.at(hoverIndex);
  if (hoverPoint !== undefined) {
    hoverPosition = [hoverPoint.lat, hoverPoint.lng];
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
      <div className="flex min-w-0 flex-col gap-3">
        <div className="overflow-hidden rounded-lg border border-gray-200">
          <MapView className="h-[55vh] min-h-80" rememberView>
            <ExploreControls />
            {tuples.length > 1 && (
              <PolylineLayer positions={tuples} color="#047857" weight={5} />
            )}
            {start !== undefined && (
              <PointMarker position={start} color="#16a34a" tooltip="Start" />
            )}
            {finish !== undefined && tuples.length > 1 && (
              <PointMarker position={finish} color="#dc2626" tooltip="Finish" />
            )}
            <HoverMarker position={hoverPosition} />
            <FitBounds positions={tuples} fitKey={route.id} />
          </MapView>
        </div>
        <Card title="Elevation profile">
          <ElevationProfile points={points} onHover={setHoverIndex} />
        </Card>
      </div>
      <div className="flex flex-col gap-4">
        <Card>
          <div className="flex flex-col gap-3">
            <div>
              <h1 className="text-xl font-bold text-gray-900">{route.name}</h1>
              <p className="text-xs text-gray-500">
                {routeSourceLabel(route.source)} · updated{" "}
                {formatTimestamp(route.updateTime)}
              </p>
            </div>
            {route.description !== "" && (
              <p className="text-sm whitespace-pre-line text-gray-700">
                {route.description}
              </p>
            )}
            <div className="flex flex-wrap gap-2">
              <button type="button" className={primaryButton} onClick={onEdit}>
                <PencilSquareIcon className="size-4" />
                Edit
              </button>
              <a
                href={`/routes/${route.id}.gpx`}
                className={secondaryButton}
                download
              >
                <ArrowDownTrayIcon className="size-4" />
                GPX
              </a>
              <Link
                to={`/forecast?route=${route.id}`}
                className={secondaryButton}
              >
                <CloudIcon className="size-4" />
                Weather forecast
              </Link>
              <ConfirmButton
                title="Delete route?"
                message={`This permanently deletes "${route.name}". Workouts and races using it lose their route.`}
                disabled={remove.isPending}
                onConfirm={() => {
                  remove.mutate({ id: route.id });
                }}
              >
                Delete
              </ConfirmButton>
            </div>
            <ErrorText error={remove.error} />
          </div>
        </Card>
        <Card title="Statistics">
          <RouteStatsPanel stats={route.stats} />
        </Card>
      </div>
    </div>
  );
};

export default RouteViewer;
