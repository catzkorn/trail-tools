import type { User } from "gen/users/v1/users_pb";
import React from "react";

interface SettingsPageProps {
  user: User | null;
}

const SettingsPage: React.FC<SettingsPageProps> = ({ user }) =>
  user === null ? (
    <p>You are not signed in</p>
  ) : (
    <p>Welcome to Trail Tools {user.name}</p>
  );

export default SettingsPage;
