import { User } from "gen/users/v1/users_pb";
import React from "react";

interface SettingsPageProps {
  user: User | null;
}

const SettingsPage: React.FC<SettingsPageProps> = ({ user }) => {
  let content = <> </>;
  if (user === null) {
    content = <p>You are not signed in</p>;
  } else {
    content = <p>Welcome to Trail Tools {user.name}</p>;
  }
  return content;
};

export default SettingsPage;
