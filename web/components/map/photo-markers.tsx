import { useMap } from "@components/map/map-view";
import type { LatLng, LatLngTuple } from "leaflet";
import * as leaflet from "leaflet";
import React, { useEffect, useEffectEvent } from "react";

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

// PhotoMarkers draws photos as round thumbnails on the map.
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

  useEffect(() => {
    const group = leaflet.layerGroup();
    for (const photo of photos) {
      const selected = photo.id === selectedId;
      const marker = leaflet.marker(photo.position, {
        draggable,
        icon: leaflet.divIcon({
          className: "",
          html: markerElement(photo, selected, draggable),
          iconAnchor: [20, 20],
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
  }, [map, photos, selectedId, draggable]);

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
