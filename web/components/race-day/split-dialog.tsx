import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { parseLocalDateTime } from "@components/forecast/forecast-utils";
import Modal from "@components/ui/modal";
import { ErrorText } from "@components/ui/query-error";
import {
  inputClass,
  labelClass,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import type { ConnectError } from "@connectrpc/connect";
import type { CheckpointSplit } from "gen/races/v1/race_plans_pb";
import React, { useState } from "react";

import { dateTimeInput, timestampInput } from "./race-day-utils";

interface TimeInputProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
}

const TimeInput: React.FC<TimeInputProps> = ({ label, value, onChange }) => (
  <div className="flex flex-col gap-1">
    <span className={labelClass}>{label}</span>
    <div className="flex gap-2">
      <input
        type="datetime-local"
        value={value}
        className={`${inputClass} text-base`}
        aria-label={label}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
      <button
        type="button"
        className={`${secondaryButton} shrink-0 px-4`}
        onClick={() => {
          onChange(dateTimeInput(Date.now()));
        }}
      >
        Now
      </button>
      <button
        type="button"
        className={`${secondaryButton} shrink-0`}
        disabled={value === ""}
        onClick={() => {
          onChange("");
        }}
      >
        Clear
      </button>
    </div>
  </div>
);

const toTimestamp = (value: string): Timestamp | undefined | null => {
  if (value === "") {
    return undefined;
  }
  const date = parseLocalDateTime(value);
  return date === null ? null : timestampFromDate(date);
};

interface SplitDialogProps {
  name: string;
  // Whether the checkpoint is the finish, which has no departure.
  finish: boolean;
  split: CheckpointSplit | undefined;
  saving: boolean;
  error: ConnectError | null;
  onSave: (
    arrival: Timestamp | undefined,
    departure: Timestamp | undefined
  ) => void;
  onClose: () => void;
}

// SplitDialog records or clears the arrival and departure at a checkpoint.
const SplitDialog: React.FC<SplitDialogProps> = ({
  name,
  finish,
  split,
  saving,
  error,
  onSave,
  onClose,
}) => {
  const [arrival, setArrival] = useState(() =>
    split?.arrivalTime === undefined
      ? dateTimeInput(Date.now())
      : timestampInput(split.arrivalTime)
  );
  const [departure, setDeparture] = useState(() =>
    timestampInput(split?.departureTime)
  );
  const [validation, setValidation] = useState("");

  return (
    <Modal open onClose={onClose} title={name === "" ? "Checkpoint" : name}>
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          const arrivalTime = toTimestamp(arrival);
          const departureTime = finish ? undefined : toTimestamp(departure);
          if (arrivalTime === null || departureTime === null) {
            setValidation("Enter a valid date and time.");
            return;
          }
          setValidation("");
          onSave(arrivalTime, departureTime);
        }}
      >
        <TimeInput label="Arrived" value={arrival} onChange={setArrival} />
        {!finish && (
          <TimeInput label="Left" value={departure} onChange={setDeparture} />
        )}
        <p className="text-xs text-gray-500">
          {finish
            ? "The time is in your local time zone. Clear it to remove the finish time."
            : "Times are in your local time zone. Clear both to remove the checkpoint's times."}
        </p>
        {validation !== "" && (
          <p className="text-sm text-red-700">{validation}</p>
        )}
        <ErrorText error={error} />
        <div className="flex justify-end gap-2">
          <button type="button" className={secondaryButton} onClick={onClose}>
            Cancel
          </button>
          <button
            type="submit"
            className={`${primaryButton} px-6`}
            disabled={saving}
          >
            Save
          </button>
        </div>
      </form>
    </Modal>
  );
};

export default SplitDialog;
