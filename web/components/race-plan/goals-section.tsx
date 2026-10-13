import { create } from "@bufbuild/protobuf";
import { TextAreaField, TextField } from "@components/ui/fields";
import { RaceGoalsSchema } from "gen/races/v1/race_plans_pb";
import type { Race } from "gen/races/v1/races_pb";
import { formatDurationLong, formatPace } from "helpers/format";
import { durationInput, parseDuration } from "helpers/parse";
import React, { useState } from "react";

import type { SectionComponentProps } from "./section";
import Section, { EmptyText, TextBlock, useSectionEdit } from "./section";

interface GoalsSectionProps extends SectionComponentProps {
  race: Race | undefined;
}

interface GoalCardProps {
  label: string;
  description: string;
  seconds: number;
  distanceMeters: number;
  className: string;
}

const GoalCard: React.FC<GoalCardProps> = ({
  label,
  description,
  seconds,
  distanceMeters,
  className,
}) => (
  <div className={`flex flex-col rounded-md border p-3 ${className}`}>
    <span className="text-xs font-semibold uppercase">{label}</span>
    <span className="text-xl font-bold tabular-nums">
      {seconds > 0 ? formatDurationLong(seconds) : "–"}
    </span>
    <span className="text-xs opacity-80">{description}</span>
    {seconds > 0 && distanceMeters > 0 && (
      <span className="text-xs tabular-nums opacity-80">
        avg {formatPace(seconds / (distanceMeters / 1000))}
      </span>
    )}
  </div>
);

const GoalsSection: React.FC<GoalsSectionProps> = ({
  plan,
  race,
  rules,
  onSave,
}) => {
  const edit = useSectionEdit(onSave);
  const { goals } = plan;
  const [goalA, setGoalA] = useState("");
  const [goalB, setGoalB] = useState("");
  const [goalC, setGoalC] = useState("");
  const [notes, setNotes] = useState("");

  const startEdit = () => {
    setGoalA(durationInput(goals?.aFinishSeconds ?? 0, true));
    setGoalB(durationInput(goals?.bFinishSeconds ?? 0, true));
    setGoalC(durationInput(goals?.cFinishSeconds ?? 0, true));
    setNotes(goals?.notes ?? "");
    edit.start();
  };

  const submit = () => {
    const aSeconds = parseDuration(goalA);
    const bSeconds = parseDuration(goalB);
    const cSeconds = parseDuration(goalC);
    if (aSeconds === null || bSeconds === null || cSeconds === null) {
      edit.setValidation("Finish times must be in the format h:mm:ss.");
      return;
    }
    edit.save(
      {
        ...plan,
        goals: create(RaceGoalsSchema, {
          aFinishSeconds: aSeconds,
          bFinishSeconds: bSeconds,
          cFinishSeconds: cSeconds,
          notes,
        }),
      },
      "goals"
    );
  };

  const distanceMeters = race?.distanceMeters ?? 0;
  return (
    <Section
      section="goals"
      title="Goals"
      rules={rules}
      editable={onSave !== undefined}
      edit={edit}
      onStartEdit={startEdit}
      onSubmit={submit}
    >
      {edit.editing ? (
        <>
          <div className="grid grid-cols-3 gap-3">
            <TextField
              label="A goal (h:mm:ss)"
              value={goalA}
              onChange={setGoalA}
              placeholder="10:30:00"
            />
            <TextField
              label="B goal (h:mm:ss)"
              value={goalB}
              onChange={setGoalB}
            />
            <TextField
              label="C goal (h:mm:ss)"
              value={goalC}
              onChange={setGoalC}
            />
          </div>
          <TextAreaField
            label="Notes"
            value={notes}
            onChange={setNotes}
            placeholder="Non-time goals, e.g. enjoy the descents"
          />
        </>
      ) : (
        <div className="flex flex-col gap-3">
          {goals === undefined ||
          (goals.aFinishSeconds === 0 &&
            goals.bFinishSeconds === 0 &&
            goals.cFinishSeconds === 0) ? (
            <EmptyText>No finish time goals yet.</EmptyText>
          ) : (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <GoalCard
                label="A goal"
                description="Everything goes right"
                seconds={goals.aFinishSeconds}
                distanceMeters={distanceMeters}
                className="border-brand-300 bg-brand-50 text-brand-800"
              />
              <GoalCard
                label="B goal"
                description="A solid day"
                seconds={goals.bFinishSeconds}
                distanceMeters={distanceMeters}
                className="border-sky-300 bg-sky-50 text-sky-900"
              />
              <GoalCard
                label="C goal"
                description="Tough day, still finishing"
                seconds={goals.cFinishSeconds}
                distanceMeters={distanceMeters}
                className="border-amber-300 bg-amber-50 text-amber-900"
              />
            </div>
          )}
          <TextBlock label="Notes" text={goals?.notes ?? ""} />
        </div>
      )}
    </Section>
  );
};

export default GoalsSection;
