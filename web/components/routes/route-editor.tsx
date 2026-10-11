import ElevationProfile from "@components/charts/elevation-profile";
import ExploreControls from "@components/map/explore-controls";
import {
  FitBounds,
  HoverMarker,
  MapClick,
  PolylineLayer,
} from "@components/map/layers";
import MapView from "@components/map/map-view";
import WaypointMarkers from "@components/map/waypoint-markers";
import Card from "@components/ui/card";
import { TextAreaField, TextField } from "@components/ui/fields";
import { ErrorText } from "@components/ui/query-error";
import {
  ghostButton,
  iconButton,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import { ConnectError, createClient } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import {
  ArrowUturnLeftIcon,
  ArrowsRightLeftIcon,
  TrashIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import type { Route, RoutePoint } from "gen/routes/v1/routes_pb";
import { RouteService, RoutingProfile } from "gen/routes/v1/routes_pb";
import { PROFILE_OPTIONS } from "helpers/labels";
import type { LatLng, LatLngTuple } from "leaflet";
import React, { useEffect, useMemo, useRef, useState } from "react";

import type { LatLngValue } from "./route-geometry";
import { insertIndex, toTuples } from "./route-geometry";
import { RouteStatsPanel } from "./route-stats";

export interface RouteDraft {
  name: string;
  description: string;
  waypoints: LatLngValue[];
  profile: RoutingProfile;
  // Whether the waypoints or profile changed from the initial values.
  geometryChanged: boolean;
}

interface RouteEditorProps {
  // The saved route, or undefined for a new route.
  route: Route | undefined;
  // The initial name of a new route.
  defaultName?: string;
  /**
   * Whether the waypoints can be edited. Routes imported from GPX or
   * activities only allow editing the name and description.
   */
  geometryEditable: boolean;
  saving: boolean;
  error: ConnectError | null;
  onSave: (draft: RouteDraft) => void;
  onCancel: () => void;
}

const roundCoordinate = (value: number): number =>
  Math.round(value * 1e6) / 1e6;

const toWaypoint = (position: LatLng): LatLngValue => ({
  lat: roundCoordinate(position.lat),
  lng: roundCoordinate(position.lng),
});

const reversed = (points: LatLngValue[]): LatLngValue[] => {
  const result: LatLngValue[] = [];
  for (const point of points) {
    result.unshift(point);
  }
  return result;
};

const EMPTY_WAYPOINTS: LatLngValue[] = [];
const EMPTY_POINTS: RoutePoint[] = [];

/**
 * RouteEditor edits a route's waypoints on a map, with a live preview of the
 * routed path, its statistics and elevation profile.
 */
const RouteEditor: React.FC<RouteEditorProps> = ({
  route,
  defaultName = "",
  geometryEditable,
  saving,
  error,
  onSave,
  onCancel,
}) => {
  const transport = useTransport();
  const client = useMemo(
    () => createClient(RouteService, transport),
    [transport]
  );
  const initialWaypoints = useMemo(
    () =>
      route === undefined
        ? EMPTY_WAYPOINTS
        : route.waypoints.map((point) => ({ lat: point.lat, lng: point.lng })),
    [route]
  );
  const initialProfile =
    route === undefined || route.profile === RoutingProfile.UNSPECIFIED
      ? RoutingProfile.TRAIL
      : route.profile;

  const [name, setName] = useState(route?.name ?? defaultName);
  const [description, setDescription] = useState(route?.description ?? "");
  const [waypoints, setWaypoints] = useState<LatLngValue[]>(initialWaypoints);
  const [history, setHistory] = useState<LatLngValue[][]>([]);
  const [profile, setProfile] = useState<RoutingProfile>(initialProfile);
  const [preview, setPreview] = useState<Route | undefined>(route);
  const [previewError, setPreviewError] = useState("");
  const [previewing, setPreviewing] = useState(false);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [validation, setValidation] = useState("");
  const requestRef = useRef(0);

  // Recompute the preview when the waypoints or profile change, debounced.
  // Responses to outdated requests are ignored.
  useEffect(() => {
    if (!geometryEditable) {
      return undefined;
    }
    if (waypoints === initialWaypoints && profile === initialProfile) {
      return undefined;
    }
    requestRef.current += 1;
    const requestId = requestRef.current;
    if (waypoints.length < 2) {
      return undefined;
    }
    const timer = globalThis.setTimeout(() => {
      setPreviewing(true);
      client
        .previewRoute({ includePoints: true, profile, waypoints })
        .then((response) => {
          if (requestRef.current !== requestId) {
            return;
          }
          setPreview(response.route);
          setPreviewError("");
          setPreviewing(false);
        })
        .catch((error_: unknown) => {
          if (requestRef.current !== requestId) {
            return;
          }
          setPreviewError(ConnectError.from(error_).rawMessage);
          setPreviewing(false);
        });
    }, 300);
    return () => {
      globalThis.clearTimeout(timer);
    };
  }, [
    client,
    waypoints,
    profile,
    geometryEditable,
    initialWaypoints,
    initialProfile,
  ]);

  const changeWaypoints = (next: LatLngValue[]) => {
    setHistory([...history.slice(-49), waypoints]);
    setWaypoints(next);
  };

  const undo = () => {
    const previous = history.at(-1);
    if (previous === undefined) {
      return;
    }
    setHistory(history.slice(0, -1));
    setWaypoints(previous);
  };

  const removeWaypoint = (index: number) => {
    changeWaypoints(waypoints.filter((_point, position) => position !== index));
  };

  // Without two waypoints there is no route, whatever the last preview was.
  const hasRoute = !geometryEditable || waypoints.length >= 2;
  const shownPreview = hasRoute ? preview : undefined;
  const points = shownPreview?.points ?? EMPTY_POINTS;
  const pathTuples = useMemo(() => toTuples(points), [points]);
  const waypointTuples = useMemo(() => toTuples(waypoints), [waypoints]);
  const fitTuples = pathTuples.length > 0 ? pathTuples : waypointTuples;
  let hoverPosition: LatLngTuple | null = null;
  const hoverPoint = hoverIndex === null ? undefined : points.at(hoverIndex);
  if (hoverPoint !== undefined) {
    hoverPosition = [hoverPoint.lat, hoverPoint.lng];
  }

  const save = () => {
    if (name.trim() === "") {
      setValidation("Give the route a name.");
      return;
    }
    if (geometryEditable && waypoints.length < 2) {
      setValidation("Add at least two waypoints by clicking on the map.");
      return;
    }
    setValidation("");
    onSave({
      description,
      geometryChanged:
        waypoints !== initialWaypoints || profile !== initialProfile,
      name: name.trim(),
      profile,
      waypoints,
    });
  };

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
      <div className="flex min-w-0 flex-col gap-3">
        {geometryEditable && (
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="text-gray-600">Routing:</span>
            <div className="inline-flex overflow-hidden rounded-md ring-1 ring-gray-300">
              {PROFILE_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  className={`px-3 py-1 text-sm ${profile === option.value ? "bg-emerald-700 text-white" : "bg-white text-gray-700 hover:bg-gray-50"}`}
                  onClick={() => {
                    setProfile(option.value);
                  }}
                >
                  {option.label}
                </button>
              ))}
            </div>
            <span className="text-xs text-gray-500">
              Click the map to add a waypoint, drag to move, click a waypoint to
              remove it, click the path to insert one.
            </span>
          </div>
        )}
        <div className="overflow-hidden rounded-lg border border-gray-200">
          <MapView className="h-[55vh] min-h-80" rememberView>
            <ExploreControls />
            {pathTuples.length > 1 && (
              <PolylineLayer
                positions={pathTuples}
                color="#047857"
                weight={5}
                onClick={
                  geometryEditable
                    ? (position) => {
                        const index = insertIndex(points, waypoints, position);
                        const next = [...waypoints];
                        next.splice(index, 0, toWaypoint(position));
                        changeWaypoints(next);
                      }
                    : undefined
                }
              />
            )}
            {geometryEditable && (
              <>
                <MapClick
                  onClick={(position) => {
                    changeWaypoints([...waypoints, toWaypoint(position)]);
                  }}
                />
                <WaypointMarkers
                  waypoints={waypoints}
                  onMove={(index, position) => {
                    const next = [...waypoints];
                    next[index] = toWaypoint(position);
                    changeWaypoints(next);
                  }}
                  onRemove={removeWaypoint}
                />
              </>
            )}
            <HoverMarker position={hoverPosition} />
            {route !== undefined && (
              <FitBounds positions={fitTuples} fitKey={route.id} />
            )}
          </MapView>
        </div>
        <Card title="Elevation profile">
          {hasRoute && previewing && (
            <p className="mb-1 text-xs text-gray-500">Updating route…</p>
          )}
          {hasRoute && previewError !== "" && (
            <p className="mb-2 rounded bg-red-50 p-2 text-sm text-red-800">
              {previewError}
            </p>
          )}
          <ElevationProfile points={points} onHover={setHoverIndex} />
        </Card>
      </div>
      <div className="flex flex-col gap-4">
        <Card title={route === undefined ? "New route" : "Edit route"}>
          <form
            className="flex flex-col gap-3"
            onSubmit={(event) => {
              event.preventDefault();
              save();
            }}
          >
            <TextField
              label="Name"
              value={name}
              onChange={setName}
              placeholder="e.g. Three peaks loop"
              required
            />
            <TextAreaField
              label="Description"
              value={description}
              onChange={setDescription}
              placeholder="Purpose, highlights, navigation notes"
            />
            {validation !== "" && (
              <p className="text-sm text-red-700">{validation}</p>
            )}
            <ErrorText error={error} />
            <div className="flex gap-2">
              <button type="submit" className={primaryButton} disabled={saving}>
                {saving ? "Saving…" : "Save route"}
              </button>
              <button
                type="button"
                className={secondaryButton}
                onClick={onCancel}
              >
                Cancel
              </button>
            </div>
          </form>
        </Card>
        <Card title="Statistics">
          <RouteStatsPanel stats={shownPreview?.stats} />
        </Card>
        {geometryEditable && (
          <Card
            title={`Waypoints (${waypoints.length})`}
            actions={
              <>
                <button
                  type="button"
                  className={ghostButton}
                  disabled={history.length === 0}
                  onClick={undo}
                  title="Undo"
                >
                  <ArrowUturnLeftIcon className="size-4" />
                  Undo
                </button>
                <button
                  type="button"
                  className={ghostButton}
                  disabled={waypoints.length < 2}
                  title="Reverse the route"
                  onClick={() => {
                    changeWaypoints(reversed(waypoints));
                  }}
                >
                  <ArrowsRightLeftIcon className="size-4" />
                </button>
                <button
                  type="button"
                  className={ghostButton}
                  disabled={waypoints.length === 0}
                  title="Remove all waypoints"
                  onClick={() => {
                    changeWaypoints([]);
                  }}
                >
                  <TrashIcon className="size-4" />
                </button>
              </>
            }
          >
            {waypoints.length === 0 ? (
              <p className="text-sm text-gray-500">
                Click on the map to place the start.
              </p>
            ) : (
              <ol className="flex max-h-72 flex-col gap-1 overflow-y-auto text-sm">
                {waypoints.map((waypoint, index) => {
                  let label = `Waypoint ${index + 1}`;
                  if (index === 0) {
                    label = "Start";
                  } else if (index === waypoints.length - 1) {
                    label = "Finish";
                  }
                  return (
                    <li
                      key={`${waypoint.lat},${waypoint.lng}`}
                      className="flex items-center justify-between gap-2 rounded px-1 hover:bg-gray-50"
                    >
                      <span>
                        <span className="font-medium">{label}</span>{" "}
                        <span className="text-xs text-gray-500 tabular-nums">
                          {waypoint.lat.toFixed(4)}, {waypoint.lng.toFixed(4)}
                        </span>
                      </span>
                      <button
                        type="button"
                        className={iconButton}
                        aria-label={`Remove ${label}`}
                        onClick={() => {
                          removeWaypoint(index);
                        }}
                      >
                        <XMarkIcon className="size-4" />
                      </button>
                    </li>
                  );
                })}
              </ol>
            )}
          </Card>
        )}
      </div>
    </div>
  );
};

export default RouteEditor;
