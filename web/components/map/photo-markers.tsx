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
 * MAX_SPREAD is the most photos spread around their location; larger groups
 * are stacked until the map is zoomed in further.
 */
const MAX_SPREAD = 8;

// Cluster is a group of photos that overlap at a zoom level.
interface Cluster {
  photos: MapPhoto[];
  // The projected pixel positions of the photos, and their center.
  points: leaflet.Point[];
  center: leaflet.Point;
}

/**
 * ClusterPhotos groups photos that would overlap at a zoom level: each photo
 * joins the first group whose center is closer than the marker spacing, so
 * groups stay about a marker wide rather than chaining along a track.
 */
const clusterPhotos = (
  map: leaflet.Map,
  photos: MapPhoto[],
  zoom: number
): Cluster[] => {
  const clusters: Cluster[] = [];
  for (const photo of photos) {
    const point = map.project(photo.position, zoom);
    const cluster = clusters.find(
      (candidate) => candidate.center.distanceTo(point) < MARKER_SPACING
    );
    if (cluster === undefined) {
      clusters.push({ center: point, photos: [photo], points: [point] });
    } else {
      cluster.photos.push(photo);
      cluster.points.push(point);
      cluster.center = cluster.center
        .multiplyBy(cluster.points.length - 1)
        .add(point)
        .divideBy(cluster.points.length);
    }
  }
  return clusters;
};

/**
 * SpreadOffsets returns the pixel offsets that arrange a cluster's photos in
 * a circle around its center, starting at the top.
 */
const spreadOffsets = (cluster: Cluster): leaflet.Point[] => {
  const count = cluster.photos.length;
  const radius = Math.max(
    MARKER_SPACING * 0.6,
    (MARKER_SPACING * count) / (2 * Math.PI)
  );
  return cluster.points.map((point, index) => {
    const angle = -Math.PI / 2 + (2 * Math.PI * index) / count;
    return cluster.center
      .add(leaflet.point(Math.cos(angle) * radius, Math.sin(angle) * radius))
      .subtract(point);
  });
};

// StackElement is a marker for a stack of photos, showing how many.
const stackElement = (cover: MapPhoto, count: number, selected: boolean) => {
  const element = document.createElement("div");
  element.className = [
    "photo-marker",
    "photo-marker-stack",
    selected ? "photo-marker-selected" : "",
  ].join(" ");
  const image = document.createElement("img");
  image.src = cover.thumbnailUrl;
  image.alt = "";
  image.draggable = false;
  element.append(image);
  const badge = document.createElement("span");
  badge.className = "photo-marker-number photo-marker-count";
  badge.textContent = String(count);
  element.append(badge);
  return element;
};

/**
 * PhotoMarkers draws photos as round thumbnails on the map. Photos that
 * overlap are stacked, showing how many, until the map is zoomed in far
 * enough to spread them around their location with a line to it.
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
  const clusters = useMemo(
    () => clusterPhotos(map, photos, zoom),
    [map, photos, zoom]
  );

  useEffect(() => {
    const group = leaflet.layerGroup();
    const addPhoto = (photo: MapPhoto, offset: leaflet.Point) => {
      const selected = photo.id === selectedId;
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
    };
    const atMaxZoom = zoom >= map.getMaxZoom();
    for (const cluster of clusters) {
      const [first] = cluster.photos;
      if (first === undefined) {
        continue;
      }
      if (cluster.photos.length === 1) {
        addPhoto(first, leaflet.point(0, 0));
        continue;
      }
      if (cluster.photos.length <= MAX_SPREAD || atMaxZoom) {
        const offsets = spreadOffsets(cluster);
        for (const [index, photo] of cluster.photos.entries()) {
          addPhoto(photo, offsets[index] ?? leaflet.point(0, 0));
        }
        continue;
      }
      // Too many to spread: a stack, which zooms in to its photos.
      const selected = cluster.photos.find((photo) => photo.id === selectedId);
      const center = map.unproject(cluster.center, zoom);
      const stack = leaflet.marker(center, {
        icon: leaflet.divIcon({
          className: "",
          html: stackElement(
            selected ?? first,
            cluster.photos.length,
            selected !== undefined
          ),
          iconAnchor: [20, 20],
          iconSize: [40, 40],
        }),
        riseOnHover: true,
        title: `${cluster.photos.length} photos: zoom in to see them`,
        zIndexOffset: selected === undefined ? 500 : 1000,
      });
      stack.on("click", () => {
        const bounds = leaflet.latLngBounds(
          cluster.photos.map((photo) => photo.position)
        );
        const target = Math.min(
          map.getMaxZoom(),
          Math.max(map.getBoundsZoom(bounds.pad(0.2)), zoom + 2)
        );
        map.flyTo(bounds.getCenter(), target, { duration: 0.4 });
      });
      group.addLayer(stack);
    }
    group.addTo(map);
    return () => {
      group.remove();
    };
  }, [map, clusters, selectedId, draggable, zoom]);

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
