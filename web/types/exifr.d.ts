/**
 * The lite bundle of exifr, which reads JPEG and HEIC files. The package's
 * own types depend on Node's types, so only the parts used are declared.
 */
declare module "exifr/dist/lite.esm.mjs" {
  interface ParseOptions {
    reviveValues?: boolean;
  }

  interface Exifr {
    parse: (
      data: Blob,
      options?: ParseOptions
    ) => Promise<Record<string, unknown> | undefined>;
  }

  const exifr: Exifr;
  export default exifr;
}
