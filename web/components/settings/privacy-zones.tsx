import { create } from "@bufbuild/protobuf";
import Loading from "@components/loading";
import { PlaceSearch } from "@components/map/explore-controls";
import { FitBounds, MapClick } from "@components/map/layers";
import MapView, { useMap } from "@components/map/map-view";
import Card from "@components/ui/card";
import QueryError, { ErrorText } from "@components/ui/query-error";
import {
  ghostDangerButton,
  inputClass,
  labelClass,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import {
  createConnectQueryKey,
  useMutation,
  useQuery,
  useTransport,
} from "@connectrpc/connect-query";
import { TrashIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import { RaceReportService } from "gen/races/v1/race_reports_pb";
import { TripReportService } from "gen/trips/v1/trips_pb";
import {
  listPrivacyZones,
  setPrivacyZones,
} from "gen/users/v1/users-UserService_connectquery";
import type { PrivacyZone } from "gen/users/v1/users_pb";
import { ListPrivacyZonesResponseSchema } from "gen/users/v1/users_pb";
import { invalidateServices } from "helpers/query";
import type { LatLng, LatLngTuple } from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useEffectEvent, useMemo, useState } from "react";

// RADIUS_OPTIONS are the radiuses offered for a zone, in meters.
const RADIUS_OPTIONS = [100, 250, 500, 1000, 2000, 5000];

/**
 * RadiusOptions returns the radiuses offered for a zone: the usual ones, and
 * the zone's own if it was set to another, e.g. through the API.
 */
const radiusOptions = (radiusMeters: number): number[] =>
  RADIUS_OPTIONS.includes(radiusMeters)
    ? RADIUS_OPTIONS
    : [
        ...RADIUS_OPTIONS.filter((meters) => meters < radiusMeters),
        radiusMeters,
        ...RADIUS_OPTIONS.filter((meters) => meters > radiusMeters),
      ];

// RadiusLabel formats a radius, e.g. "500 m" or "2 km".
const radiusLabel = (meters: number): string =>
  meters >= 1000 ? `${meters / 1000} km` : `${meters} m`;

// DEFAULT_RADIUS_METERS is the radius of a new zone.
const DEFAULT_RADIUS_METERS = 500;
// MAX_ZONES is the most zones the server accepts.
const MAX_ZONES = 20;
// METERS_PER_DEGREE is about how many meters a degree of latitude is.
const METERS_PER_DEGREE = 111_320;

// Draft is a zone being edited, with a key for zones that have no ID yet.
interface Draft {
  key: string;
  name: string;
  lat: number;
  lng: number;
  radiusMeters: number;
}

const toDraft = (zone: PrivacyZone): Draft => ({
  key: zone.id,
  lat: zone.lat,
  lng: zone.lng,
  name: zone.name,
  radiusMeters: zone.radiusMeters,
});

const sameZones = (drafts: Draft[], zones: PrivacyZone[]): boolean =>
  drafts.length === zones.length &&
  drafts.every((draft, index) => {
    const zone = zones[index];
    return (
      zone !== undefined &&
      draft.name.trim() === zone.name &&
      draft.lat === zone.lat &&
      draft.lng === zone.lng &&
      draft.radiusMeters === zone.radiusMeters
    );
  });

/**
 * ZoneCorners returns the corners of the square around each zone, so the map
 * fits the zones' whole circles rather than their centers.
 */
const zoneCorners = (zones: Draft[]): LatLngTuple[] =>
  zones.flatMap((zone) => {
    const latDegrees = zone.radiusMeters / METERS_PER_DEGREE;
    const lngDegrees =
      latDegrees / Math.max(Math.cos((zone.lat * Math.PI) / 180), 0.01);
    return [
      [zone.lat - latDegrees, zone.lng - lngDegrees],
      [zone.lat + latDegrees, zone.lng + lngDegrees],
    ];
  });

interface ZoneLayerProps {
  zone: Draft;
  // The zone's number, shown on its marker.
  number: number;
  selected: boolean;
  onSelect: () => void;
  // Called when the zone's marker is dropped elsewhere.
  onMove: (position: LatLng) => void;
}

/**
 * ZoneLayer draws a zone as a shaded circle with a numbered marker at its
 * center, which can be dragged to move the zone.
 */
const ZoneLayer: React.FC<ZoneLayerProps> = ({
  zone,
  number,
  selected,
  onSelect,
  onMove,
}) => {
  const map = useMap();
  const { lat, lng, radiusMeters } = zone;
  const handleSelect = useEffectEvent(onSelect);
  const handleMove = useEffectEvent(onMove);
  useEffect(() => {
    const color = selected ? "#c2410c" : "#4b5563";
    const circle = leaflet
      .circle([lat, lng], {
        // Clicks on the circle select it rather than adding a zone.
        bubblingMouseEvents: false,
        color,
        dashArray: "4 4",
        fillOpacity: selected ? 0.25 : 0.15,
        radius: radiusMeters,
        weight: selected ? 2.5 : 1.5,
      })
      .addTo(map);
    const label = document.createElement("span");
    label.className = `flex size-6 cursor-move items-center justify-center rounded-full border-2 border-white text-xs font-semibold text-white shadow ${selected ? "bg-ember" : "bg-gray-600"}`;
    label.textContent = String(number);
    const marker = leaflet
      .marker([lat, lng], {
        draggable: true,
        icon: leaflet.divIcon({
          className: "",
          html: label,
          iconAnchor: [12, 12],
          iconSize: [24, 24],
        }),
        title: `Zone ${number}`,
      })
      .addTo(map);
    circle.on("click", () => {
      handleSelect();
    });
    marker.on("click", () => {
      handleSelect();
    });
    marker.on("drag", () => {
      circle.setLatLng(marker.getLatLng());
    });
    marker.on("dragend", () => {
      handleMove(marker.getLatLng());
    });
    return () => {
      circle.remove();
      marker.remove();
    };
  }, [map, lat, lng, radiusMeters, number, selected]);
  return null;
};

interface ZonesEditorProps {
  zones: PrivacyZone[];
}

/**
 * ZonesEditor edits the user's privacy zones on a map: clicking the map adds
 * a zone there, and dragging a zone's marker moves it. It is keyed by the
 * saved zones, so its drafts start from them.
 */
const ZonesEditor: React.FC<ZonesEditorProps> = ({ zones }) => {
  const queryClient = useQueryClient();
  const transport = useTransport();
  const [drafts, setDrafts] = useState(() => zones.map(toDraft));
  const [selectedKey, setSelectedKey] = useState("");
  const [nextKey, setNextKey] = useState(1);
  const save = useMutation(setPrivacyZones, {
    onSuccess: async (response) => {
      queryClient.setQueryData(
        createConnectQueryKey({
          cardinality: "finite",
          input: {},
          schema: listPrivacyZones,
          transport,
        }),
        create(ListPrivacyZonesResponseSchema, { zones: response.zones })
      );
      // Public reports hide other stretches of their tracks now.
      await invalidateServices(queryClient, [
        TripReportService,
        RaceReportService,
      ]);
    },
  });
  // The map fits the saved zones when it opens, and not as they're edited.
  const fitPositions = useMemo(() => zoneCorners(zones.map(toDraft)), [zones]);
  const dirty = !sameZones(drafts, zones);
  const full = drafts.length >= MAX_ZONES;

  const change = (key: string, fields: Partial<Draft>) => {
    setDrafts((current) =>
      current.map((draft) =>
        draft.key === key ? { ...draft, ...fields } : draft
      )
    );
  };
  const add = (position: LatLng) => {
    if (full) {
      return;
    }
    const key = `new-${nextKey}`;
    setNextKey(nextKey + 1);
    setDrafts([
      ...drafts,
      {
        key,
        lat: position.lat,
        lng: position.lng,
        name: drafts.length === 0 ? "Home" : "",
        radiusMeters: DEFAULT_RADIUS_METERS,
      },
    ]);
    setSelectedKey(key);
  };

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate({
          zones: drafts.map((draft) => ({
            lat: draft.lat,
            lng: draft.lng,
            name: draft.name.trim(),
            radiusMeters: draft.radiusMeters,
          })),
        });
      }}
    >
      <p className="text-sm text-gray-600">
        Your tracks and photos within privacy zones, like around your home, are
        hidden on your public trip and race reports. Click the map to add a
        zone, and drag its marker to move it.
      </p>
      <div className="overflow-hidden rounded-lg border border-gray-200">
        <MapView className="h-72" rememberView={zones.length === 0}>
          <div className="pointer-events-none absolute top-2.5 right-16.5 left-13 z-[999]">
            <PlaceSearch />
          </div>
          {drafts.map((draft, index) => (
            <ZoneLayer
              key={draft.key}
              zone={draft}
              number={index + 1}
              selected={draft.key === selectedKey}
              onSelect={() => {
                setSelectedKey(draft.key);
              }}
              onMove={(position) => {
                change(draft.key, { lat: position.lat, lng: position.lng });
                setSelectedKey(draft.key);
              }}
            />
          ))}
          {!full && <MapClick onClick={add} />}
          <FitBounds positions={fitPositions} fitKey="zones" />
        </MapView>
      </div>
      {full && (
        <p className="text-sm text-gray-600">
          You have {MAX_ZONES} zones, the most there can be. Remove one to add
          another.
        </p>
      )}
      {drafts.length > 0 && (
        <ul className="flex flex-col divide-y divide-gray-100">
          {drafts.map((draft, index) => (
            <li
              key={draft.key}
              className={`flex flex-wrap items-end gap-3 py-2 ${draft.key === selectedKey ? "bg-brand-50/60" : ""}`}
              onFocus={() => {
                setSelectedKey(draft.key);
              }}
            >
              <span className="mb-2 flex size-6 shrink-0 items-center justify-center rounded-full bg-gray-600 text-xs font-semibold text-white tabular-nums">
                {index + 1}
              </span>
              <label className="flex min-w-32 grow flex-col gap-1">
                <span className={labelClass}>Name</span>
                <input
                  className={inputClass}
                  value={draft.name}
                  maxLength={100}
                  placeholder="e.g. Home"
                  onChange={(event) => {
                    change(draft.key, { name: event.target.value });
                  }}
                />
              </label>
              <label className="flex flex-col gap-1">
                <span className={labelClass}>Radius</span>
                <select
                  className={inputClass}
                  value={draft.radiusMeters}
                  onChange={(event) => {
                    change(draft.key, {
                      radiusMeters: Number(event.target.value),
                    });
                  }}
                >
                  {radiusOptions(draft.radiusMeters).map((meters) => (
                    <option key={meters} value={meters}>
                      {radiusLabel(meters)}
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                className={`${ghostDangerButton} mb-0.5`}
                aria-label={`Remove zone ${index + 1}`}
                title="Remove"
                onClick={() => {
                  setDrafts(drafts.filter((other) => other.key !== draft.key));
                }}
              >
                <TrashIcon className="size-4" />
              </button>
            </li>
          ))}
        </ul>
      )}
      {(dirty || save.error !== null) && (
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="submit"
            className={primaryButton}
            disabled={!dirty || save.isPending}
          >
            {save.isPending ? "Saving…" : "Save zones"}
          </button>
          <button
            type="button"
            className={secondaryButton}
            disabled={save.isPending}
            onClick={() => {
              setDrafts(zones.map(toDraft));
              setSelectedKey("");
            }}
          >
            Discard
          </button>
          <ErrorText error={save.error} />
        </div>
      )}
    </form>
  );
};

// PrivacyZonesCard edits the places hidden on the user's public reports.
const PrivacyZonesCard: React.FC = () => {
  const { isPending, isError, data, error, refetch, dataUpdatedAt } =
    useQuery(listPrivacyZones);
  let content: React.ReactNode = <Loading />;
  if (isError) {
    content = (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  } else if (!isPending) {
    content = <ZonesEditor key={dataUpdatedAt} zones={data.zones} />;
  }
  return <Card title="Privacy zones">{content}</Card>;
};

export default PrivacyZonesCard;
