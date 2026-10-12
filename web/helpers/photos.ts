import exifr from "exifr/dist/lite.esm.mjs";

/**
 * The longest side of uploaded photos in pixels. Larger photos are scaled
 * down in the browser before uploading.
 */
const MAX_PHOTO_SIZE = 2048;
const JPEG_QUALITY = 0.85;

export interface PreparedPhoto {
  // The photo as a JPEG.
  data: Uint8Array;
  // The GPS position recorded in the photo.
  location: { lat: number; lng: number } | undefined;
  // When the photo was taken, if its UTC offset is known.
  takenTime: Date | undefined;
  /**
   * When the photo was taken as a local time (YYYY-MM-DDTHH:MM:SS), if its
   * offset is unknown, as with most cameras.
   */
  takenLocalTime: string;
}

// EXIF dates look like "2026:09:07 10:15:30".
const EXIF_DATE = /^(\d{4}):(\d{2}):(\d{2})[ T](\d{2}):(\d{2}):(\d{2})/u;
const OFFSET = /^[+-]\d{2}:\d{2}$/u;

type Tags = Record<string, unknown>;

interface Metadata {
  location: PreparedPhoto["location"];
  takenTime: Date | undefined;
  takenLocalTime: string;
}

/**
 * CaptureTime reads when the photo was taken from its EXIF tags: the
 * original date and time, and its UTC offset if the camera recorded one.
 */
const captureTime = (
  tags: Tags
): Pick<Metadata, "takenTime" | "takenLocalTime"> => {
  const none = { takenLocalTime: "", takenTime: undefined };
  const raw = tags.DateTimeOriginal ?? tags.CreateDate;
  if (typeof raw !== "string") {
    return none;
  }
  const match = EXIF_DATE.exec(raw);
  if (match === null) {
    return none;
  }
  const [, year, month, day, hour, minute, second] = match;
  const local = `${year}-${month}-${day}T${hour}:${minute}:${second}`;
  const offset = tags.OffsetTimeOriginal;
  if (typeof offset === "string" && OFFSET.test(offset)) {
    const date = new Date(`${local}${offset}`);
    if (!Number.isNaN(date.getTime())) {
      return { takenLocalTime: "", takenTime: date };
    }
  }
  return { takenLocalTime: local, takenTime: undefined };
};

// Location reads the GPS position from the EXIF tags, as computed by exifr.
const location = (tags: Tags): PreparedPhoto["location"] => {
  const { latitude, longitude } = tags;
  if (
    typeof latitude !== "number" ||
    typeof longitude !== "number" ||
    !Number.isFinite(latitude) ||
    !Number.isFinite(longitude) ||
    (latitude === 0 && longitude === 0)
  ) {
    return undefined;
  }
  return { lat: latitude, lng: longitude };
};

/**
 * ReadMetadata reads the GPS position and capture time of a photo. Missing
 * or unreadable metadata only means the photo isn't placed automatically.
 */
const readMetadata = async (file: File): Promise<Metadata> => {
  let tags: Tags | undefined = undefined;
  try {
    // Raw values keep the capture time as the camera's local time.
    tags = await exifr.parse(file, { reviveValues: false });
  } catch {
    tags = undefined;
  }
  if (tags === undefined) {
    return { location: undefined, takenLocalTime: "", takenTime: undefined };
  }
  return { location: location(tags), ...captureTime(tags) };
};

const loadImageElement = async (file: File): Promise<HTMLImageElement> => {
  const url = URL.createObjectURL(file);
  const image = new Image();
  image.src = url;
  try {
    await image.decode();
    return image;
  } finally {
    URL.revokeObjectURL(url);
  }
};

interface Decoded {
  source: CanvasImageSource;
  width: number;
  height: number;
}

/**
 * DecodeImage decodes a photo, applying its EXIF orientation. Browsers that
 * can display HEIC photos, like Safari, can decode them too.
 */
const decodeImage = async (file: File): Promise<Decoded> => {
  try {
    const bitmap = await createImageBitmap(file, {
      imageOrientation: "from-image",
    });
    return { height: bitmap.height, source: bitmap, width: bitmap.width };
  } catch {
    // Some browsers can only decode some formats with an image element.
  }
  try {
    const image = await loadImageElement(file);
    return {
      height: image.naturalHeight,
      source: image,
      width: image.naturalWidth,
    };
  } catch {
    throw new Error(
      `This browser can't read ${file.name}. Export it as a JPEG and try again.`
    );
  }
};

const canvasToJPEG = async (canvas: HTMLCanvasElement): Promise<Blob> =>
  new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (blob === null) {
          reject(new Error("Failed to encode the photo"));
        } else {
          resolve(blob);
        }
      },
      "image/jpeg",
      JPEG_QUALITY
    );
  });

/**
 * PreparePhoto reads a photo's GPS position and capture time and converts it
 * to a JPEG of at most 2048 pixels for uploading.
 */
export const preparePhoto = async (file: File): Promise<PreparedPhoto> => {
  const metadata = await readMetadata(file);
  const decoded = await decodeImage(file);
  const scale = Math.min(
    1,
    MAX_PHOTO_SIZE / Math.max(decoded.width, decoded.height)
  );
  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.round(decoded.width * scale));
  canvas.height = Math.max(1, Math.round(decoded.height * scale));
  const context = canvas.getContext("2d");
  if (context === null) {
    throw new Error("Failed to prepare the photo");
  }
  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = "high";
  context.drawImage(decoded.source, 0, 0, canvas.width, canvas.height);
  if (decoded.source instanceof ImageBitmap) {
    decoded.source.close();
  }
  const blob = await canvasToJPEG(canvas);
  return {
    data: new Uint8Array(await blob.arrayBuffer()),
    location: metadata.location,
    takenLocalTime: metadata.takenLocalTime,
    takenTime: metadata.takenTime,
  };
};
