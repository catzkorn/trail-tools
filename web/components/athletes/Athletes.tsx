import AddAthleteButton from "@components/athletes/AddAthleteButton";
import AthletesTable from "@components/athletes/AthletesTable";
import { Client, ConnectError } from "@connectrpc/connect";
import { Athlete, AthleteService } from "gen/athletes/v1/athletes_pb";
import React, { useEffect, useState } from "react";

interface AthletesProps {
  client: Client<typeof AthleteService>;
}

const Athletes: React.FC<AthletesProps> = ({ client }) => {
  const [athletes, setAthletes] = useState<Athlete[]>([]);

  const listAthletes = async () => {
    try {
      const result = await client.listAthletes({});
      setAthletes(result.athletes);
    } catch (err: unknown) {
      console.error("Failed to get athletes:", ConnectError.from(err));
    }
  };

  useEffect(() => {
    void listAthletes();
  }, [client]);

  return (
    <div className="flex flex-col grow gap-4 px-4">
      <AthletesTable
        client={client}
        athletes={athletes}
        listAthletes={listAthletes}
      />
      <AddAthleteButton client={client} listAthletes={listAthletes} />
    </div>
  );
};

export default Athletes;
