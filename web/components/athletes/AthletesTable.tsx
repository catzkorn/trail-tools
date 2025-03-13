import AthleteRow from "@components/athletes/AthleteRow";
import { Athlete } from "gen/athletes/v1/athletes_pb";
import React from "react";

interface AthletesTableProps {
  athletes: Athlete[];
}

const AthletesTable: React.FC<AthletesTableProps> = ({ athletes }) => {
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
          <AthleteRow key={athlete.id} athlete={athlete} />
        ))}
      </tbody>
    </table>
  );
};

export default AthletesTable;
