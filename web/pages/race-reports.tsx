import Loading from "@components/loading";
import { RaceReportCard } from "@components/races/race-report-bits";
import { SelectField } from "@components/ui/fields";
import { EmptyState, PageHeader } from "@components/ui/misc";
import Modal from "@components/ui/modal";
import QueryError, { ErrorText } from "@components/ui/query-error";
import {
  linkClass,
  primaryButton,
  secondaryButton,
} from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { GlobeAltIcon, PlusIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  createRaceReport,
  listRaceReports,
} from "gen/races/v1/race_reports-RaceReportService_connectquery";
import { RaceReportService } from "gen/races/v1/race_reports_pb";
import { listRaces } from "gen/races/v1/races-RaceService_connectquery";
import type { Race } from "gen/races/v1/races_pb";
import { getProfile } from "gen/trips/v1/trips-TripReportService_connectquery";
import { today } from "helpers/dates";
import { formatDate } from "helpers/format";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";
import { Link, useNavigate } from "react-router";

/**
 * NewRaceReportDialog picks one of the user's past races, the ones on the
 * calendar first, and creates a report on it.
 */
const NewRaceReportDialog: React.FC<{ onClose: () => void }> = ({
  onClose,
}) => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const races = useQuery(listRaces, {});
  const now = today();
  // Past races, the ones on the calendar first, most recent first.
  const pastOnCalendar: Race[] = [];
  const pastOther: Race[] = [];
  for (const race of races.data?.races ?? []) {
    if (race.date <= now) {
      (race.onCalendar ? pastOnCalendar : pastOther).unshift(race);
    }
  }
  const past = [...pastOnCalendar, ...pastOther];
  const [raceId, setRaceId] = useState("");
  const selected = raceId === "" ? past[0]?.id : raceId;
  const create = useMutation(createRaceReport, {
    onSuccess: async (response) => {
      await invalidateServices(queryClient, [RaceReportService]);
      if (response.report !== undefined) {
        void navigate(`/race-reports/${response.report.id}`);
      }
    },
  });

  let content: React.ReactNode = null;
  if (races.isPending) {
    content = <Loading />;
  } else if (past.length === 0) {
    content = (
      <p className="text-sm text-gray-600">
        You have no past races yet. Add the races you run in{" "}
        <Link to="/races" className={linkClass}>
          Races
        </Link>
        .
      </p>
    );
  } else {
    content = (
      <SelectField
        label="Race"
        value={selected ?? ""}
        onChange={setRaceId}
        options={past.map((race) => ({
          label: `${race.name} · ${formatDate(race.date)}`,
          value: race.id,
        }))}
      />
    );
  }

  return (
    <Modal open onClose={onClose} title="New race report">
      <div className="flex flex-col gap-3">
        <p className="text-sm text-gray-600">
          The report compares your race plan with what happened, using the
          activity on the race date.
        </p>
        {content}
        <ErrorText error={races.error ?? create.error} />
        <div className="flex justify-end gap-2 pt-2">
          <button type="button" className={secondaryButton} onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className={primaryButton}
            disabled={selected === undefined || create.isPending}
            onClick={() => {
              if (selected !== undefined) {
                create.mutate({ raceId: selected });
              }
            }}
          >
            {create.isPending ? "Creating…" : "Write race report"}
          </button>
        </div>
      </div>
    </Modal>
  );
};

// RaceReportsPage lists the user's race reports, drafts and published.
const RaceReportsPage: React.FC = () => {
  const { isPending, isError, data, error, refetch } = useQuery(
    listRaceReports,
    {}
  );
  const profile = useQuery(getProfile);
  const handle = profile.data?.profile?.handle;
  const [creating, setCreating] = useState(false);

  let content: React.ReactNode = null;
  if (isPending) {
    content = <Loading />;
  } else if (isError) {
    content = (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  } else if (data.reports.length === 0) {
    content = (
      <EmptyState title="No race reports yet">
        Write about how a race went: how you prepared, how you executed the
        plan, and what you learned, next to your result against your goals.
      </EmptyState>
    );
  } else {
    content = (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {data.reports.map((report) => (
          <RaceReportCard
            key={report.id}
            report={report}
            to={`/race-reports/${report.id}`}
            showStatus
          />
        ))}
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Race reports"
        subtitle="How your races went against the plan, and what you learned."
        actions={
          <div className="flex gap-2">
            {handle !== undefined && (
              <Link to={`/u/${handle}`} className={secondaryButton}>
                <GlobeAltIcon className="size-4" />
                Your public page
              </Link>
            )}
            <button
              type="button"
              className={primaryButton}
              onClick={() => {
                setCreating(true);
              }}
            >
              <PlusIcon className="size-4" />
              New race report
            </button>
          </div>
        }
      />
      {content}
      {creating && (
        <NewRaceReportDialog
          onClose={() => {
            setCreating(false);
          }}
        />
      )}
    </div>
  );
};

export default RaceReportsPage;
