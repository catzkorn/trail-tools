import { create } from "@bufbuild/protobuf";
import { CheckboxField, TextAreaField, TextField } from "@components/ui/fields";
import { Stat } from "@components/ui/misc";
import { inputClass } from "@components/ui/styles";
import {
  CrewPlanSchema,
  MentalStrategySchema,
  PacingStrategySchema,
} from "gen/races/v1/race_plans_pb";
import { numberInput, parseNumber } from "helpers/parse";
import React, { useState } from "react";

import ListControls, { AddButton } from "./list-controls";
import type { SectionComponentProps } from "./section";
import Section, {
  EmptyText,
  TextBlock,
  nextKey,
  useSectionEdit,
} from "./section";

export const PacingSection: React.FC<SectionComponentProps> = ({
  plan,
  rules,
  onSave,
}) => {
  const edit = useSectionEdit(onSave);
  const { pacing } = plan;
  const [strategy, setStrategy] = useState("");
  const [hikeGrade, setHikeGrade] = useState("");
  const [heartRate, setHeartRate] = useState("");
  const [notes, setNotes] = useState("");

  const startEdit = () => {
    setStrategy(pacing?.strategy ?? "");
    setHikeGrade(numberInput(pacing?.hikeGradePercent ?? 0));
    setHeartRate(numberInput(pacing?.heartRateCapBpm ?? 0));
    setNotes(pacing?.notes ?? "");
    edit.start();
  };

  const submit = () => {
    const grade = parseNumber(hikeGrade);
    const bpm = parseNumber(heartRate);
    if (grade === null || bpm === null) {
      edit.setValidation("Hike grade and heart rate must be numbers.");
      return;
    }
    edit.save(
      {
        ...plan,
        pacing: create(PacingStrategySchema, {
          heartRateCapBpm: Math.round(bpm),
          hikeGradePercent: grade,
          notes,
          strategy,
        }),
      },
      "pacing"
    );
  };

  return (
    <Section
      section="pacing"
      title="Pacing"
      rules={rules}
      editable={onSave !== undefined}
      edit={edit}
      onStartEdit={startEdit}
      onSubmit={submit}
    >
      {edit.editing ? (
        <>
          <TextAreaField
            label="Strategy"
            value={strategy}
            onChange={setStrategy}
            rows={3}
            placeholder="Start conservatively, run the flats and descents, hike everything steep"
          />
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="Hike climbs steeper than (%)"
              value={hikeGrade}
              onChange={setHikeGrade}
              inputMode="decimal"
            />
            <TextField
              label="Heart rate cap, first half (bpm)"
              value={heartRate}
              onChange={setHeartRate}
              inputMode="numeric"
            />
          </div>
          <TextAreaField
            label="Effort notes"
            value={notes}
            onChange={setNotes}
            placeholder="RPE targets, how to handle the night"
          />
        </>
      ) : (
        <div className="flex flex-col gap-3">
          {pacing === undefined ||
          (pacing.strategy === "" &&
            pacing.notes === "" &&
            pacing.hikeGradePercent === 0 &&
            pacing.heartRateCapBpm === 0) ? (
            <EmptyText>No pacing strategy yet.</EmptyText>
          ) : (
            <>
              <TextBlock label="Strategy" text={pacing.strategy} />
              <div className="grid grid-cols-2 gap-3">
                {pacing.hikeGradePercent > 0 && (
                  <Stat
                    label="Hike above"
                    value={`${numberInput(pacing.hikeGradePercent)}% grade`}
                  />
                )}
                {pacing.heartRateCapBpm > 0 && (
                  <Stat
                    label="HR cap (first half)"
                    value={`${pacing.heartRateCapBpm} bpm`}
                  />
                )}
              </div>
              <TextBlock label="Effort notes" text={pacing.notes} />
            </>
          )}
        </div>
      )}
    </Section>
  );
};

export const CrewSection: React.FC<SectionComponentProps> = ({
  plan,
  rules,
  onSave,
}) => {
  const edit = useSectionEdit(onSave);
  const { crew } = plan;
  const [hasCrew, setHasCrew] = useState(false);
  const [members, setMembers] = useState("");
  const [notes, setNotes] = useState("");

  const startEdit = () => {
    setHasCrew(crew?.hasCrew ?? false);
    setMembers(crew?.members ?? "");
    setNotes(crew?.notes ?? "");
    edit.start();
  };

  return (
    <Section
      section="crew"
      title="Crew"
      rules={rules}
      editable={onSave !== undefined}
      edit={edit}
      onStartEdit={startEdit}
      onSubmit={() => {
        edit.save(
          {
            ...plan,
            crew: create(CrewPlanSchema, { hasCrew, members, notes }),
          },
          "crew"
        );
      }}
    >
      {edit.editing ? (
        <>
          <CheckboxField
            label="I have crew for this race"
            checked={hasCrew}
            onChange={setHasCrew}
          />
          {hasCrew && (
            <TextAreaField
              label="Crew members and how to reach them"
              value={members}
              onChange={setMembers}
              rows={2}
            />
          )}
          <TextAreaField
            label="Logistics"
            value={notes}
            onChange={setNotes}
            rows={4}
            placeholder="Driving between checkpoints, parking, timings"
          />
        </>
      ) : (
        <div className="flex flex-col gap-3">
          <p className="text-sm text-gray-800">
            {crew?.hasCrew === true
              ? "Crewed race."
              : "No crew: rely on aid stations and drop bags."}
          </p>
          <TextBlock label="Crew" text={crew?.members ?? ""} />
          <TextBlock label="Logistics" text={crew?.notes ?? ""} />
        </div>
      )}
    </Section>
  );
};

