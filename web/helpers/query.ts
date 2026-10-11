import type { DescService } from "@bufbuild/protobuf";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import type { QueryClient } from "@tanstack/react-query";

/**
 * InvalidateServices marks every cached query of the given services as stale,
 * so they are refetched. Invalidating whole services keeps mutations simple:
 * there is no need to track exactly which queries a change affects.
 */
export const invalidateServices = async (
  queryClient: QueryClient,
  services: DescService[]
): Promise<void> => {
  await Promise.all(
    services.map(async (service) =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          cardinality: undefined,
          schema: service,
        }),
      })
    )
  );
};
