import { User } from "gen/users/v1/users_pb";
import React from "react";

interface ProfilePageProps {
  user: User | null;
}

const ProfilePage: React.FC<ProfilePageProps> = ({ user }) => {
  let content = <p>Welcome to Trail Tools {user?.givenName}</p>;
  if (user === null) {
    content = <p>You are not signed in</p>;
  }
  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">{content}</div>
  );
};

export default ProfilePage;
