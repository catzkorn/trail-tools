import Invitations from "@components/settings/invitations";
import Passkeys from "@components/settings/passkeys";
import { PageHeader } from "@components/ui/misc";
import type { User } from "gen/users/v1/users_pb";
import React from "react";

interface SettingsPageProps {
  user: User | null;
}

const SettingsPage: React.FC<SettingsPageProps> = ({ user }) => (
  <div>
    <PageHeader title="Settings" />
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div className="flex flex-col gap-4">
        <Passkeys />
      </div>
      <div className="flex flex-col gap-4">
        <Invitations admin={user !== null && user.admin} />
      </div>
    </div>
  </div>
);

export default SettingsPage;
