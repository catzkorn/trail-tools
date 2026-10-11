import Loading from "@components/loading";
import type { RouteDraft } from "@components/routes/route-editor";
import RouteEditor from "@components/routes/route-editor";
import RouteViewer from "@components/routes/route-viewer";
import Card from "@components/ui/card";
import { SelectField, TextField } from "@components/ui/fields";
import type { SelectOption } from "@components/ui/fields";
import QueryError, { ErrorText } from "@components/ui/query-error";
import { primaryButton } from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { listActivities } from "gen/activities/v1/activities-ActivityService_connectquery";
import {
  createRoute,
  getRoute,
  updateRoute,
} from "gen/routes/v1/routes-RouteService_connectquery";
import type { Route } from "gen/routes/v1/routes_pb";
import { RouteService, RouteSource } from "gen/routes/v1/routes_pb";
import { formatDate, formatKm } from "helpers/format";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";

/**
 * UseRouteCreated returns a function to call after creating a route, which
 * opens the new route.
 */
const useRouteCreated = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const onCreated = async (route: Route | undefined) => {
    await invalidateServices(queryClient, [RouteService]);
    if (route !== undefined) {
      void navigate(`/routes/${route.id}`);
    }
  };
  return { onCreated };
};

const ImportGpx: React.FC = () => {
  const { onCreated } = useRouteCreated();
  const [name, setName] = useState("");
  const [gpx, setGpx] = useState("");
  const [fileName, setFileName] = useState("");
  const create = useMutation(createRoute, {
    onSuccess: async (response) => {
      await onCreated(response.route);
    },
  });
  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(event) => {
        event.preventDefault();
        if (gpx === "") {
          return;
        }
        create.mutate({
          name: name.trim() === "" ? fileName : name.trim(),
          source: { case: "gpx", value: gpx },
        });
      }}
    >
      <label className="flex flex-col gap-1 text-sm">
        <span className="text-xs font-medium text-gray-600">GPX file</span>
        <input
          type="file"
          accept=".gpx,application/gpx+xml"
          className="text-sm file:mr-3 file:rounded-md file:border-0 file:bg-emerald-50 file:px-3 file:py-1.5 file:text-emerald-800 hover:file:bg-emerald-100"
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file === undefined) {
              return;
            }
            setFileName(file.name.replace(/\.gpx$/iu, ""));
            void file.text().then(setGpx);
          }}
        />
      </label>
      <TextField
        label="Name"
        value={name}
        onChange={setName}
        placeholder={fileName === "" ? "Route name" : fileName}
      />
      <ErrorText error={create.error} />
      <div>
        <button
          type="submit"
          className={primaryButton}
          disabled={gpx === "" || create.isPending}
        >
          {create.isPending ? "Importing…" : "Import route"}
        </button>
      </div>
    </form>
  );
};

const FromActivity: React.FC = () => {
  const { onCreated } = useRouteCreated();
  const activities = useQuery(listActivities, { pageSize: 50 });
  const [activityId, setActivityId] = useState("");
  const [name, setName] = useState("");
  const create = useMutation(createRoute, {
    onSuccess: async (response) => {
      await onCreated(response.route);
    },
  });
  if (activities.isPending) {
    return <Loading />;
  }
  if (activities.isError) {
    return <QueryError error={activities.error} />;
  }
  const withGps = activities.data.activities.filter(
    (activity) => activity.summaryPolyline !== ""
  );
  if (withGps.length === 0) {
    return (
      <p className="text-sm text-gray-500">
        No activities with GPS data yet. Connect Strava in{" "}
        <Link to="/settings" className="text-emerald-700 hover:underline">
          Settings
        </Link>
        .
      </p>
    );
  }
  const options: SelectOption[] = [{ label: "Choose an activity", value: "" }];
  for (const activity of withGps) {
    options.push({
      label: `${formatDate(activity.startDateLocal)} · ${activity.name} · ${formatKm(activity.distanceMeters)}`,
      value: activity.id,
    });
  }
  const selected = withGps.find((activity) => activity.id === activityId);
  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(event) => {
        event.preventDefault();
        if (selected === undefined) {
          return;
        }
        create.mutate({
          name: name.trim() === "" ? selected.name : name.trim(),
          source: { case: "activityId", value: selected.id },
        });
      }}
    >
      <SelectField
        label="Activity"
        value={activityId}
        onChange={setActivityId}
        options={options}
      />
      <TextField
        label="Name"
        value={name}
        onChange={setName}
        placeholder={selected === undefined ? "Route name" : selected.name}
      />
      <ErrorText error={create.error} />
      <div>
        <button
          type="submit"
          className={primaryButton}
          disabled={selected === undefined || create.isPending}
        >
          {create.isPending ? "Creating…" : "Create route"}
        </button>
      </div>
    </form>
  );
};

