import { Client } from "@connectrpc/connect";
import { Athlete, AthleteService } from "gen/athletes/v1/athletes_pb";
import React from "react";
import AthleteRow from "./AthleteRow";

interface AthletesTableProps {
  athletes: Athlete[];
  listAthletes: () => Promise<void>;
  client: Client<typeof AthleteService>;
}

const AthletesTable: React.FC<AthletesTableProps> = ({
  athletes,
  listAthletes,
  client,
}) => {
  return (
    <table className="min-w-full">
      <thead>
        <tr className="text-left text-lg">
          <th scope="col">Name</th>
          <th scope="col">Aerobic threshold trend</th>
          <th scope="col">Next race</th>
          <th scope="col"></th>
        </tr>
      </thead>
      <tbody className="divide-y divide-black border-y border-black">
        {athletes.map((athlete: Athlete) => (
          <AthleteRow
            key={athlete.id}
            client={client}
            athlete={athlete}
            listAthletes={listAthletes}
          />
        ))}
      </tbody>
    </table>
  );
};

export default AthletesTable;