interface MantraDraft {
  key: number;
  text: string;
}

export const MentalSection: React.FC<SectionComponentProps> = ({
  plan,
  rules,
  onSave,
}) => {
  const edit = useSectionEdit(onSave);
  const { mental } = plan;
  const [mantras, setMantras] = useState<MantraDraft[]>([]);
  const [notes, setNotes] = useState("");

  const startEdit = () => {
    setMantras(
      (mental?.mantras ?? []).map((text) => ({ key: nextKey(), text }))
    );
    setNotes(mental?.notes ?? "");
    edit.start();
  };

  return (
    <Section
      section="mental"
      title="Mental strategy"
      rules={rules}
      editable={onSave !== undefined}
      edit={edit}
      onStartEdit={startEdit}
      onSubmit={() => {
        edit.save(
          {
            ...plan,
            mental: create(MentalStrategySchema, {
              mantras: mantras
                .map((mantra) => mantra.text.trim())
                .filter((text) => text !== ""),
              notes,
            }),
          },
          "mental"
        );
      }}
    >
      {edit.editing ? (
        <>
          <div className="flex flex-col gap-2">
            <span className="text-xs font-medium text-gray-600">Mantras</span>
            {mantras.map((mantra, index) => (
              <div key={mantra.key} className="flex items-center gap-2">
                <input
                  value={mantra.text}
                  className={inputClass}
                  aria-label="Mantra"
                  placeholder="Relentless forward progress"
                  onChange={(event) => {
                    setMantras(
                      mantras.map((other) =>
                        other.key === mantra.key
                          ? { key: other.key, text: event.target.value }
                          : other
                      )
                    );
                  }}
                />
                <ListControls
                  index={index}
                  count={mantras.length}
                  onRemove={(position) => {
                    setMantras(
                      mantras.filter((_mantra, other) => other !== position)
                    );
                  }}
                />
              </div>
            ))}
            <AddButton
              label="Add mantra"
              onClick={() => {
                setMantras([...mantras, { key: nextKey(), text: "" }]);
              }}
            />
          </div>
          <TextAreaField
            label="Notes"
            value={notes}
            onChange={setNotes}
            rows={3}
            placeholder="How to break the race down, what to think about on the hardest climb"
          />
        </>
      ) : (
        <div className="flex flex-col gap-3">
          {(mental?.mantras ?? []).length === 0 &&
          (mental?.notes ?? "") === "" ? (
            <EmptyText>No mental strategy yet.</EmptyText>
          ) : (
            <>
              {(mental?.mantras ?? []).length > 0 && (
                <ul className="flex flex-wrap gap-2">
                  {(mental?.mantras ?? []).map((mantra) => (
                    <li
                      key={mantra}
                      className="rounded-full bg-brand-50 px-3 py-1 text-sm font-medium text-brand-800 ring-1 ring-brand-200"
                    >
                      “{mantra}”
                    </li>
                  ))}
                </ul>
              )}
              <TextBlock label="Notes" text={mental?.notes ?? ""} />
            </>
          )}
        </div>
      )}
    </Section>
  );
};

export const NotesSection: React.FC<SectionComponentProps> = ({
  plan,
  rules,
  onSave,
}) => {
  const edit = useSectionEdit(onSave);
  const [notes, setNotes] = useState("");
  return (
    <Section
      section="notes"
      title="Notes"
      rules={rules}
      editable={onSave !== undefined}
      edit={edit}
      onStartEdit={() => {
        setNotes(plan.notes);
        edit.start();
      }}
      onSubmit={() => {
        edit.save({ ...plan, notes }, "notes");
      }}
    >
      {edit.editing && (
        <TextAreaField
          label="Notes"
          value={notes}
          onChange={setNotes}
          rows={5}
        />
      )}
      {!edit.editing && plan.notes === "" && <EmptyText>No notes.</EmptyText>}
      {!edit.editing && plan.notes !== "" && (
        <p className="text-sm whitespace-pre-line text-gray-800">
          {plan.notes}
        </p>
      )}
    </Section>
  );
};
