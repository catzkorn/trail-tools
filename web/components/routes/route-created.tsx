import { ghostButton } from "@components/ui/styles";
import { ArrowPathIcon } from "@heroicons/react/24/outline";
import React from "react";
import { useLocation, useNavigate } from "react-router";

// CreatedRouteState is the navigation state of a newly created route's page.
interface CreatedRouteState {
  // Why the route differs from what was asked for, if it does.
  warning: string;
}

const isCreatedRouteState = (state: unknown): state is CreatedRouteState =>
  typeof state === "object" &&
  state !== null &&
  "warning" in state &&
  typeof state.warning === "string";

/**
 * CreatedRouteLink returns the path and navigation state that open a newly
 * created route, showing the warning about it, if any, on its page.
 */
export const createdRouteLink = (
  id: string,
  warning: string
): [string, { state: CreatedRouteState }] => [
  `/routes/${id}`,
  { state: { warning } },
];

/**
 * CreatedRouteWarning shows the warning about a newly created route, e.g. that
 * its track couldn't be snapped to trails, until it's dismissed.
 */
export const CreatedRouteWarning: React.FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const state: unknown = location.state;
  if (!isCreatedRouteState(state) || state.warning === "") {
    return null;
  }
  return (
    <div className="flex items-start justify-between gap-2 rounded-md bg-amber-50 p-2 text-sm text-amber-900">
      <span>{state.warning}</span>
      <button
        type="button"
        className={ghostButton}
        onClick={() => {
          void navigate(location.pathname + location.search, {
            replace: true,
            state: null,
          });
        }}
      >
        Dismiss
      </button>
    </div>
  );
};

/**
 * SnapProgress says that a track is being snapped to trails, which takes a
 * while for long tracks.
 */
export const SnapProgress: React.FC<{ pending: boolean }> = ({ pending }) =>
  pending ? (
    <p
      role="status"
      className="flex items-center gap-2 rounded-md bg-sky-50 p-2 text-sm text-sky-900"
    >
      <ArrowPathIcon className="size-4 shrink-0 animate-spin" />
      Snapping the track to trails. Long tracks take up to a minute.
    </p>
  ) : null;
