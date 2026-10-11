import { SelectField, TextAreaField, TextField } from "@components/ui/fields";
import type { SelectOption } from "@components/ui/fields";
import Modal from "@components/ui/modal";
import { ErrorText } from "@components/ui/query-error";
import { primaryButton, secondaryButton } from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { RacePlanService } from "gen/races/v1/race_plans_pb";
import {
  createRace,
  updateRace,
} from "gen/races/v1/races-RaceService_connectquery";
import type { Race } from "gen/races/v1/races_pb";
import { RacePriority, RaceService } from "gen/races/v1/races_pb";
import { listRoutes } from "gen/routes/v1/routes-RouteService_connectquery";
import { isDateString } from "helpers/dates";
import { formatKm } from "helpers/format";
import { PRIORITY_OPTIONS } from "helpers/labels";
import {
  durationInput,
  kmInput,
  numberInput,
  parseDuration,
  parseKm,
  parseNumber,
} from "helpers/parse";
import { invalidateServices } from "helpers/query";
import React, { useState } from "react";

interface RaceDialogProps {
  // The race to edit, or undefined to create a race.
  race: Race | undefined;
  onClose: () => void;
}

const PRIORITY_SELECT: SelectOption[] = PRIORITY_OPTIONS.map((option) => ({
  label: option.label,
  value: String(option.value),
}));

// RaceDialog creates or edits a race.
const RaceDialog: React.FC<RaceDialogProps> = ({ race, onClose }) => {
  const queryClient = useQueryClient();
  const routes = useQuery(listRoutes);
  const [name, setName] = useState(race?.name ?? "");
  const [date, setDate] = useState(race?.date ?? "");
  const [startTime, setStartTime] = useState(race?.startTime ?? "");
  const [location, setLocation] = useState(race?.location ?? "");
  const [website, setWebsite] = useState(race?.websiteUrl ?? "");
  const [distance, setDistance] = useState(kmInput(race?.distanceMeters ?? 0));
  const [vert, setVert] = useState(numberInput(race?.elevationGainMeters ?? 0));
  const [cutoff, setCutoff] = useState(
    durationInput(race?.cutoffSeconds ?? 0, false)
  );
  const [priority, setPriority] = useState(
    String(race?.priority ?? RacePriority.A)
  );
  const [courseRouteId, setCourseRouteId] = useState(race?.courseRouteId ?? "");
  const [notes, setNotes] = useState(race?.notes ?? "");
  const [validation, setValidation] = useState("");

  const onSuccess = async () => {
    await invalidateServices(queryClient, [RaceService, RacePlanService]);
    onClose();
  };
  const create = useMutation(createRace, { onSuccess });
  const update = useMutation(updateRace, { onSuccess });

  const routeOptions: SelectOption[] = [
    { label: "No course route", value: "" },
  ];
  for (const route of routes.data?.routes ?? []) {
    routeOptions.push({
      label:
        route.stats === undefined
          ? route.name
          : `${route.name} (${formatKm(route.stats.distanceMeters)})`,
      value: route.id,
    });
  }

  const applyCourseStats = () => {
    const route = routes.data?.routes.find(
      (candidate) => candidate.id === courseRouteId
    );
    if (route?.stats === undefined) {
      return;
    }
    setDistance(kmInput(route.stats.distanceMeters));
    setVert(numberInput(Math.round(route.stats.elevationGainMeters)));
  };

  const submit = () => {
    const distanceMeters = parseKm(distance);
    const vertMeters = parseNumber(vert);
    const cutoffSeconds = parseDuration(cutoff);
    if (name.trim() === "" || !isDateString(date)) {
      setValidation("Name and date are required.");
      return;
    }
    if (distanceMeters === null || vertMeters === null) {
      setValidation("Distance and elevation must be positive numbers.");
      return;
    }
    if (cutoffSeconds === null) {
      setValidation("The time limit must be in the format h:mm.");
      return;
    }
    setValidation("");
    const fields = {
      courseRouteId,
      cutoffSeconds,
      date,
      distanceMeters,
      elevationGainMeters: Math.round(vertMeters),
      location,
      name: name.trim(),
      notes,
      priority: Number(priority),
      startTime,
      websiteUrl: website.trim(),
    };
    if (race === undefined) {
      create.mutate({ race: fields });
    } else {
      update.mutate({ race: { ...fields, id: race.id } });
    }
  };

  return (
    <Modal
      open
      onClose={onClose}
      title={race === undefined ? "Add race" : "Edit race"}
      wide
    >
      <form
        className="flex flex-col gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          submit();
        }}
      >
        <TextField
          label="Name"
          value={name}
          onChange={setName}
          placeholder="e.g. Lavaredo Ultra Trail 120K"
          autoFocus
          required
        />
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <TextField
            label="Date"
            type="date"
            value={date}
            onChange={setDate}
            required
          />
          <TextField
            label="Start time"
            type="time"
            value={startTime}
            onChange={setStartTime}
          />
          <SelectField
            label="Priority"
            value={priority}
            onChange={setPriority}
            options={PRIORITY_SELECT}
          />
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <TextField label="Location" value={location} onChange={setLocation} />
          <TextField
            label="Website"
            type="url"
            value={website}
            onChange={setWebsite}
            placeholder="https://"
          />
        </div>
        <div className="flex flex-col gap-1">
          <SelectField
            label="Course route"
            value={courseRouteId}
            onChange={setCourseRouteId}
            options={routeOptions}
          />
          <span className="text-xs text-gray-500">
            Import the race's GPX on the Routes page. The course is used for
            segment elevation in race plans.
            {courseRouteId !== "" && (
              <>
                {" "}
                <button
                  type="button"
                  className="font-medium text-emerald-700 hover:underline"
                  onClick={applyCourseStats}
                >
                  Use course distance and gain
                </button>
              </>
            )}
          </span>
        </div>
        <div className="grid grid-cols-3 gap-3">
          <TextField
            label="Distance (km)"
            value={distance}
            onChange={setDistance}
            inputMode="decimal"
          />
          <TextField
            label="Elevation gain (m)"
            value={vert}
            onChange={setVert}
            inputMode="numeric"
          />
          <TextField
            label="Time limit (h:mm)"
            value={cutoff}
            onChange={setCutoff}
            placeholder="30:00"
          />
        </div>
        <TextAreaField
          label="Notes"
          value={notes}
          onChange={setNotes}
          placeholder="Terrain, weather, qualification requirements"
        />
        {validation !== "" && (
          <p className="text-sm text-red-700">{validation}</p>
        )}
        <ErrorText error={create.error ?? update.error} />
        <div className="flex justify-end gap-2 pt-2">
          <button type="button" className={secondaryButton} onClick={onClose}>
            Cancel
          </button>
          <button
            type="submit"
            className={primaryButton}
            disabled={create.isPending || update.isPending}
          >
            Save
          </button>
        </div>
      </form>
    </Modal>
  );
};

export default RaceDialog;
