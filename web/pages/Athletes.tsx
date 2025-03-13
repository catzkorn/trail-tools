import Athletes from "@components/athletes/Athletes";
import SignInDialog from "@components/header/SignInDialog";
import { User } from "gen/users/v1/users_pb";
import React from "react";

interface AthletesPageProps {
  user: User | null;
}

const AthletesPage: React.FC<AthletesPageProps> = ({ user }) => {
  if (user === null) {
    return (
      <div className="flex grow flex-col items-center">
        <div className="pb-5">
          <h1 className="text-4xl text-center">Welcome to Trail Tools!</h1>
          <p className="text-lg text-center">Please sign in to continue.</p>
        </div>
        <SignInDialog />
      </div>
    );
  }

  return <Athletes />;
};

export default AthletesPage;
