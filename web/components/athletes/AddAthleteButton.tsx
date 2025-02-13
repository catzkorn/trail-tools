import { Client, ConnectError } from "@connectrpc/connect";
import {
  Button,
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
  Field,
  Fieldset,
  Input,
  Label,
} from "@headlessui/react";
import { PlusIcon } from "@heroicons/react/24/outline";
import { AthleteService } from "gen/athletes/v1/athletes_pb";
import React, { useState, useTransition } from "react";

interface AddAthleteButtonProps {
  client: Client<typeof AthleteService>;
  listAthletes: () => Promise<void>;
}

const AddAthleteButton: React.FC<AddAthleteButtonProps> = ({
  client,
  listAthletes,
}) => {
  const [isPending, startTransition] = useTransition();
  const [isOpen, setIsOpen] = useState(false);
  const [athleteName, setAthleteName] = useState("");

  const [error, setError] = useState<ConnectError | null>(null);

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    startTransition(async () => {
      if (athleteName === "") {
        return;
      }
      try {
        await client.createAthlete({ name: athleteName });
        setError(null);
        setIsOpen(false);
        await listAthletes();
      } catch (err: unknown) {
        setError(ConnectError.from(err));
      }
    });
  };
  return (
    <div className="flex flex-row items-center justify-center">
      <Button
        className="bg-gray-800 text-white rounded-md px-4 py-2 mt-4"
        onClick={() => {
          setIsOpen(true);
        }}
      >
        <div className="flex flex-row gap-2 items-center justify-center">
          <PlusIcon className="size-6" />
          <p>Add athlete</p>
        </div>
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
                      Add athlete
                    </DialogTitle>
                    <form onSubmit={handleSubmit} aria-disabled={isPending}>
                      <Fieldset className="flex flex-col space-y-6 items-center">
                        <Field className="flex flex-col">
                          <Label className="text-left">Athlete name</Label>
                          <Input
                            name="athlete_name"
                            type="text"
                            onChange={(e) => {
                              setAthleteName(e.target.value);
                            }}
                            className="mt-3 block w-full rounded-md border border-button-border-light rounded-md bg-white px-3 py-1.5 text-base text-gray-900"
                          />
                          {error !== null && (
                            <div className="text-red-500">{error.message}</div>
                          )}
                        </Field>
                        <Button
                          type="submit"
                          className="bg-gray-800 text-white rounded-md px-4 py-2 mt-4"
                          disabled={isPending}
                        >
                          Add athlete
                        </Button>
                      </Fieldset>
                    </form>
                  </div>
                </div>
              </div>
            </DialogPanel>
          </div>
        </div>
      </Dialog>
    </div>
  );
};

export default AddAthleteButton;
