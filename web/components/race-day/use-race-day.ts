import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { Code } from "@connectrpc/connect";
import {
  createConnectQueryKey,
  useMutation,
  useQuery,
  useTransport,
} from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  getRaceDayStatus,
  setCheckpointSplit,
  updateRaceDay,
} from "gen/races/v1/race_plans-RacePlanService_connectquery";
import type {
  CheckpointSplit,
  RaceDay,
  RaceDayStatus,
} from "gen/races/v1/race_plans_pb";
import { GetRaceDayStatusResponseSchema } from "gen/races/v1/race_plans_pb";

// PlanRef identifies a plan by its ID (signed in) or its share token.
export interface PlanRef {
  planId: string;
  shareToken: string;
}

export interface RaceDayState {
  status: RaceDayStatus | undefined;
  isPending: boolean;
  error: ConnectError | null;
  refetch: () => void;
  saving: boolean;
  saveError: ConnectError | null;
  // Update the given race day fields: "start_time" or "livetrack_url".
  save: (raceDay: RaceDay, paths: string[]) => Promise<void>;
  // Replace the recorded times of one checkpoint.
  saveSplit: (split: CheckpointSplit) => Promise<void>;
}

// Gone reports whether the plan doesn't exist or is no longer shared.
const gone = (err: ConnectError | null): boolean =>
  err?.code === Code.InvalidArgument ||
  err?.code === Code.NotFound ||
  err?.code === Code.PermissionDenied;

/**
 * UseRaceDay loads the race day status, polling every minute and on window
 * focus, and saves race day updates into the query cache. Pages call it next
 * to their plan query so both load together. It stops polling once the plan
 * turns out to be gone, which the page reports.
 */
export const useRaceDay = (ref: PlanRef): RaceDayState => {
  const queryClient = useQueryClient();
  const transport = useTransport();
  const input =
    ref.shareToken === ""
      ? { plan: { case: "planId" as const, value: ref.planId } }
      : { plan: { case: "shareToken" as const, value: ref.shareToken } };
  const query = useQuery(getRaceDayStatus, input, {
    refetchInterval: (statusQuery) =>
      gone(statusQuery.state.error) ? false : 60 * 1000,
    refetchOnWindowFocus: true,
    retry: (failureCount, err) => !gone(err) && failureCount < 3,
  });
  const update = useMutation(updateRaceDay);
  const updateSplit = useMutation(setCheckpointSplit);

  const storeStatus = (status: RaceDayStatus | undefined) => {
    queryClient.setQueryData(
      createConnectQueryKey({
        cardinality: "finite",
        input,
        schema: getRaceDayStatus,
        transport,
      }),
      create(GetRaceDayStatusResponseSchema, { status })
    );
  };

  const save = async (raceDay: RaceDay, paths: string[]): Promise<void> => {
    const response = await update.mutateAsync({
      plan: input.plan,
      raceDay,
      updateMask: { paths },
    });
    storeStatus(response.status);
  };

  const saveSplit = async (split: CheckpointSplit): Promise<void> => {
    const response = await updateSplit.mutateAsync({
      plan: input.plan,
      split,
    });
    storeStatus(response.status);
  };

  return {
    error: query.error,
    isPending: query.isPending,
    refetch: () => {
      void query.refetch();
    },
    save,
    saveError: update.error ?? updateSplit.error,
    saveSplit,
    saving: update.isPending || updateSplit.isPending,
    status: query.data?.status,
  };
};
