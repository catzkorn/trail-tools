import { createConnectQueryKey, useMutation } from "@connectrpc/connect-query";
import {
  Button,
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  deleteAthlete,
  listAthletes,
} from "gen/athletes/v1/athletes-AthleteService_connectquery";
import { Athlete } from "gen/athletes/v1/athletes_pb";
import React, { useState } from "react";

interface AthleteRowProps {
  athlete: Athlete;
}

const AthleteRow: React.FC<AthleteRowProps> = ({ athlete }) => {
  const queryClient = useQueryClient();
  const deleteAthleteRPC = useMutation(deleteAthlete, {
    onSuccess: async () => {
      // Invalidate any listAthletes queries
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: listAthletes,
          cardinality: undefined,
        }),
      });
    },
  });
  const [isOpen, setIsOpen] = useState(false);

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
        <Dialog
          open={isOpen}
          onClose={setIsOpen}
          aria-disabled={deleteAthleteRPC.isPending}
          className="relative z-10"
        >
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
                      {deleteAthleteRPC.error !== null && (
                        <p>{deleteAthleteRPC.error.message}</p>
                      )}
                      <Button
                        onClick={(e) => {
                          e.preventDefault();
                          deleteAthleteRPC.mutate({ id: athlete.id });
                          setIsOpen(false);
                        }}
                        autoFocus
                        className="bg-gray-800 text-white rounded-md px-4 py-2 mt-4"
                        disabled={deleteAthleteRPC.isPending}
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
