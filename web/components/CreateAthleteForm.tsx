import React, { useActionState, useState } from "react";
import { useTransition } from "react";
import { Athlete, AthleteService } from "gen/athletes/v1/athletes_pb";
import { createClient, ConnectError } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

const CreateAthleteForm: React.FC = () => {
  const [athlete, setAthlete] = useState<Athlete | null>(null);
  const [error, setError] = useState<ConnectError | null>(null);
  const [name, setName] = useState("");
  const [isPending, startTransition] = useTransition();
  const transport = createConnectTransport({
    baseUrl: ".",
  });
  const client = createClient(AthleteService, transport);

  const handleSubmit = () => {
    startTransition(async () => {
      try {
        setError(null);
        const res = await client.createAthlete({ name: name });
        if (res.athlete !== undefined) {
          setAthlete(res.athlete);
          return;
        }
      } catch (err) {
        if (err instanceof ConnectError) {
          setError(err);
          return;
        }
        setError(ConnectError.from(err));
      }
    });
  };

  return (
    <div>
      <input value={name} onChange={(event) => setName(event.target.value)} />
      <button onClick={handleSubmit} disabled={isPending}>
        Create Athlete
      </button>
      {athlete && (
        <div>
          {athlete.id}, {athlete.name}
        </div>
      )}
      {error !== null && <div>{error.message}</div>}
    </div>
  );
};

export default CreateAthleteForm;
