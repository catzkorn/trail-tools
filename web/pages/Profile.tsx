import { User } from "gen/users/v1/users_pb";
import React from "react";

interface ProfilePageProps {
  user: User | null;
}

const ProfilePage: React.FC<ProfilePageProps> = ({ user }) => {
  let content = <> </>;
  if (user === null) {
    content = <p>You are not signed in</p>;
  } else {
    content = <p>Welcome to Trail Tools {user.name}</p>;
  }
  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">{content}</div>
  );
};

export default ProfilePage;
