import { TextAreaField, TextField } from "@components/ui/fields";
import Modal from "@components/ui/modal";
import { ErrorText } from "@components/ui/query-error";
import { primaryButton, secondaryButton } from "@components/ui/styles";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  createLactateTest,
  updateLactateTest,
} from "gen/lactate/v1/lactate-LactateTestService_connectquery";
import type { LactateTest } from "gen/lactate/v1/lactate_pb";
import { LactateTestService } from "gen/lactate/v1/lactate_pb";
import { isDateString, today } from "helpers/dates";
import { numberInput, parseNumber } from "helpers/parse";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";
import { useNavigate } from "react-router";

import type { StageInput } from "./lactate-bits";
import { parseStages, stageInputs, StagesEditor } from "./lactate-bits";

interface LactateTestDialogProps {
  // The test to edit, or undefined to add the results of a test.
  test: LactateTest | undefined;
  onClose: () => void;
}

// LactateTestDialog adds or edits the results of a lactate test.
const LactateTestDialog: React.FC<LactateTestDialogProps> = ({
  test,
  onClose,
}) => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [date, setDate] = useState(test?.date ?? today());
  const [resting, setResting] = useState(
    numberInput(test?.restingLactateMmolPerLiter ?? 0)
  );
  const [notes, setNotes] = useState(test?.notes ?? "");
  const [stages, setStages] = useState<StageInput[]>(
    test === undefined
      ? [{ heartRate: "100", lactate: "" }]
      : stageInputs(test.stages)
  );
  const [validation, setValidation] = useState("");

  const create = useMutation(createLactateTest, {
    onSuccess: async (response) => {
      await invalidateServices(queryClient, [LactateTestService]);
      onClose();
      if (response.test !== undefined) {
        void navigate(`/lactate-tests/${response.test.id}`);
      }
    },
  });
  const update = useMutation(updateLactateTest, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [LactateTestService]);
      onClose();
    },
  });
  const pending = create.isPending || update.isPending;

  const submit = (): void => {
    if (!isDateString(date)) {
      setValidation("The date must be YYYY-MM-DD.");
      return;
    }
    const restingMmol = parseNumber(resting.replace(",", "."));
    if (restingMmol === null) {
      setValidation("The resting lactate must be a number of mmol/L.");
      return;
    }
    const parsed = parseStages(stages);
    if (typeof parsed === "string") {
      setValidation(parsed);
      return;
    }
    setValidation("");
    const fields = {
      date,
      notes,
      restingLactateMmolPerLiter: restingMmol,
      stages: parsed,
    };
    if (test === undefined) {
      create.mutate({ test: fields });
    } else {
      update.mutate({ test: { ...fields, id: test.id } });
    }
  };

  return (
    <Modal
      open
      onClose={onClose}
      title={test === undefined ? "Add test results" : "Edit test"}
    >
      <form
        className="flex flex-col gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          submit();
        }}
      >
        <div className="grid grid-cols-2 gap-3">
          <TextField label="Date" type="date" value={date} onChange={setDate} />
          <TextField
            label="Resting lactate (mmol/L)"
            value={resting}
            onChange={setResting}
            inputMode="decimal"
            hint="Before the warm-up. Empty if not measured."
          />
        </div>
        <StagesEditor stages={stages} onChange={setStages} />
        <TextAreaField
          label="Notes"
          value={notes}
          onChange={setNotes}
          rows={3}
          placeholder="e.g. treadmill at 10% incline, 2 hours after breakfast"
        />
        {validation !== "" && (
          <p className="text-sm text-red-700">{validation}</p>
        )}
        <ErrorText error={create.error ?? update.error} />
        <div className="flex justify-end gap-2 pt-2">
          <button type="button" className={secondaryButton} onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className={primaryButton} disabled={pending}>
            {pending ? "Saving…" : "Save"}
          </button>
        </div>
      </form>
    </Modal>
  );
};

export default LactateTestDialog;
