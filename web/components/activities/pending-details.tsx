import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { Code } from "@connectrpc/connect";
import { useMutation } from "@connectrpc/connect-query";
import { ArrowPathIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import { ActivityService } from "gen/activities/v1/activities_pb";
import { syncActivity } from "gen/strava/v1/strava-StravaService_connectquery";
import { StravaService } from "gen/strava/v1/strava_pb";
import { formatTimestamp } from "helpers/format";
import { invalidateServices } from "helpers/query";
import React, { useEffect } from "react";
import { Link } from "react-router";

/**
 * How long after the rate limit resets to ask again, by when the backlog has
 * usually synced the requested activity.
 */
const afterResetMs = 30_000;
/**
 * The shortest wait before asking again, and the wait when the activity is
 * queued without a known resume time.
 */
const retryMs = 60_000;
/** The longest delay setTimeout supports. */
const maxDelayMs = 2 ** 31 - 1;

/** FormatResumeTime formats when the rate limit resets, e.g. "at 14:15". */
const formatResumeTime = (timestamp: Timestamp): string => {
  const date = timestampDate(timestamp);
  if (date.toDateString() !== new Date().toDateString()) {
    return formatTimestamp(timestamp);
  }
  return `at ${date.toLocaleTimeString(undefined, { timeStyle: "short" })}`;
};

interface PendingDetailsProps {
  activityId: string;
}

/**
 * PendingDetails syncs the details, photos and streams of an activity that
 * is still waiting for them right away, ahead of the backlog of activities,
 * and says how that's going. Once synced, the page's queries refresh. If
 * Strava's rate limit is reached, the activity is synced first when it
 * resets, and it asks again then.
 */
const PendingDetails: React.FC<PendingDetailsProps> = ({ activityId }) => {
  const queryClient = useQueryClient();
  const sync = useMutation(syncActivity, {
    onSuccess: async (response) => {
      if (response.synced) {
        await invalidateServices(queryClient, [ActivityService, StravaService]);
      }
    },
  });
  const { mutate, data } = sync;
  useEffect(() => {
    mutate({ activityId });
  }, [activityId, mutate]);
  useEffect(() => {
    if (data === undefined || data.synced) {
      return undefined;
    }
    const delay =
      data.resumeTime === undefined
        ? retryMs
        : timestampDate(data.resumeTime).getTime() - Date.now() + afterResetMs;
    const timer = setTimeout(
      () => {
        mutate({ activityId });
      },
      Math.min(Math.max(delay, retryMs), maxDelayMs)
    );
    return () => {
      clearTimeout(timer);
    };
  }, [activityId, data, mutate]);

  let note: React.ReactNode = null;
  if (sync.isPending || data?.synced === true) {
    note = (
      <span className="flex items-center gap-2">
        <ArrowPathIcon className="size-4 animate-spin" />
        Fetching details, photos and streams from Strava…
      </span>
    );
  } else if (sync.error?.code === Code.FailedPrecondition) {
    note = (
      <>
        Detailed data for this activity has not been synced yet.{" "}
        <Link
          to="/settings"
          className="font-medium text-emerald-700 hover:underline"
        >
          Connect Strava
        </Link>{" "}
        to sync it.
      </>
    );
  } else if (sync.error !== null) {
    note = "Detailed data for this activity has not been synced yet.";
  } else if (data?.resumeTime !== undefined) {
    note = `Strava's rate limit is reached. This activity's details, photos and streams sync first when it resets ${formatResumeTime(data.resumeTime)}.`;
  } else if (data !== undefined) {
    note = "This activity's details, photos and streams sync next, shortly.";
  }
  return <p className="text-sm text-gray-500">{note}</p>;
};

export default PendingDetails;
