import Loading from "@components/loading";
import Card from "@components/ui/card";
import ConfirmButton from "@components/ui/confirm-button";
import QueryError, { ErrorText } from "@components/ui/query-error";
import { ghostButton } from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { TrashIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  deleteConnection,
  listConnections,
} from "gen/oauth/v1/oauth-OAuthService_connectquery";
import { OAuthService } from "gen/oauth/v1/oauth_pb";
import { formatRedirectTarget, formatTimestamp } from "helpers/format";
import { invalidateServices } from "helpers/query";
import React from "react";

// Connections lists the apps connected with OAuth and disconnects them.
// Custom connectors of claude.ai are connected this way.
const Connections: React.FC = () => {
  const queryClient = useQueryClient();
  const { isPending, isError, data, error, refetch } =
    useQuery(listConnections);
  const remove = useMutation(deleteConnection, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [OAuthService]);
    },
  });

  let list: React.ReactNode = null;
  if (isPending) {
    list = <Loading />;
  } else if (isError) {
    list = (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  } else if (data.connections.length === 0) {
    list = <p className="text-sm text-gray-500">No connected apps yet.</p>;
  } else {
    list = (
      <ul className="divide-y divide-gray-100">
        {data.connections.map((connection) => (
          <li
            key={connection.id}
            className="flex items-center justify-between gap-2 py-2"
          >
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-gray-900">
                {connection.clientName}
                <span className="font-normal text-gray-500">
                  {" "}
                  · {formatRedirectTarget(connection.redirectUri)}
                </span>
              </p>
              <p className="text-xs text-gray-500">
                Connected {formatTimestamp(connection.createTime)} · last used{" "}
                {formatTimestamp(connection.lastUsedTime)}
              </p>
            </div>
            <ConfirmButton
              className={ghostButton}
              title="Disconnect app?"
              confirmLabel="Disconnect"
              message={`"${connection.clientName}" will no longer be able to use Trail Tools. You can connect it again later.`}
              disabled={remove.isPending}
              onConfirm={() => {
                remove.mutate({ id: connection.id });
              }}
            >
              <TrashIcon className="size-4" aria-label="Disconnect app" />
            </ConfirmButton>
          </li>
        ))}
      </ul>
    );
  }

  return (
    <Card title="Connected apps">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-gray-600">
          Apps you connected with OAuth, like claude.ai custom connectors, use
          Trail Tools on your behalf. Disconnecting an app revokes its access.
        </p>
        <ErrorText error={remove.error} />
        {list}
      </div>
    </Card>
  );
};

export default Connections;
