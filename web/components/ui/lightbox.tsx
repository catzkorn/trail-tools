import { Dialog, DialogPanel } from "@headlessui/react";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  MagnifyingGlassMinusIcon,
  MagnifyingGlassPlusIcon,
  MapPinIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import type { KeyboardEvent, PointerEvent } from "react";
import React, { useCallback, useEffect, useRef, useState } from "react";

// LightboxPhoto is a photo shown in the lightbox.
export interface LightboxPhoto {
  id: string;
  url: string;
  caption: string;
  // The size of the image in pixels, 0 if unknown.
  width: number;
  height: number;
  // Set if the photo can be shown on the map.
  location?: unknown;
}

interface LightboxProps<Photo extends LightboxPhoto> {
  photos: Photo[];
  // The index of the photo shown, or null when closed.
  index: number | null;
  onChange: (index: number | null) => void;
  // Shows the photo on the map, if it has a location.
  onShowOnMap?: ((photo: Photo) => void) | undefined;
}

// The largest zoom, relative to the photo fitted to the screen.
const MAX_ZOOM = 5;
// The zoom a double click or tap zooms to.
const DOUBLE_CLICK_ZOOM = 2.5;

interface View {
  zoom: number;
  // The offset of the photo's center from the frame's center, in pixels.
  panX: number;
  panY: number;
}

const FITTED: View = { panX: 0, panY: 0, zoom: 1 };

interface Position {
  clientX: number;
  clientY: number;
}

/**
 * ZoomAt returns the view zoomed to a zoom level, keeping the point at a
 * client position still, and moved no further than the photo's edges.
 */
const zoomAt = (
  frame: HTMLElement | null,
  image: HTMLElement | null,
  current: View,
  zoom: number,
  at: Position
): View => {
  if (frame === null || image === null) {
    return current;
  }
  const bounds = frame.getBoundingClientRect();
  const nextZoom = Math.min(Math.max(zoom, 1), MAX_ZOOM);
  // The point relative to the frame's center.
  const pointX = at.clientX - (bounds.left + bounds.width / 2);
  const pointY = at.clientY - (bounds.top + bounds.height / 2);
  const ratio = nextZoom / current.zoom;
  const panX = pointX - (pointX - current.panX) * ratio;
  const panY = pointY - (pointY - current.panY) * ratio;
  // How far the zoomed photo can move before showing its edge.
  const maxX = Math.max((image.offsetWidth * nextZoom - bounds.width) / 2, 0);
  const maxY = Math.max((image.offsetHeight * nextZoom - bounds.height) / 2, 0);
  return {
    panX: Math.min(Math.max(panX, -maxX), maxX),
    panY: Math.min(Math.max(panY, -maxY), maxY),
    zoom: nextZoom,
  };
};

// The center of an element, as a client position.
const centerOf = (element: HTMLElement): Position => {
  const bounds = element.getBoundingClientRect();
  return {
    clientX: bounds.left + bounds.width / 2,
    clientY: bounds.top + bounds.height / 2,
  };
};

interface ZoomableImageProps {
  photo: LightboxPhoto;
  // Called with a function that zooms by a factor, for zoom buttons.
  onZoomControl: (zoomBy: (factor: number) => void) => void;
}

/**
 * ZoomableImage shows a photo fitted to its frame, which can be zoomed with
 * the scroll wheel, a pinch or a double click, and panned by dragging.
 */
