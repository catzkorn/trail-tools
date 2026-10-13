import { timestampMs } from "@bufbuild/protobuf/wkt";
import Loading from "@components/loading";
import Card from "@components/ui/card";
import ConfirmButton from "@components/ui/confirm-button";
import { Badge } from "@components/ui/misc";
import QueryError, { ErrorText } from "@components/ui/query-error";
import {
  ghostDangerButton,
  inputClass,
  primaryButton,
} from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { TrashIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  createInvitation,
  deleteInvitation,
  listInvitations,
} from "gen/auth/v1/auth-AuthService_connectquery";
import type { Invitation } from "gen/auth/v1/auth_pb";
import { AuthService, InvitationKind } from "gen/auth/v1/auth_pb";
import { formatTimestamp } from "helpers/format";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";

import InvitationLink from "./invitation-link";

type InvitationStatus = "pending" | "used" | "expired";

const invitationStatus = (invitation: Invitation): InvitationStatus => {
  if (invitation.useTime !== undefined) {
    return "used";
  }
  if (
    invitation.expireTime !== undefined &&
    timestampMs(invitation.expireTime) < Date.now()
  ) {
    return "expired";
  }
  return "pending";
};

const statusBadge = (status: InvitationStatus): React.ReactNode => {
  if (status === "used") {
    return (
      <Badge className="bg-emerald-50 text-emerald-800 ring-emerald-200">
        Used
      </Badge>
    );
  }
  if (status === "expired") {
    return <Badge>Expired</Badge>;
  }
  return (
    <Badge className="bg-amber-50 text-amber-800 ring-amber-200">Pending</Badge>
  );
};

const invitationTitle = (invitation: Invitation): string => {
  if (invitation.kind === InvitationKind.ADD_PASSKEY) {
    return "Add a passkey on another device";
  }
  if (invitation.bootstrap) {
    return "Administrator account";
  }
  return invitation.note === "" ? "New user" : `New user: ${invitation.note}`;
};

interface InvitationsProps {
  admin: boolean;
}

/**
 * Invitations lists the invitations the user created. Administrators can
 * also invite new users here.
 */
const Invitations: React.FC<InvitationsProps> = ({ admin }) => {
  const queryClient = useQueryClient();
  const { isPending, isError, data, error, refetch } =
    useQuery(listInvitations);
  const [note, setNote] = useState("");
  const [inviteUrl, setInviteUrl] = useState("");
  const onChanged = async () => {
    await invalidateServices(queryClient, [AuthService]);
  };
  const create = useMutation(createInvitation, {
    onSuccess: async (response) => {
      setInviteUrl(response.url);
      setNote("");
      await onChanged();
    },
  });
  const remove = useMutation(deleteInvitation, { onSuccess: onChanged });

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
  } else if (data.invitations.length === 0) {
    list = <p className="text-sm text-gray-500">No invitations.</p>;
  } else {
    list = (
      <ul className="divide-y divide-gray-100">
        {data.invitations.map((invitation) => (
          <li
            key={invitation.id}
            className="flex items-center justify-between gap-2 py-2"
          >
            <div className="min-w-0">
              <p className="flex flex-wrap items-center gap-2 text-sm font-medium text-gray-900">
                <span className="truncate">{invitationTitle(invitation)}</span>
                {statusBadge(invitationStatus(invitation))}
              </p>
              <p className="text-xs text-gray-500">
                Created {formatTimestamp(invitation.createTime)}
                {invitation.useTime === undefined
                  ? ` · expires ${formatTimestamp(invitation.expireTime)}`
                  : ` · used ${formatTimestamp(invitation.useTime)}`}
              </p>
            </div>
            <ConfirmButton
              className={ghostDangerButton}
              title="Delete invitation?"
              message="The invitation link will stop working."
              disabled={remove.isPending}
              onConfirm={() => {
                remove.mutate({ id: invitation.id });
              }}
            >
              <TrashIcon className="size-4" aria-label="Delete invitation" />
            </ConfirmButton>
          </li>
        ))}
      </ul>
    );
  }

  return (
    <Card title="Invitations">
      <div className="flex flex-col gap-4">
        {admin ? (
          <form
            className="flex flex-col gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              create.mutate({
                kind: InvitationKind.NEW_USER,
                note: note.trim(),
              });
            }}
          >
            <p className="text-sm text-gray-600">
              trailrunn.ing is invite only. Invite someone by sending them a
              link; they create their account with a passkey.
            </p>
            <div className="flex flex-wrap items-end gap-2">
              <label className="flex min-w-48 grow flex-col gap-1">
                <span className="text-xs font-medium text-gray-600">
                  Note (who is it for?)
                </span>
                <input
                  value={note}
                  className={inputClass}
                  placeholder="e.g. Anna"
                  onChange={(event) => {
                    setNote(event.target.value);
                  }}
                />
              </label>
              <button
                type="submit"
                className={primaryButton}
                disabled={create.isPending}
              >
                Invite a new user
              </button>
            </div>
          </form>
        ) : (
          <p className="text-sm text-gray-600">
            Links you created to add passkeys on other devices.
          </p>
        )}
        {inviteUrl !== "" && (
          <InvitationLink
            url={inviteUrl}
            description="Send this link to the person you are inviting."
            onDone={() => {
              setInviteUrl("");
            }}
          />
        )}
        <ErrorText error={create.error ?? remove.error} />
        {list}
      </div>
    </Card>
  );
};

export default Invitations;
