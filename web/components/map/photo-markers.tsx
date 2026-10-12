import { useMap } from "@components/map/map-view";
import type { LatLng, LatLngTuple } from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useEffectEvent, useMemo, useState } from "react";

// MapPhoto is a photo with a location, shown as a marker.
export interface MapPhoto {
  id: string;
  position: LatLngTuple;
  thumbnailUrl: string;
  // The photo's number, shown on the marker.
  number: number;
  caption: string;
}

interface PhotoMarkersProps {
  photos: MapPhoto[];
  selectedId: string | null;
  onSelect?: ((id: string) => void) | undefined;
  // Makes the markers draggable, and is called when one is dropped.
  onMove?: ((id: string, position: LatLng) => void) | undefined;
}

const markerElement = (
  photo: MapPhoto,
  selected: boolean,
  draggable: boolean
) => {
  const element = document.createElement("div");
  element.className = [
    "photo-marker",
    selected ? "photo-marker-selected" : "",
    draggable ? "photo-marker-draggable" : "",
  ].join(" ");
  const image = document.createElement("img");
  image.src = photo.thumbnailUrl;
  image.alt = "";
  image.draggable = false;
  element.append(image);
  const number = document.createElement("span");
  number.className = "photo-marker-number";
  number.textContent = String(photo.number);
  element.append(number);
  return element;
};

// MARKER_SPACING is how far apart in pixels photo markers are kept.
const MARKER_SPACING = 44;

/**
 * SpreadOffsets returns pixel offsets that move markers which would overlap
 * at a zoom level apart: each group of overlapping photos is arranged in a
 * circle around its center. Photos that don't overlap aren't moved.
 */
const spreadOffsets = (
  map: leaflet.Map,
  photos: MapPhoto[],
  zoom: number
): Map<string, leaflet.Point> => {
  const items = photos.map((photo) => ({
    id: photo.id,
    point: map.project(photo.position, zoom),
  }));
  // Group photos closer than the spacing to any photo of a group.
  let groups: (typeof items)[] = [];
  for (const item of items) {
    const near = groups.filter((group) =>
      group.some((other) => other.point.distanceTo(item.point) < MARKER_SPACING)
    );
    const members = new Set([item, ...near.flat()]);
    groups = [
      ...groups.filter((group) => !near.includes(group)),
      // In the order of the photos.
      items.filter((other) => members.has(other)),
    ];
  }
  const offsets = new Map<string, leaflet.Point>();
  for (const group of groups) {
    if (group.length < 2) {
      continue;
    }
    let center = leaflet.point(0, 0);
    for (const item of group) {
      center = center.add(item.point);
    }
    center = center.divideBy(group.length);
    // A circle with room for every marker, starting at the top.
    const radius = Math.max(
      MARKER_SPACING * 0.6,
      (MARKER_SPACING * group.length) / (2 * Math.PI)
    );
    for (const [index, item] of group.entries()) {
      const angle = -Math.PI / 2 + (2 * Math.PI * index) / group.length;
      const target = center.add(
        leaflet.point(Math.cos(angle) * radius, Math.sin(angle) * radius)
      );
      offsets.set(item.id, target.subtract(item.point));
    }
  }
  return offsets;
};

/**
 * PhotoMarkers draws photos as round thumbnails on the map. Photos close
 * together are spread out around their location, with a line to it.
 */
const PhotoMarkers: React.FC<PhotoMarkersProps> = ({
  photos,
  selectedId,
  onSelect,
  onMove,
}) => {
  const map = useMap();
  const draggable = onMove !== undefined;
  const handleSelect = useEffectEvent((id: string) => {
    onSelect?.(id);
  });
  const handleMove = useEffectEvent((id: string, position: LatLng) => {
    onMove?.(id, position);
  });
  const [zoom, setZoom] = useState(() => map.getZoom());
  useEffect(() => {
    const onZoom = () => {
      setZoom(map.getZoom());
    };
    map.on("zoomend", onZoom);
    return () => {
      map.off("zoomend", onZoom);
    };
  }, [map]);
  const offsets = useMemo(
    () => spreadOffsets(map, photos, zoom),
    [map, photos, zoom]
  );

  useEffect(() => {
    const group = leaflet.layerGroup();
    for (const photo of photos) {
      const selected = photo.id === selectedId;
      const offset = offsets.get(photo.id) ?? leaflet.point(0, 0);
      if (offset.x !== 0 || offset.y !== 0) {
        // A line from where the photo was taken to its moved marker.
        const at = map.project(photo.position, zoom);
        group.addLayer(
          leaflet.polyline(
            [photo.position, map.unproject(at.add(offset), zoom)],
            {
              color: "#374151",
              interactive: false,
              opacity: 0.7,
              weight: 1.5,
            }
          )
        );
      }
      const marker = leaflet.marker(photo.position, {
        draggable,
        icon: leaflet.divIcon({
          className: "",
          html: markerElement(photo, selected, draggable),
          iconAnchor: [20 - offset.x, 20 - offset.y],
          iconSize: [40, 40],
        }),
        riseOnHover: true,
        title: photo.caption === "" ? `Photo ${photo.number}` : photo.caption,
        zIndexOffset: selected ? 1000 : 0,
      });
      marker.on("click", () => {
        handleSelect(photo.id);
      });
      marker.on("dragend", () => {
        handleMove(photo.id, marker.getLatLng());
      });
      group.addLayer(marker);
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, photos, selectedId, draggable, offsets, zoom]);

  // Bring the selected photo into view.
  const selected = photos.find((photo) => photo.id === selectedId);
  const [lat, lng] = selected?.position ?? [0, 0];
  const hasSelected = selected !== undefined;
  useEffect(() => {
    if (hasSelected && !map.getBounds().pad(-0.1).contains([lat, lng])) {
      map.panTo([lat, lng]);
    }
  }, [map, hasSelected, lat, lng]);
  return null;
};

export default PhotoMarkers;
