import Loading from "@components/loading";
import RaceDialog from "@components/races/race-dialog";
import ConfirmButton from "@components/ui/confirm-button";
import { Badge, EmptyState, PageHeader } from "@components/ui/misc";
import QueryError, { ErrorText } from "@components/ui/query-error";
import {
  cardClass,
  ghostButton,
  ghostDangerButton,
  linkClass,
  primaryButton,
} from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  ClipboardDocumentListIcon,
  MapIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import {
  createRacePlan,
  listRacePlans,
} from "gen/races/v1/race_plans-RacePlanService_connectquery";
import type { RacePlanSummary } from "gen/races/v1/race_plans_pb";
import { RacePlanService } from "gen/races/v1/race_plans_pb";
import {
  deleteRace,
  listRaces,
} from "gen/races/v1/races-RaceService_connectquery";
import type { Race } from "gen/races/v1/races_pb";
import { RaceService } from "gen/races/v1/races_pb";
import { today } from "helpers/dates";
import {
  formatDaysUntil,
  formatDuration,
  formatKm,
  formatShortDate,
} from "helpers/format";
import { priorityClass, priorityLabel } from "helpers/labels";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";
import { Link, useNavigate } from "react-router";

interface RaceCardProps {
  race: Race;
  plan: RacePlanSummary | undefined;
  past: boolean;
  onEdit: () => void;
}

const RaceCard: React.FC<RaceCardProps> = ({ race, plan, past, onEdit }) => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const remove = useMutation(deleteRace, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [RaceService, RacePlanService]);
    },
  });
  const createPlan = useMutation(createRacePlan, {
    onSuccess: async (response) => {
      await invalidateServices(queryClient, [RacePlanService]);
      if (response.plan !== undefined) {
        void navigate(`/race-plans/${response.plan.id}`);
      }
    },
  });

  return (
    <div
      className={`${cardClass} flex flex-col gap-3 p-4 sm:flex-row sm:items-center ${past ? "opacity-70" : ""}`}
    >
      <div className="flex w-20 shrink-0 flex-row items-center gap-2 sm:flex-col sm:items-start sm:gap-0">
        <span className="text-sm font-semibold text-gray-900">
          {formatShortDate(race.date)}
        </span>
        <span className="text-xs text-gray-500">{race.date.slice(0, 4)}</span>
        <span className="text-xs text-gray-600">
          {formatDaysUntil(race.date)}
        </span>
      </div>
      <div className="min-w-0 grow">
        <div className="flex flex-wrap items-center gap-2">
          <Badge className={priorityClass(race.priority)}>
            {priorityLabel(race.priority)}
          </Badge>
          <h2 className="font-semibold text-gray-900">
            {race.websiteUrl === "" ? (
              race.name
            ) : (
              <a
                href={race.websiteUrl}
                target="_blank"
                rel="noreferrer"
                className="hover:underline"
              >
                {race.name}
              </a>
            )}
          </h2>
        </div>
        <p className="mt-1 text-sm text-gray-600">
          {[
            race.location,
            race.startTime === "" ? "" : `starts ${race.startTime}`,
            race.distanceMeters > 0 ? formatKm(race.distanceMeters) : "",
            race.elevationGainMeters > 0
              ? `${Math.round(race.elevationGainMeters)} m gain`
              : "",
            race.cutoffSeconds > 0
              ? `${formatDuration(race.cutoffSeconds)} limit`
              : "",
          ]
            .filter((part) => part !== "")
            .join(" · ")}
        </p>
        {race.courseRouteId !== "" && (
          <Link
            to={`/routes/${race.courseRouteId}`}
            className={`${linkClass} mt-1 inline-flex items-center gap-1 text-xs`}
          >
            <MapIcon className="size-3.5" />
            Course
          </Link>
        )}
        <ErrorText error={remove.error ?? createPlan.error} />
      </div>
      <div className="flex shrink-0 flex-wrap items-center gap-1">
        {plan === undefined ? (
          <button
            type="button"
            className={primaryButton}
            disabled={createPlan.isPending}
            onClick={() => {
              createPlan.mutate({ raceId: race.id });
            }}
          >
            <ClipboardDocumentListIcon className="size-4" />
            Create race plan
          </button>
        ) : (
          <Link to={`/race-plans/${plan.id}`} className={primaryButton}>
            <ClipboardDocumentListIcon className="size-4" />
            Race plan
            <span className="rounded bg-white/20 px-1 text-xs">
              {plan.rulesPassed}/{plan.rulesTotal}
            </span>
          </Link>
        )}
        <button
          type="button"
          className={ghostButton}
          onClick={onEdit}
          aria-label="Edit race"
        >
          <PencilSquareIcon className="size-4" />
        </button>
        <ConfirmButton
          className={ghostDangerButton}
          title="Delete race?"
          message={`This deletes "${race.name}" and its race plan.`}
          disabled={remove.isPending}
          onConfirm={() => {
            remove.mutate({ id: race.id });
          }}
        >
          <TrashIcon className="size-4" aria-label="Delete race" />
        </ConfirmButton>
      </div>
    </div>
  );
};

interface RaceEdit {
  race: Race | undefined;
}

const RacesPage: React.FC = () => {
  const { isPending, isError, data, error, refetch } = useQuery(listRaces);
  const plans = useQuery(listRacePlans);
  const [editing, setEditing] = useState<RaceEdit | null>(null);

  const planByRace = new Map<string, RacePlanSummary>();
  for (const plan of plans.data?.plans ?? []) {
    if (plan.race !== undefined) {
      planByRace.set(plan.race.id, plan);
    }
  }

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
  } else if (data.races.length === 0) {
    content = (
      <EmptyState title="No races yet">
        Add the races you are targeting. They become goals for training plans
        and get their own race day plan.
      </EmptyState>
    );
  } else {
    const now = today();
    const upcoming = data.races.filter((race) => race.date >= now);
    const past: Race[] = [];
    for (const race of data.races) {
      if (race.date < now) {
        past.unshift(race);
      }
    }
    content = (
      <div className="flex flex-col gap-6">
        <div className="flex flex-col gap-2">
          {upcoming.length === 0 && (
            <p className="text-sm text-gray-500">No upcoming races.</p>
          )}
          {upcoming.map((race) => (
            <RaceCard
              key={race.id}
              race={race}
              plan={planByRace.get(race.id)}
              past={false}
              onEdit={() => {
                setEditing({ race });
              }}
            />
          ))}
        </div>
        {past.length > 0 && (
          <div className="flex flex-col gap-2">
            <h2 className="text-sm font-semibold tracking-wide text-gray-500 uppercase">
              Past races
            </h2>
            {past.map((race) => (
              <RaceCard
                key={race.id}
                race={race}
                plan={planByRace.get(race.id)}
                past
                onEdit={() => {
                  setEditing({ race });
                }}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Races"
        subtitle="Your goal races and tune-ups, with a race day plan for each."
        actions={
          <button
            type="button"
            className={primaryButton}
            onClick={() => {
              setEditing({ race: undefined });
            }}
          >
            <PlusIcon className="size-4" />
            Add race
          </button>
        }
      />
      {content}
      {editing !== null && (
        <RaceDialog
          race={editing.race}
          onClose={() => {
            setEditing(null);
          }}
        />
      )}
    </div>
  );
};

export default RacesPage;
