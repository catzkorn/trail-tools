import { useMutation } from "@connectrpc/connect-query";
import { MagnifyingGlassIcon } from "@heroicons/react/24/outline";
import { searchPlaces } from "gen/geo/v1/geo-GeoService_connectquery";
import type { Place } from "gen/geo/v1/geo_pb";
import { readStorageFlag, writeStorageFlag } from "helpers/storage";
import React, { useState } from "react";

import { useViewport } from "./layers";
import { useMap } from "./map-view";
import {
  HeatmapOverlay,
  POI_MIN_ZOOM,
  PoiOverlay,
  TRAILS_MIN_ZOOM,
  TrailsOverlay,
} from "./overlays";

const HEATMAP_KEY = "trail-tools.overlay.heatmap";
const TRAILS_KEY = "trail-tools.overlay.trails";
const POI_KEY = "trail-tools.overlay.poi";

interface ToggleProps {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  note?: string | undefined;
}

const Toggle: React.FC<ToggleProps> = ({ label, checked, onChange, note }) => (
  <label className="flex cursor-pointer items-center gap-2 text-xs text-gray-800">
    <input
      type="checkbox"
      className="size-3.5 accent-emerald-700"
      checked={checked}
      onChange={(event) => {
        onChange(event.target.checked);
      }}
    />
    <span>
      {label}
      {note !== undefined && (
        <span className="ml-1 text-gray-500">({note})</span>
      )}
    </span>
  </label>
);

const PlaceSearch: React.FC = () => {
  const map = useMap();
  const [query, setQuery] = useState("");
  const [places, setPlaces] = useState<Place[]>([]);
  const search = useMutation(searchPlaces, {
    onSuccess: (response) => {
      setPlaces(response.places);
    },
  });

  const goTo = (place: Place) => {
    setPlaces([]);
    if (
      place.bounds !== undefined &&
      place.bounds.north !== place.bounds.south
    ) {
      const { south, west, north, east } = place.bounds;
      map.flyToBounds(
        [
          [south, west],
          [north, east],
        ],
        { maxZoom: 15 }
      );
      return;
    }
    if (place.location !== undefined) {
      map.flyTo([place.location.lat, place.location.lng], 14);
    }
  };

  return (
    <div className="relative w-64 max-w-[calc(100vw-7rem)]">
      <form
        className="flex rounded-md bg-white shadow ring-1 ring-black/10"
        onSubmit={(event) => {
          event.preventDefault();
          if (query.trim() !== "") {
            search.mutate({ limit: 8, query: query.trim() });
          }
        }}
      >
        <input
          type="search"
          value={query}
          placeholder="Search places"
          className="min-w-0 grow rounded-l-md px-2 py-1.5 text-sm focus:outline-none"
          onChange={(event) => {
            setQuery(event.target.value);
          }}
        />
        <button
          type="submit"
          className="px-2 text-gray-600 hover:text-gray-900"
          aria-label="Search"
          disabled={search.isPending}
        >
          <MagnifyingGlassIcon className="size-4" />
        </button>
      </form>
      {search.error !== null && (
        <p className="mt-1 rounded bg-white px-2 py-1 text-xs text-red-700 shadow">
          {search.error.rawMessage}
        </p>
      )}
      {search.isSuccess && places.length === 0 && query !== "" && (
        <p className="mt-1 rounded bg-white px-2 py-1 text-xs text-gray-600 shadow">
          No places found
        </p>
      )}
      {places.length > 0 && (
        <ul className="absolute mt-1 max-h-72 w-full overflow-y-auto rounded-md bg-white py-1 text-sm shadow-lg ring-1 ring-black/10">
          {places.map((place) => (
            <li
              key={`${place.name}-${place.location?.lat}-${place.location?.lng}`}
            >
              <button
                type="button"
                className="w-full px-2 py-1.5 text-left hover:bg-emerald-50"
                onClick={() => {
                  goTo(place);
                }}
              >
                <span className="line-clamp-2">{place.name}</span>
                {place.type !== "" && (
                  <span className="text-xs text-gray-500">{place.type}</span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

/**
 * ExploreControls adds a place search and toggles for the heatmap, trail and
 * point of interest overlays. It must be rendered inside a MapView.
 */
const ExploreControls: React.FC = () => {
  const viewport = useViewport();
  const [heatmap, setHeatmap] = useState(() => readStorageFlag(HEATMAP_KEY));
  const [trails, setTrails] = useState(() => readStorageFlag(TRAILS_KEY));
  const [poi, setPoi] = useState(() => readStorageFlag(POI_KEY));

  return (
    <>
      <div className="absolute top-2 left-12 z-[1000]">
        <PlaceSearch />
      </div>
      <div className="absolute bottom-6 left-2 z-[1000] flex flex-col gap-1 rounded-md bg-white/95 px-2 py-1.5 shadow ring-1 ring-black/10">
        <Toggle
          label="My heatmap"
          checked={heatmap}
          onChange={(checked) => {
            setHeatmap(checked);
            writeStorageFlag(HEATMAP_KEY, checked);
          }}
        />
        <Toggle
          label="Trails"
          checked={trails}
          note={
            trails && viewport.zoom < TRAILS_MIN_ZOOM ? "zoom in" : undefined
          }
          onChange={(checked) => {
            setTrails(checked);
            writeStorageFlag(TRAILS_KEY, checked);
          }}
        />
        <Toggle
          label="Points of interest"
          checked={poi}
          note={poi && viewport.zoom < POI_MIN_ZOOM ? "zoom in" : undefined}
          onChange={(checked) => {
            setPoi(checked);
            writeStorageFlag(POI_KEY, checked);
          }}
        />
      </div>
      {heatmap && <HeatmapOverlay viewport={viewport} />}
      {trails && <TrailsOverlay viewport={viewport} />}
      {poi && <PoiOverlay viewport={viewport} />}
    </>
  );
};

export default ExploreControls;