const ZoomableImage: React.FC<ZoomableImageProps> = ({
  photo,
  onZoomControl,
}) => {
  const frameRef = useRef<HTMLDivElement>(null);
  const imageRef = useRef<HTMLImageElement>(null);
  const [view, setView] = useState<View>(FITTED);
  // The pointers down on the photo, for dragging and pinching.
  const pointers = useRef(new Map<number, Position>());
  const lastTap = useRef(0);

  // Zooms from the buttons, around the frame's center.
  useEffect(() => {
    onZoomControl((factor) => {
      const frame = frameRef.current;
      if (frame !== null) {
        setView((current) =>
          zoomAt(
            frame,
            imageRef.current,
            current,
            current.zoom * factor,
            centerOf(frame)
          )
        );
      }
    });
  }, [onZoomControl]);

  /**
   * The wheel zooms instead of scrolling, which needs a listener that isn't
   * passive.
   */
  useEffect(() => {
    const frame = frameRef.current;
    if (frame === null) {
      return undefined;
    }
    const onWheel = (event: WheelEvent) => {
      event.preventDefault();
      const factor = Math.exp(-event.deltaY * 0.002);
      setView((current) =>
        zoomAt(frame, imageRef.current, current, current.zoom * factor, event)
      );
    };
    frame.addEventListener("wheel", onWheel, { passive: false });
    return () => {
      frame.removeEventListener("wheel", onWheel);
    };
  }, []);

  const onPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    event.currentTarget.setPointerCapture(event.pointerId);
    pointers.current.set(event.pointerId, {
      clientX: event.clientX,
      clientY: event.clientY,
    });
    // A second tap soon after the first toggles the zoom.
    const now = Date.now();
    if (pointers.current.size === 1 && now - lastTap.current < 300) {
      const at = { clientX: event.clientX, clientY: event.clientY };
      setView((current) =>
        current.zoom > 1
          ? FITTED
          : zoomAt(
              frameRef.current,
              imageRef.current,
              current,
              DOUBLE_CLICK_ZOOM,
              at
            )
      );
      lastTap.current = 0;
      return;
    }
    lastTap.current = now;
  };
  const onPointerMove = (event: PointerEvent<HTMLDivElement>) => {
    const previous = pointers.current.get(event.pointerId);
    if (previous === undefined) {
      return;
    }
    const other = [...pointers.current.entries()].find(
      ([id]) => id !== event.pointerId
    );
    const at = { clientX: event.clientX, clientY: event.clientY };
    pointers.current.set(event.pointerId, at);
    const frame = frameRef.current;
    const image = imageRef.current;
    if (other === undefined) {
      // Dragging pans a zoomed photo.
      setView((current) =>
        current.zoom === 1
          ? current
          : zoomAt(
              frame,
              image,
              {
                ...current,
                panX: current.panX + at.clientX - previous.clientX,
                panY: current.panY + at.clientY - previous.clientY,
              },
              current.zoom,
              frame === null ? at : centerOf(frame)
            )
      );
      return;
    }
    // Pinching zooms around the middle of the two pointers.
    const [, fixed] = other;
    const before = Math.hypot(
      previous.clientX - fixed.clientX,
      previous.clientY - fixed.clientY
    );
    const after = Math.hypot(
      at.clientX - fixed.clientX,
      at.clientY - fixed.clientY
    );
    if (before > 0) {
      const middle = {
        clientX: (at.clientX + fixed.clientX) / 2,
        clientY: (at.clientY + fixed.clientY) / 2,
      };
      setView((current) =>
        zoomAt(frame, image, current, (current.zoom * after) / before, middle)
      );
    }
  };
  const onPointerUp = (event: PointerEvent<HTMLDivElement>) => {
    pointers.current.delete(event.pointerId);
  };

  return (
    <div
      ref={frameRef}
      className={`flex h-full w-full touch-none items-center justify-center overflow-hidden select-none ${view.zoom > 1 ? "cursor-grab active:cursor-grabbing" : "cursor-zoom-in"}`}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
    >
      <img
        ref={imageRef}
        src={photo.url}
        alt={photo.caption}
        width={photo.width > 0 ? photo.width : undefined}
        height={photo.height > 0 ? photo.height : undefined}
        draggable={false}
        style={{
          transform: `translate(${view.panX}px, ${view.panY}px) scale(${view.zoom})`,
        }}
        className="h-auto max-h-full w-auto max-w-full rounded object-contain shadow-2xl transition-transform duration-75 ease-out"
      />
    </div>
  );
};

