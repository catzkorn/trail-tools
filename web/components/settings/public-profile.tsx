import Loading from "@components/loading";
import Card from "@components/ui/card";
import { TextField } from "@components/ui/fields";
import QueryError, { ErrorText } from "@components/ui/query-error";
import { linkClass, primaryButton } from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  getProfile,
  updateProfile,
} from "gen/trips/v1/trips-TripReportService_connectquery";
import type { Profile } from "gen/trips/v1/trips_pb";
import { TripReportService } from "gen/trips/v1/trips_pb";
import { invalidateServices } from "helpers/query";
import { isValidHandle } from "helpers/slug";
import React, { useState } from "react";
import { Link } from "react-router";

interface ProfileFormProps {
  profile: Profile | undefined;
  suggestedName: string;
}

const ProfileForm: React.FC<ProfileFormProps> = ({
  profile,
  suggestedName,
}) => {
  const queryClient = useQueryClient();
  const [handle, setHandle] = useState(profile?.handle ?? "");
  const [name, setName] = useState(profile?.displayName ?? suggestedName);
  const [saved, setSaved] = useState(false);
  const update = useMutation(updateProfile, {
    onSuccess: async () => {
      setSaved(true);
      await invalidateServices(queryClient, [TripReportService]);
    },
  });
  const handleValid = isValidHandle(handle);
  const changed =
    handle !== (profile?.handle ?? "") ||
    name.trim() !== (profile?.displayName ?? "");
  const { origin } = globalThis.location;

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        update.mutate({ profile: { displayName: name, handle } });
      }}
    >
      <p className="text-sm text-gray-600">
        Published trip reports are listed on your public page, and can be read
        by anyone with the link. Drafts stay private.
      </p>
      <div className="flex flex-col gap-1">
        <TextField
          label="Handle"
          value={handle}
          placeholder="e.g. jane-doe"
          onChange={(value) => {
            setHandle(value.toLowerCase().replaceAll(/\s/gu, "-"));
            setSaved(false);
          }}
        />
        <span
          className={`text-xs ${handle === "" || handleValid ? "text-gray-500" : "text-red-700"}`}
        >
          3 to 30 lowercase letters, digits and dashes. Your page:{" "}
          <span className="font-mono">
            {origin}/u/{handle === "" ? "<handle>" : handle}
          </span>
        </span>
        {profile !== undefined && handle !== profile.handle && (
          <span className="text-xs text-amber-800">
            Changing your handle changes the address of all your published
            reports, and breaks existing links to them.
          </span>
        )}
      </div>
      <TextField
        label="Display name"
        value={name}
        onChange={(value) => {
          setName(value);
          setSaved(false);
        }}
      />
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="submit"
          className={primaryButton}
          disabled={
            !handleValid || name.trim() === "" || !changed || update.isPending
          }
        >
          {profile === undefined ? "Create public profile" : "Save"}
        </button>
        {saved && !changed && (
          <span className="text-sm text-emerald-700">Saved</span>
        )}
        {profile !== undefined && (
          <Link to={`/u/${profile.handle}`} className={`${linkClass} text-sm`}>
            View your public page
          </Link>
        )}
      </div>
      <ErrorText error={update.error} />
    </form>
  );
};

/**
 * PublicProfileCard sets up the public profile trip reports are published
 * under.
 */
const PublicProfileCard: React.FC = () => {
  const { isPending, isError, data, error, refetch } = useQuery(getProfile);
  let content: React.ReactNode = null;
  if (isPending) {
    content = <Loading />;
  } else if (isError) {
    content = (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  } else {
    content = (
      <ProfileForm
        profile={data.profile}
        suggestedName={data.suggestedDisplayName}
      />
    );
  }
  return <Card title="Public profile">{content}</Card>;
};

export default PublicProfileCard;