type NewMode = "draw" | "gpx" | "activity";

const NewRoute: React.FC = () => {
  const navigate = useNavigate();
  const [mode, setMode] = useState<NewMode>("draw");
  const { onCreated } = useRouteCreated();
  const create = useMutation(createRoute, {
    onSuccess: async (response) => {
      await onCreated(response.route);
    },
  });

  const tabs: { mode: NewMode; label: string }[] = [
    { label: "Draw on map", mode: "draw" },
    { label: "Import GPX", mode: "gpx" },
    { label: "From activity", mode: "activity" },
  ];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <Link to="/routes" className="text-sm text-gray-500 hover:underline">
            Routes
          </Link>
          <h1 className="text-2xl font-bold text-gray-900">New route</h1>
        </div>
        <div className="inline-flex overflow-hidden rounded-md ring-1 ring-gray-300">
          {tabs.map((tab) => (
            <button
              key={tab.mode}
              type="button"
              className={`px-3 py-1.5 text-sm ${mode === tab.mode ? "bg-gray-800 text-white" : "bg-white text-gray-700 hover:bg-gray-50"}`}
              onClick={() => {
                setMode(tab.mode);
              }}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>
      {mode === "draw" && (
        <RouteEditor
          route={undefined}
          defaultName=""
          geometryEditable
          saving={create.isPending}
          error={create.error}
          onCancel={() => {
            void navigate(-1);
          }}
          onSave={(draft: RouteDraft) => {
            create.mutate({
              description: draft.description,
              name: draft.name,
              source: {
                case: "waypoints",
                value: { profile: draft.profile, waypoints: draft.waypoints },
              },
            });
          }}
        />
      )}
      {mode === "gpx" && (
        <Card title="Import a GPX file" className="max-w-xl">
          <ImportGpx />
        </Card>
      )}
      {mode === "activity" && (
        <Card title="Create a route from an activity" className="max-w-xl">
          <FromActivity />
        </Card>
      )}
    </div>
  );
};

const ExistingRoute: React.FC = () => {
  const { id = "" } = useParams();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const { isPending, isError, data, error, refetch } = useQuery(getRoute, {
    id,
    includePoints: true,
  });
  const update = useMutation(updateRoute, {
    onSuccess: async () => {
      await invalidateServices(queryClient, [RouteService]);
      setEditing(false);
    },
  });

  if (isPending) {
    return <Loading />;
  }
  if (isError) {
    return (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  }
  const { route } = data;
  if (route === undefined) {
    return <p>Route not found.</p>;
  }
  const geometryEditable =
    route.source === RouteSource.WAYPOINTS ||
    route.source === RouteSource.UNSPECIFIED;

  if (!editing) {
    return (
      <div className="flex flex-col gap-2">
        <Link to="/routes" className="text-sm text-gray-500 hover:underline">
          Routes
        </Link>
        <RouteViewer
          route={route}
          onEdit={() => {
            update.reset();
            setEditing(true);
          }}
        />
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-2">
      {!geometryEditable && (
        <p className="rounded-md bg-amber-50 p-2 text-sm text-amber-900">
          This route was imported, so only its name and description can be
          edited.
        </p>
      )}
      <RouteEditor
        route={route}
        geometryEditable={geometryEditable}
        saving={update.isPending}
        error={update.error}
        onCancel={() => {
          setEditing(false);
        }}
        onSave={(draft: RouteDraft) => {
          update.mutate({
            description: draft.description,
            id: route.id,
            name: draft.name,
            waypoints:
              geometryEditable && draft.geometryChanged
                ? { profile: draft.profile, waypoints: draft.waypoints }
                : undefined,
          });
        }}
      />
    </div>
  );
};

interface RoutePageProps {
  isNew: boolean;
}

const RoutePage: React.FC<RoutePageProps> = ({ isNew }) =>
  isNew ? <NewRoute /> : <ExistingRoute />;

export default RoutePage;
