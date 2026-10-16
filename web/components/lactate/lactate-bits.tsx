import { inputClass, linkClass } from "@components/ui/styles";
import type { LactateStage } from "gen/lactate/v1/lactate_pb";
import React from "react";

// PREPARATION is what to do before a test, so tests are comparable.
export const PREPARATION = [
  "No food for 60 minutes before, and no caffeine for 4 hours.",
  "Rested: no hard training the day before.",
  "The same conditions as your last test: the same route or treadmill incline, and about the same time of day.",
  "A chest strap heart rate monitor, which reacts faster and more reliably than a watch.",
  "Your lactate meter, test strips, lancets, alcohol wipes and tissues, within reach.",
];

// SAMPLING_TIPS is how to take a sample that reads true.
export const SAMPLING_TIPS = [
  "Use an earlobe, or a fingertip warmed up first.",
  "Clean the spot with an alcohol wipe and let it dry.",
  "Prick it, and wipe away the first drop: it's diluted with tissue fluid.",
  "Squeeze out a second drop about 3 mm across, and touch only its top with the strip.",
  "If a reading looks wrong, hold the same heart rate and measure again.",
];

// SamplingTips lists how to take a sample.
export const SamplingTips: React.FC = () => (
  <ul className="list-disc space-y-1 pl-5 text-sm text-gray-700">
    {SAMPLING_TIPS.map((tip) => (
      <li key={tip}>{tip}</li>
    ))}
  </ul>
);

/**
 * LactateGuide explains how a lactate test works and how to do one, as in
 * Uphill Athlete's DIY guide.
 */
export const LactateGuide: React.FC = () => (
  <div className="flex flex-col gap-4 text-sm leading-6 text-gray-700">
    <p>
      Your aerobic threshold (LT1, or AeT) is the heart rate where your blood
      lactate starts to rise above its baseline. Below it, you run mostly on
      fat; it's the top of your easy, zone 2 running. A lactate test finds it by
      measuring your lactate at a series of increasing heart rates. The app uses
      2 mmol/L, as{" "}
      <a
        href="https://uphillathlete.com/blood-lactate-testing/"
        target="_blank"
        rel="noreferrer"
        className={linkClass}
      >
        Uphill Athlete
      </a>{" "}
      does, and shows two other common definitions to compare.
    </p>
    <div>
      <h3 className="font-semibold text-gray-900">Before the test</h3>
      <ul className="mt-1 list-disc space-y-1 pl-5">
        {PREPARATION.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
    <div>
      <h3 className="font-semibold text-gray-900">The test</h3>
      <ol className="mt-1 list-decimal space-y-1 pl-5">
        <li>
          Measure your resting lactate, your baseline, after a few minutes
          sitting still. Above about 2 mmol/L at rest suggests a recent meal or
          hard training.
        </li>
        <li>
          Warm up for 15 to 20 minutes, very easy, until you're lightly
          sweating.
        </li>
        <li>
          Run stages at increasing heart rates, e.g. 100, 110, 120 bpm, starting
          lower than you think you need to. Hold each for 3 to 5 minutes, then
          measure your lactate.
        </li>
        <li>
          Stop once lactate is at least 2.5 mmol/L and 1 mmol/L above your
          lowest reading: then every definition has been found.
        </li>
      </ol>
    </div>
    <div>
      <h3 className="font-semibold text-gray-900">Taking a sample</h3>
      <div className="mt-1">
        <SamplingTips />
      </div>
    </div>
    <p>
      Lactate often dips in the first stages after the warm-up, which is a good
      sign. Test again every two to three months, the same way each time, to see
      your threshold move with training.
    </p>
  </div>
);

interface StagesEditorProps {
  stages: StageInput[];
  onChange: (stages: StageInput[]) => void;
}

// StageInput is a stage being edited, as the text of its inputs.
export interface StageInput {
  heartRate: string;
  lactate: string;
}

// StageInputs converts stages to inputs.
export const stageInputs = (stages: LactateStage[]): StageInput[] =>
  stages.map((stage) => ({
    heartRate: String(stage.heartRateBpm),
    lactate: String(stage.lactateMmolPerLiter),
  }));

/**
 * StagesEditor edits a test's stages as rows of a heart rate and a lactate
 * reading, adding a row at the next heart rate.
 */
export const StagesEditor: React.FC<StagesEditorProps> = ({
  stages,
  onChange,
}) => {
  const update = (index: number, change: Partial<StageInput>): void => {
    onChange(
      stages.map((stage, other) =>
        other === index ? { ...stage, ...change } : stage
      )
    );
  };
  return (
    <div className="flex flex-col gap-2">
      <div className="grid grid-cols-[1fr_1fr_auto] gap-2 text-xs font-medium text-gray-600">
        <span>Heart rate (bpm)</span>
        <span>Lactate (mmol/L)</span>
        <span className="w-14" />
      </div>
      {stages.map((stage, index) => (
        // eslint-disable-next-line react/no-array-index-key -- rows have no identity
        <div key={index} className="grid grid-cols-[1fr_1fr_auto] gap-2">
          <input
            className={inputClass}
            inputMode="numeric"
            aria-label={`Stage ${index + 1} heart rate`}
            value={stage.heartRate}
            onChange={(event) => {
              update(index, { heartRate: event.target.value });
            }}
          />
          <input
            className={inputClass}
            inputMode="decimal"
            aria-label={`Stage ${index + 1} lactate`}
            value={stage.lactate}
            onChange={(event) => {
              update(index, { lactate: event.target.value });
            }}
          />
          <button
            type="button"
            className="w-14 text-sm text-gray-500 hover:text-red-700"
            onClick={() => {
              onChange(stages.filter((_stage, other) => other !== index));
            }}
          >
            Remove
          </button>
        </div>
      ))}
      <button
        type="button"
        className={`${linkClass} self-start text-sm`}
        onClick={() => {
          const last = Number(stages.at(-1)?.heartRate ?? "");
          onChange([
            ...stages,
            {
              heartRate:
                Number.isFinite(last) && last > 0 ? String(last + 10) : "",
              lactate: "",
            },
          ]);
        }}
      >
        Add a stage
      </button>
    </div>
  );
};

/**
 * ParseStages parses stage inputs, returning an error message for the first
 * invalid one.
 */
export const parseStages = (
  inputs: StageInput[]
): { heartRateBpm: number; lactateMmolPerLiter: number }[] | string => {
  const stages: { heartRateBpm: number; lactateMmolPerLiter: number }[] = [];
  for (const [index, input] of inputs.entries()) {
    const heartRate = Number(input.heartRate.trim());
    const lactate = Number(input.lactate.trim().replace(",", "."));
    if (!Number.isInteger(heartRate) || heartRate <= 0) {
      return `Stage ${index + 1} needs a heart rate in bpm.`;
    }
    if (!Number.isFinite(lactate) || lactate <= 0) {
      return `Stage ${index + 1} needs a lactate reading in mmol/L.`;
    }
    stages.push({ heartRateBpm: heartRate, lactateMmolPerLiter: lactate });
  }
  return stages;
};
