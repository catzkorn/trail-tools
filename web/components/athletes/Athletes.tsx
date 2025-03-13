import AddAthleteButton from "@components/athletes/AddAthleteButton";
import AthletesTable from "@components/athletes/AthletesTable";
import Loading from "@components/Loading";
import { useQuery } from "@connectrpc/connect-query";
import { Button } from "@headlessui/react";
import { listAthletes } from "gen/athletes/v1/athletes-AthleteService_connectquery";
import React from "react";

const Athletes: React.FC = () => {
  const { isPending, isError, data, error, refetch } = useQuery(listAthletes);

  if (isPending) {
    return <Loading />;
  }

  if (isError) {
    return (
      <div className="flex grow items-center justify-center">
        <h1>Error: {error.message}</h1>
        <Button
          onClick={() => {
            void refetch();
          }}
        />
      </div>
    );
  }

  const athletes = data.athletes;

  return (
    <div className="flex flex-col grow gap-4 px-4">
      <AthletesTable athletes={athletes} />
      <AddAthleteButton />
    </div>
  );
};

export default Athletes;
