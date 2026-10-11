import Modal from "@components/ui/modal";
import { primaryButton, secondaryButton } from "@components/ui/styles";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { skipToken } from "@tanstack/react-query";
import { checkTrackCoverage } from "gen/routes/v1/routes-RouteService_connectquery";
import type { CheckTrackCoverageRequest } from "gen/routes/v1/routes_pb";
import { formatPercent } from "helpers/format";
import React, { useState } from "react";
import { Link } from "react-router";

// The share of a track that may be outside the map data before warning.
const MAX_UNCOVERED = 0.01;

type TrackSource = CheckTrackCoverageRequest["source"];

// CoverageMessage explains what part of a track is outside the map data.
const CoverageMessage: React.FC<{ uncovered: number }> = ({ uncovered }) => (
  <>
    {uncovered === 1
      ? "This track is outside the downloaded map areas"
      : `About ${formatPercent(uncovered)} of this track is outside the downloaded map areas`}
    , so snapping can only follow it with straight lines there. To snap it to
    trails, have an administrator download the area on the{" "}
    <Link to="/map-data" className="font-medium underline">
      Map data
    </Link>{" "}
    page first.
  </>
);

interface CoverageWarningProps {
  // The track that would be snapped, or undefined if none is chosen yet.
  source: TrackSource | undefined;
}

/**
 * CoverageWarning warns when a track that would be snapped to the map lies
 * outside the areas map data has been downloaded for, where snapping can only
 * follow it with straight lines.
 */
const CoverageWarning: React.FC<CoverageWarningProps> = ({ source }) => {
  const coverage = useQuery(
    checkTrackCoverage,
    source === undefined ? skipToken : { source },
    // The imported areas rarely change, so don't check again on every focus.
    { staleTime: 10 * 60 * 1000 }
  );
  if (coverage.data === undefined) {
    return null;
  }
  const uncovered = 1 - coverage.data.coveredFraction;
  if (uncovered < MAX_UNCOVERED) {
    return null;
  }
  return (
    <p className="rounded-md border border-amber-300 bg-amber-50 p-2 text-sm text-amber-900">
      <CoverageMessage uncovered={uncovered} />
    </p>
  );
};

export default CoverageWarning;

// A snap waiting for confirmation, as the track is outside the map data.
interface PendingSnap {
  snap: () => void;
  uncovered: number;
}

export interface CoverageCheck {
  /**
   * Checks whether a track lies outside the downloaded map areas, and snaps
   * it if it doesn't. If it does, CoverageConfirm asks whether to snap it
   * anyway.
   */
  checkAndSnap: (source: TrackSource, snap: () => void) => void;
  close: () => void;
  isPending: boolean;
  open: boolean;
  pending: PendingSnap | null;
}

/**
 * UseCoverageCheck checks a track's coverage when it's about to be snapped,
 * rather than whenever it's shown, as most tracks are never snapped. Render
 * CoverageConfirm with the result.
 */
export const useCoverageCheck = (): CoverageCheck => {
  const check = useMutation(checkTrackCoverage);
  const [pending, setPending] = useState<PendingSnap | null>(null);
  // Kept apart from pending, so the dialog keeps its text while closing.
  const [open, setOpen] = useState(false);
  return {
    checkAndSnap: (source, snap) => {
      check.mutate(
        { source },
        {
          // The check only warns, so snap anyway if it fails.
          onError: () => {
            snap();
          },
          onSuccess: (response) => {
            const uncovered = 1 - response.coveredFraction;
            if (uncovered < MAX_UNCOVERED) {
              snap();
              return;
            }
            setPending({ snap, uncovered });
            setOpen(true);
          },
        }
      );
    },
    close: () => {
      setOpen(false);
    },
    isPending: check.isPending,
    open,
    pending,
  };
};

/**
 * CoverageConfirm asks whether to snap a track that lies outside the
 * downloaded map areas, as checked by useCoverageCheck.
 */
export const CoverageConfirm: React.FC<{ check: CoverageCheck }> = ({
  check,
}) => (
  <Modal
    open={check.open}
    onClose={check.close}
    title="Create the route anyway?"
  >
    <div className="flex flex-col gap-4">
      <p className="text-sm text-gray-700">
        <CoverageMessage uncovered={check.pending?.uncovered ?? 1} />
      </p>
      <div className="flex justify-end gap-2">
        <button type="button" className={secondaryButton} onClick={check.close}>
          Cancel
        </button>
        <button
          type="button"
          className={primaryButton}
          onClick={() => {
            check.close();
            check.pending?.snap();
          }}
        >
          Create anyway
        </button>
      </div>
    </div>
  </Modal>
);
