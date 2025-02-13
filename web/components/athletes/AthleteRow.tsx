import { Client, ConnectError } from "@connectrpc/connect";
import {
  Button,
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import { Athlete, AthleteService } from "gen/athletes/v1/athletes_pb";
import React, { useState, useTransition } from "react";

interface AthleteRowProps {
  client: Client<typeof AthleteService>;
  athlete: Athlete;
  listAthletes: () => Promise<void>;
}

const AthleteRow: React.FC<AthleteRowProps> = ({
  client,
  athlete,
  listAthletes,
}) => {
  const [isPending, startTransition] = useTransition();
  const [isOpen, setIsOpen] = useState(false);

  const [error, setError] = useState<ConnectError | null>(null);

  const handleSubmit = (e: React.FormEvent<HTMLButtonElement>) => {
    e.preventDefault();
    startTransition(async () => {
      try {
        await client.deleteAthlete({ id: athlete.id });
        setError(null);
        setIsOpen(false);
        await listAthletes();
      } catch (err: unknown) {
        setError(ConnectError.from(err));
      }
    });
  };

  return (
    <tr key={athlete.id}>
      <td className="py-4 w-1/3">{athlete.name}</td>
      <td className="py-4 w-1/3">Unknown</td>
      <td className="py-4 w-1/6">Unknown</td>
      <td className="py-4 w-1/6 text-right">
        <Button
          className="bg-red-500 text-white rounded-md p-1 mr-4"
          onClick={() => {
            setIsOpen(true);
          }}
        >
          <XMarkIcon className="size-4" />
        </Button>
        <Dialog open={isOpen} onClose={setIsOpen} className="relative z-10">
          <DialogBackdrop
            transition
            className="fixed inset-0 bg-gray-500/80 transition-opacity data-[closed]:opacity-0 data-[enter]:duration-300 data-[leave]:duration-200 data-[enter]:ease-out data-[leave]:ease-in"
          />

          <div className="fixed inset-0 z-10 w-screen overflow-y-auto">
            <div className="flex min-h-full items-center justify-center p-4 text-center">
              <DialogPanel
                transition
                className="relative transform overflow-hidden rounded-lg bg-white text-left shadow-xl transition-all data-[closed]:translate-y-4 data-[closed]:opacity-0 data-[enter]:duration-300 data-[leave]:duration-200 data-[enter]:ease-out data-[leave]:ease-in"
              >
                <div className="bg-white px-4 pb-4 pt-5">
                  <div>
                    <div className="flex flex-col gap-4 text-center">
                      <DialogTitle
                        as="h3"
                        className="text-base font-semibold text-gray-900"
                      >
                        Are you sure you wish to delete {athlete.name}?
                      </DialogTitle>
                      {error !== null && <p>{error.message}</p>}
                      <Button
                        onClick={handleSubmit}
                        className="bg-gray-800 text-white rounded-md px-4 py-2 mt-4"
                        disabled={isPending}
                      >
                        Delete
                      </Button>
                    </div>
                  </div>
                </div>
              </DialogPanel>
            </div>
          </div>
        </Dialog>
      </td>
    </tr>
  );
};

export default AthleteRow;
