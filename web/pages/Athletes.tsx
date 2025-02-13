import Athletes from "@components/athletes/Athletes";
import SignInDialog from "@components/header/SignInDialog";
import { Client } from "@connectrpc/connect";
import { AthleteService } from "gen/athletes/v1/athletes_pb";
import { User } from "gen/users/v1/users_pb";
import React from "react";

interface AthletesPageProps {
  client: Client<typeof AthleteService>;
  user: User | null;
}

const AthletesPage: React.FC<AthletesPageProps> = ({ client, user }) => {
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

  return <Athletes client={client} />;
};

export default AthletesPage;
