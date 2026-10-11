import Loading from "@components/loading";
import PlanFormDialog from "@components/plans/plan-form-dialog";
import { planStatus } from "@components/plans/plan-utils";
import type { PlanStatus } from "@components/plans/plan-utils";
import { Badge, EmptyState, PageHeader } from "@components/ui/misc";
import QueryError from "@components/ui/query-error";
import { cardClass, primaryButton } from "@components/ui/styles";
import { useQuery } from "@connectrpc/connect-query";
import { PlusIcon, TrophyIcon } from "@heroicons/react/24/outline";
import { listTrainingPlans } from "gen/plans/v1/plans-TrainingPlanService_connectquery";
import { listRaces } from "gen/races/v1/races-RaceService_connectquery";
import type { Race } from "gen/races/v1/races_pb";
import { daysBetween } from "helpers/dates";
import { formatDate, formatDaysUntil } from "helpers/format";
import React, { useState } from "react";
import { Link } from "react-router";

const statusBadge = (status: PlanStatus): React.ReactNode => {
  switch (status) {
    case "active": {
      return (
        <Badge className="bg-emerald-100 text-emerald-800 ring-emerald-300">
          Active
        </Badge>
      );
    }
    case "upcoming": {
      return (
        <Badge className="bg-sky-100 text-sky-800 ring-sky-300">Upcoming</Badge>
      );
    }
    default: {
      return <Badge>Finished</Badge>;
    }
  }
};

const PlansPage: React.FC = () => {
  const { isPending, isError, data, error, refetch } =
    useQuery(listTrainingPlans);
  const races = useQuery(listRaces);
  const [creating, setCreating] = useState(false);

  const raceById = new Map<string, Race>();
  for (const race of races.data?.races ?? []) {
    raceById.set(race.id, race);
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
  } else if (data.plans.length === 0) {
    content = (
      <EmptyState title="No training plans yet">
        Create a plan leading up to your goal races, then fill in months, weeks
        and workouts.
      </EmptyState>
    );
  } else {
    content = (
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {data.plans.map((plan) => {
          const weeks = Math.ceil(
            (daysBetween(plan.startDate, plan.endDate) + 1) / 7
          );
          return (
            <Link
              key={plan.id}
              to={`/plans/${plan.id}`}
              className={`${cardClass} flex flex-col gap-2 p-4 hover:border-emerald-400`}
            >
              <div className="flex items-start justify-between gap-2">
                <h2 className="font-semibold text-gray-900">{plan.name}</h2>
                {statusBadge(planStatus(plan))}
              </div>
              <p className="text-sm text-gray-600">
                {formatDate(plan.startDate)} – {formatDate(plan.endDate)} ·{" "}
                {weeks} weeks
              </p>
              {plan.description !== "" && (
                <p className="line-clamp-2 text-sm text-gray-500">
                  {plan.description}
                </p>
              )}
              {plan.goalRaceIds.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {plan.goalRaceIds.map((id) => {
                    const race = raceById.get(id);
                    if (race === undefined) {
                      return null;
                    }
                    return (
                      <Badge
                        key={id}
                        className="bg-yellow-50 text-yellow-900 ring-yellow-300"
                      >
                        <TrophyIcon className="size-3" />
                        {race.name} · {formatDaysUntil(race.date)}
                      </Badge>
                    );
                  })}
                </div>
              )}
            </Link>
          );
        })}
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Training plans"
        subtitle="Plan months, weeks and workouts towards your goal races."
        actions={
          <button
            type="button"
            className={primaryButton}
            onClick={() => {
              setCreating(true);
            }}
          >
            <PlusIcon className="size-4" />
            New plan
          </button>
        }
      />
      {content}
      {creating && (
        <PlanFormDialog
          plan={undefined}
          onClose={() => {
            setCreating(false);
          }}
        />
      )}
    </div>
  );
};

export default PlansPage;
