import { Dialog, DialogPanel } from "@headlessui/react";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  MapPinIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import type { KeyboardEvent } from "react";
import React from "react";

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
                <img
                  key={photo.id}
                  src={photo.url}
                  alt={photo.caption}
                  width={photo.width > 0 ? photo.width : undefined}
                  height={photo.height > 0 ? photo.height : undefined}
                  className="h-auto max-h-full w-auto max-w-full rounded object-contain shadow-2xl"
                />
              </div>
              <div className="mt-3 flex max-w-3xl flex-wrap items-center justify-center gap-x-4 gap-y-1 text-center text-sm text-gray-200">
                <span className="text-gray-400 tabular-nums">
                  {index + 1} / {photos.length}
                </span>
                {photo.caption !== "" && <span>{photo.caption}</span>}
                {photo.location !== undefined && onShowOnMap !== undefined && (
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 text-emerald-300 hover:text-emerald-200"
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
