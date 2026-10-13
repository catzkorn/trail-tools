import Avatar from "@components/header/avatar";
import Card from "@components/ui/card";
import { ErrorText } from "@components/ui/query-error";
import { secondaryButton, secondaryDangerButton } from "@components/ui/styles";
import { useMutation } from "@connectrpc/connect-query";
import { ArrowUpTrayIcon, TrashIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  deleteAvatar,
  uploadAvatar,
} from "gen/users/v1/users-UserService_connectquery";
import type { User } from "gen/users/v1/users_pb";
import { UserService } from "gen/users/v1/users_pb";
import { preparePhoto } from "helpers/photos";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";

// AvatarCard uploads and removes the user's avatar.
const AvatarCard: React.FC<{ user: User }> = ({ user }) => {
  const queryClient = useQueryClient();
  const [preparing, setPreparing] = useState(false);
  const [prepareError, setPrepareError] = useState("");
  const onSuccess = async () => {
    await invalidateServices(queryClient, [UserService]);
  };
  const upload = useMutation(uploadAvatar, { onSuccess });
  const remove = useMutation(deleteAvatar, { onSuccess });
  const busy = preparing || upload.isPending || remove.isPending;

  const choose = async (file: File) => {
    setPrepareError("");
    setPreparing(true);
    try {
      /**
       * Decoding in the browser applies the photo's orientation, and reads
       * formats like HEIC on browsers that can.
       */
      const prepared = await preparePhoto(file);
      upload.mutate({ data: prepared.data });
    } catch (error) {
      setPrepareError(error instanceof Error ? error.message : String(error));
    } finally {
      setPreparing(false);
    }
  };

  let status = "";
  if (preparing || upload.isPending) {
    status = "Uploading…";
  } else if (remove.isPending) {
    status = "Removing…";
  }

  return (
    <Card title="Profile picture">
      <div className="flex items-center gap-4">
        <Avatar
          avatarUrl={user.avatarUrl}
          name={user.name}
          className="size-20 shrink-0 text-2xl ring-1 ring-gray-200"
        />
        <div className="flex flex-col gap-2">
          <p className="text-sm text-gray-600">
            Shown in the header. The middle square of the picture is used.
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <label
              className={`${secondaryButton} cursor-pointer ${busy ? "pointer-events-none opacity-50" : ""}`}
            >
              <ArrowUpTrayIcon className="size-4" />
              {user.avatarUrl === "" ? "Upload picture" : "Change picture"}
              <input
                type="file"
                accept="image/*"
                className="sr-only"
                disabled={busy}
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  event.target.value = "";
                  if (file !== undefined) {
                    void choose(file);
                  }
                }}
              />
            </label>
            {user.avatarUrl !== "" && (
              <button
                type="button"
                className={secondaryDangerButton}
                disabled={busy}
                onClick={() => {
                  remove.mutate({});
                }}
              >
                <TrashIcon className="size-4" />
                Remove
              </button>
            )}
            {status !== "" && (
              <span className="text-sm text-gray-500">{status}</span>
            )}
          </div>
        </div>
      </div>
      {prepareError !== "" && (
        <p className="text-sm text-red-700">{prepareError}</p>
      )}
      <ErrorText error={upload.error ?? remove.error} />
    </Card>
  );
};

export default AvatarCard;
