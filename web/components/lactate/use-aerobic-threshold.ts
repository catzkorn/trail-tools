import { useQuery } from "@connectrpc/connect-query";
import { listLactateTests } from "gen/lactate/v1/lactate-LactateTestService_connectquery";

import type { AerobicThreshold } from "./lactate-utils";
import { latestThreshold } from "./lactate-utils";

/**
 * STALE_TIME is how long the tests are fresh: they change only when the
 * user records a test.
 */
const STALE_TIME = 5 * 60 * 1000;

/**
 * UseAerobicThreshold returns the user's aerobic threshold, from their latest
 * lactate test that found one, or undefined while loading or if none did.
 */
export const useAerobicThreshold = (): AerobicThreshold | undefined => {
  const { data } = useQuery(listLactateTests, {}, { staleTime: STALE_TIME });
  return data === undefined ? undefined : latestThreshold(data.tests);
};