const zoomButton =
  "rounded-full p-1 text-gray-300 hover:bg-white/10 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white";

const navButton =
  "absolute top-1/2 -translate-y-1/2 rounded-full bg-black/40 p-2 text-white hover:bg-black/70 focus:outline-none focus-visible:ring-2 focus-visible:ring-white";

// Lightbox shows photos full screen, with arrows to move between them.
const Lightbox = <Photo extends LightboxPhoto>({
  photos,
  index,
  onChange,
  onShowOnMap,
}: LightboxProps<Photo>): React.ReactElement => {
  const photo = index === null ? undefined : photos[index];
  const zoomBy = useRef<((factor: number) => void) | null>(null);
  const setZoomControl = useCallback((control: (factor: number) => void) => {
    zoomBy.current = control;
  }, []);
  const close = () => {
    onChange(null);
  };
  const step = (delta: number) => {
    if (index !== null && photos.length > 0) {
      onChange((index + delta + photos.length) % photos.length);
    }
  };
  const handleKey = (event: KeyboardEvent) => {
    if (event.key === "ArrowLeft") {
      step(-1);
    } else if (event.key === "ArrowRight") {
      step(1);
    }
  };

  return (
    <Dialog
      open={photo !== undefined}
      onClose={close}
      onKeyDown={handleKey}
      className="relative z-[2000]"
    >
      <div className="fixed inset-0 bg-gray-950/95" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center">
        <DialogPanel className="relative flex h-full w-full flex-col items-center justify-center px-2 py-12 sm:px-16">
          {photo !== undefined && index !== null && (
            <>
              <div className="flex min-h-0 w-full flex-1 items-center justify-center">
                <ZoomableImage
                  key={photo.id}
                  photo={photo}
                  onZoomControl={setZoomControl}
                />
              </div>
              <div className="mt-3 flex max-w-3xl flex-wrap items-center justify-center gap-x-4 gap-y-1 text-center text-sm text-gray-200">
                <span className="text-gray-400 tabular-nums">
                  {index + 1} / {photos.length}
                </span>
                <span className="inline-flex items-center gap-1">
                  <button
                    type="button"
                    className={zoomButton}
                    aria-label="Zoom out"
                    onClick={() => {
                      zoomBy.current?.(1 / 1.5);
                    }}
                  >
                    <MagnifyingGlassMinusIcon className="size-5" />
                  </button>
                  <button
                    type="button"
                    className={zoomButton}
                    aria-label="Zoom in"
                    onClick={() => {
                      zoomBy.current?.(1.5);
                    }}
                  >
                    <MagnifyingGlassPlusIcon className="size-5" />
                  </button>
                </span>
                {photo.caption !== "" && <span>{photo.caption}</span>}
                {photo.location !== undefined && onShowOnMap !== undefined && (
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 text-brand-300 hover:text-brand-200"
                    onClick={() => {
                      onShowOnMap(photo);
                    }}
                  >
                    <MapPinIcon className="size-4" />
                    Show on map
                  </button>
                )}
              </div>
              {photos.length > 1 && (
                <>
                  <button
                    type="button"
                    className={`${navButton} left-2`}
                    aria-label="Previous photo"
                    onClick={() => {
                      step(-1);
                    }}
                  >
                    <ChevronLeftIcon className="size-6" />
                  </button>
                  <button
                    type="button"
                    className={`${navButton} right-2`}
                    aria-label="Next photo"
                    onClick={() => {
                      step(1);
                    }}
                  >
                    <ChevronRightIcon className="size-6" />
                  </button>
                </>
              )}
            </>
          )}
          <button
            type="button"
            className="absolute top-2 right-2 rounded-full p-2 text-white hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-white"
            aria-label="Close"
            onClick={close}
          >
            <XMarkIcon className="size-6" />
          </button>
        </DialogPanel>
      </div>
    </Dialog>
  );
};

export default Lightbox;
