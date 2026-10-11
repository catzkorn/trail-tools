import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";

// FormatTimestamp formats a protobuf timestamp as a local date and time.
export const formatTimestamp = (timestamp: Timestamp | undefined): string => {
  if (timestamp === undefined) {
    return "never";
  }
  return timestampDate(timestamp).toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
};
