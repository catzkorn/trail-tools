import { ErrorText } from "@components/ui/query-error";
import { secondaryButton } from "@components/ui/styles";
import { useMutation } from "@connectrpc/connect-query";
import { ArrowDownTrayIcon } from "@heroicons/react/24/outline";
import { importStravaPhotos } from "gen/trips/v1/trips-TripReportService_connectquery";
import type { ImportStravaPhotosResponse } from "gen/trips/v1/trips_pb";
import React from "react";

const plural = (count: number, noun: string): string =>
  `${count} ${noun}${count === 1 ? "" : "s"}`;

// ImportSummary describes the result of an import.
const importSummary = (result: ImportStravaPhotosResponse): string => {
  const parts: string[] = [];
  if (result.photos.length > 0) {
    parts.push(`Imported ${plural(result.photos.length, "photo")}.`);
  } else if (result.failedCount === 0) {
    parts.push(
      result.skippedCount > 0
        ? "All photos on Strava are already in the report."
        : "The activity has no photos on Strava."
    );
  }
  if (result.failedCount > 0) {
    parts.push(
      `${plural(result.failedCount, "photo")} couldn't be downloaded from Strava; try again later.`
    );
  }
  return parts.join(" ");
};

interface StravaImportProps {
  reportId: string;
  onImported: () => Promise<void>;
}

/**
 * StravaImport imports the photos of the report's activity from Strava.
 * Publishing the report shares the photos, so they're only imported when
 * the author asks.
 */
const StravaImport: React.FC<StravaImportProps> = ({
  reportId,
  onImported,
}) => {
  const importPhotos = useMutation(importStravaPhotos, {
    onSuccess: onImported,
  });
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <button
          type="button"
          className={secondaryButton}
          disabled={importPhotos.isPending}
          onClick={() => {
            importPhotos.mutate({ reportId });
          }}
        >
          <ArrowDownTrayIcon className="size-4" />
          {importPhotos.isPending
            ? "Importing from Strava…"
            : "Import photos from Strava"}
        </button>
        <span className="text-xs text-gray-500">
          Adds the activity&apos;s photos and captions from Strava, skipping
          ones already imported. They&apos;re published with the report.
        </span>
      </div>
      {importPhotos.data !== undefined && (
        <p className="text-sm text-gray-700">
          {importSummary(importPhotos.data)}
        </p>
      )}
      <ErrorText error={importPhotos.error} />
    </div>
  );
};

export default StravaImport;
