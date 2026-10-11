import { ErrorText } from "@components/ui/query-error";
import {
  cardClass,
  ghostButton,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import { ConnectError } from "@connectrpc/connect";
import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  PencilSquareIcon,
  XCircleIcon,
} from "@heroicons/react/20/solid";
import type { RacePlan, RuleResult } from "gen/races/v1/race_plans_pb";
import { RuleStatus } from "gen/races/v1/race_plans_pb";
import type { ReactNode } from "react";
import React, { useState } from "react";

// SaveSection saves one section (a top level RacePlan field) of a plan.
export type SaveSection = (plan: RacePlan, section: string) => Promise<void>;

export interface SectionComponentProps {
  plan: RacePlan;
  rules: RuleResult[];
  // Undefined when the plan is read only.
  onSave: SaveSection | undefined;
}

export interface SectionEdit {
  editing: boolean;
  saving: boolean;
  error: ConnectError | null;
  validation: string;
  setValidation: (message: string) => void;
  start: () => void;
  cancel: () => void;
  save: (plan: RacePlan, section: string) => void;
}

// UseSectionEdit tracks the edit state of a plan section.
export const useSectionEdit = (
  onSave: SaveSection | undefined
): SectionEdit => {
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<ConnectError | null>(null);
  const [validation, setValidation] = useState("");
  return {
    cancel: () => {
      setEditing(false);
      setError(null);
      setValidation("");
    },
    editing,
    error,
    save: (plan: RacePlan, section: string) => {
      if (onSave === undefined) {
        return;
      }
      setSaving(true);
      onSave(plan, section)
        .then(() => {
          setEditing(false);
          setError(null);
          setValidation("");
        })
        .catch((error_: unknown) => {
          setError(ConnectError.from(error_));
        })
        .finally(() => {
          setSaving(false);
        });
    },
    saving,
    setValidation,
    start: () => {
      setEditing(true);
      setError(null);
      setValidation("");
    },
    validation,
  };
};

// WorstStatus returns the most severe status of the rules.
export const worstStatus = (rules: RuleResult[]): RuleStatus => {
  let worst = RuleStatus.UNSPECIFIED;
  for (const rule of rules) {
    if (rule.status === RuleStatus.FAIL) {
      return RuleStatus.FAIL;
    }
    if (rule.status === RuleStatus.WARN) {
      worst = RuleStatus.WARN;
    } else if (
      rule.status === RuleStatus.PASS &&
      worst === RuleStatus.UNSPECIFIED
    ) {
      worst = RuleStatus.PASS;
    }
  }
  return worst;
};

interface StatusIconProps {
  status: RuleStatus;
  className?: string;
}

export const StatusIcon: React.FC<StatusIconProps> = ({
  status,
  className = "size-5",
}) => {
  switch (status) {
    case RuleStatus.PASS: {
      return (
        <CheckCircleIcon
          className={`${className} text-emerald-600`}
          aria-label="Pass"
        />
      );
    }
    case RuleStatus.WARN: {
      return (
        <ExclamationTriangleIcon
          className={`${className} text-amber-500`}
          aria-label="Warning"
        />
      );
    }
    case RuleStatus.FAIL: {
      return (
        <XCircleIcon
          className={`${className} text-red-600`}
          aria-label="Fail"
        />
      );
    }
    default: {
      return null;
    }
  }
};

export const sectionElementId = (section: string): string =>
  `section-${section}`;

interface SectionProps {
  // The RacePlan field name of the section, e.g. "checkpoints".
  section: string;
  title: string;
  rules: RuleResult[];
  editable: boolean;
  edit: SectionEdit;
  onStartEdit: () => void;
  onSubmit: () => void;
  children: ReactNode;
  // Optional content shown in the header, e.g. summary numbers.
  extra?: ReactNode;
}

// Section is the frame of a race plan section with its edit controls.
const Section: React.FC<SectionProps> = ({
  section,
  title,
  rules,
  editable,
  edit,
  onStartEdit,
  onSubmit,
  children,
  extra,
}) => {
  const status = worstStatus(rules);
  const failing = rules.filter((rule) => rule.status !== RuleStatus.PASS);
  return (
    <section
      id={sectionElementId(section)}
      className={`${cardClass} scroll-mt-4 break-inside-avoid-page`}
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-100 px-4 py-3">
        <h2 className="flex items-center gap-2 text-base font-semibold text-gray-900">
          <StatusIcon status={status} className="size-4 print:hidden" />
          {title}
        </h2>
        <div className="flex items-center gap-2">
          {extra}
          {editable && !edit.editing && (
            <button
              type="button"
              className={`${ghostButton} print:hidden`}
              onClick={onStartEdit}
            >
              <PencilSquareIcon className="size-4" />
              Edit
            </button>
          )}
        </div>
      </div>
      {editable && failing.length > 0 && (
        <ul className="flex flex-col gap-1 border-b border-gray-100 bg-amber-50/50 px-4 py-2 text-xs print:hidden">
          {failing.map((rule) => (
            <li key={rule.id} className="flex items-start gap-1.5">
              <StatusIcon status={rule.status} className="size-3.5 shrink-0" />
              <span>
                <span className="font-medium">{rule.title}:</span>{" "}
                {rule.message}
              </span>
            </li>
          ))}
        </ul>
      )}
      {edit.editing ? (
        <form
          className="flex flex-col gap-3 p-4"
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit();
          }}
        >
          {children}
          {edit.validation !== "" && (
            <p className="text-sm text-red-700">{edit.validation}</p>
          )}
          <ErrorText error={edit.error} />
          <div className="flex gap-2">
            <button
              type="submit"
              className={primaryButton}
              disabled={edit.saving}
            >
              {edit.saving ? "Saving…" : "Save"}
            </button>
            <button
              type="button"
              className={secondaryButton}
              onClick={edit.cancel}
            >
              Cancel
            </button>
          </div>
        </form>
      ) : (
        <div className="p-4">{children}</div>
      )}
    </section>
  );
};

export default Section;

let lastKey = 0;

// NextKey returns a unique key for items in editable lists.
export const nextKey = (): number => {
  lastKey += 1;
  return lastKey;
};

interface EmptyTextProps {
  children: ReactNode;
}

export const EmptyText: React.FC<EmptyTextProps> = ({ children }) => (
  <p className="text-sm text-gray-500 italic">{children}</p>
);

interface TextBlockProps {
  label: string;
  text: string;
}

// TextBlock shows a labelled block of free text, if there is any.
export const TextBlock: React.FC<TextBlockProps> = ({ label, text }) => {
  if (text === "") {
    return null;
  }
  return (
    <div>
      <p className="text-xs font-medium tracking-wide text-gray-500 uppercase">
        {label}
      </p>
      <p className="text-sm whitespace-pre-line text-gray-800">{text}</p>
    </div>
  );
};
