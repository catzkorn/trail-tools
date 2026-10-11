import Loading from "@components/loading";
import {
  ConnectResultNotice,
  useConnectResult,
} from "@components/strava/connect-result";
import Card from "@components/ui/card";
import ConfirmButton from "@components/ui/confirm-button";
import { Stat } from "@components/ui/misc";
import QueryError, { ErrorText } from "@components/ui/query-error";
import { primaryButton, secondaryButton } from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { ArrowPathIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import { ActivityService } from "gen/activities/v1/activities_pb";
import { GeoService } from "gen/geo/v1/geo_pb";
import {
  disconnectStrava,
  getStravaConnection,
  syncStrava,
} from "gen/strava/v1/strava-StravaService_connectquery";
import type { StravaConnection } from "gen/strava/v1/strava_pb";
import { StravaService, SyncState } from "gen/strava/v1/strava_pb";
import { formatTimestamp } from "helpers/format";
import { invalidateServices } from "helpers/query";
import React, { useEffect, useRef } from "react";

const isSyncing = (state: SyncState | undefined): boolean =>
  state === SyncState.RUNNING || state === SyncState.RATE_LIMITED;

const syncStateLabel = (connection: StravaConnection): string => {
  switch (connection.syncState) {
    case SyncState.RUNNING: {
      return "Syncing…";
    }
    case SyncState.RATE_LIMITED: {
      return `Paused by Strava rate limit, resumes ${formatTimestamp(connection.resumeTime)}`;
    }
    case SyncState.FAILED: {
      return "Last sync failed";
    }
    case SyncState.IDLE: {
      return "Idle";
    }
    default: {
      return "Not synced yet";
    }
  }
};

const syncStateClass = (state: SyncState): string => {
  switch (state) {
    case SyncState.RUNNING: {
      return "bg-sky-100 text-sky-800";
    }
    case SyncState.RATE_LIMITED: {
      return "bg-amber-100 text-amber-800";
    }
    case SyncState.FAILED: {
      return "bg-red-100 text-red-800";
    }
    default: {
      return "bg-gray-100 text-gray-700";
    }
  }
};

interface StravaCardProps {
  // Show the disconnect button, used on the settings page.
  showDisconnect?: boolean;
}

/**
 * StravaCard shows the Strava connection and sync progress. While a sync is
 * running the status is polled every five seconds.
 */
const StravaCard: React.FC<StravaCardProps> = ({ showDisconnect = false }) => {
  const queryClient = useQueryClient();
  const connectResult = useConnectResult();
  const { isPending, isError, data, error, refetch } = useQuery(
    getStravaConnection,
    {},
    {
      refetchInterval: (query) =>
        isSyncing(query.state.data?.connection?.syncState) ? 5000 : false,
    }
  );
  const sync = useMutation(syncStrava, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [StravaService]);
    },
  });
  const disconnect = useMutation(disconnectStrava, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [StravaService]);
    },
  });

  // Refresh activity data when a sync finishes.
  const syncState = data?.connection?.syncState;
  const previousState = useRef(syncState);
  useEffect(() => {
    if (isSyncing(previousState.current) && !isSyncing(syncState)) {
      void invalidateServices(queryClient, [ActivityService, GeoService]);
    }
    previousState.current = syncState;
  }, [syncState, queryClient]);

  if (isPending) {
    return (
      <Card title="Strava">
        <Loading />
      </Card>
    );
  }
  if (isError) {
    return (
      <Card title="Strava">
        <QueryError
          error={error}
          onRetry={() => {
            void refetch();
          }}
        />
      </Card>
    );
  }

  const { connection } = data;
  if (connection === undefined || !connection.configured) {
    return (
      <Card title="Strava">
        <p className="text-sm text-gray-700">
          Strava is not configured on this server. Start the server with the{" "}
          <code className="rounded bg-gray-100 px-1">-strava-client-id</code>{" "}
          and{" "}
          <code className="rounded bg-gray-100 px-1">
            -strava-client-secret
          </code>{" "}
          flags to enable syncing activities.
        </p>
      </Card>
    );
  }

  if (!connection.connected) {
    return (
      <Card title="Strava">
        <div className="flex flex-col items-start gap-3">
          <ConnectResultNotice result={connectResult} />
          <p className="text-sm text-gray-700">
            Connect your Strava account to sync your activities. Your training
            history powers the weekly volume, plan tracking and your personal
            heatmap.
          </p>
          <a
            href="/strava/connect"
            className="inline-flex items-center rounded-md bg-orange-600 px-3 py-1.5 text-sm font-semibold text-white shadow-sm hover:bg-orange-700"
          >
            Connect Strava
          </a>
        </div>
      </Card>
    );
  }

  const syncing = isSyncing(connection.syncState);
  return (
    <Card
      title="Strava"
      actions={
        <span
          className={`rounded-full px-2 py-0.5 text-xs font-medium ${syncStateClass(connection.syncState)}`}
        >
          {syncStateLabel(connection)}
        </span>
      }
    >
      <div className="flex flex-col gap-4">
        <ConnectResultNotice result={connectResult} />
        <p className="text-sm text-gray-700">
          Connected as{" "}
          <a
            href={`https://www.strava.com/athletes/${connection.athleteId}`}
            target="_blank"
            rel="noreferrer"
            className="font-semibold text-orange-700 hover:underline"
          >
            {connection.athleteName}
          </a>
        </p>
        <div className="grid grid-cols-2 gap-3">
          <Stat label="Activities" value={connection.activityCount} />
          <Stat label="Pending streams" value={connection.pendingStreamCount} />
        </div>
        <div className="text-xs text-gray-600">
          Last sync: {formatTimestamp(connection.lastSyncTime)}
        </div>
        {connection.syncMessage !== "" && (
          <p
            className={`rounded-md p-2 text-xs ${connection.syncState === SyncState.FAILED ? "bg-red-50 text-red-800" : "bg-gray-50 text-gray-700"}`}
          >
            {connection.syncMessage}
          </p>
        )}
        <ErrorText error={sync.error ?? disconnect.error} />
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={primaryButton}
            disabled={syncing || sync.isPending}
            onClick={() => {
              sync.mutate({ full: false });
            }}
          >
            <ArrowPathIcon
              className={`size-4 ${syncing ? "animate-spin" : ""}`}
            />
            {syncing ? "Syncing" : "Sync now"}
          </button>
          {showDisconnect && (
            <>
              <button
                type="button"
                className={secondaryButton}
                disabled={syncing || sync.isPending}
                title="Re-fetch all activities to pick up edits made on Strava"
                onClick={() => {
                  sync.mutate({ full: true });
                }}
              >
                Full resync
              </button>
              <ConfirmButton
                title="Disconnect Strava?"
                message="Synced activities are kept, but no new activities will be synced until you connect again."
                confirmLabel="Disconnect"
                disabled={disconnect.isPending}
                onConfirm={() => {
                  disconnect.mutate({});
                }}
              >
                Disconnect
              </ConfirmButton>
            </>
          )}
        </div>
      </div>
    </Card>
  );
};

export default StravaCard;
