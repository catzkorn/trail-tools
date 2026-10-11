import { useQuery } from "@connectrpc/connect-query";
import { skipToken } from "@tanstack/react-query";
import { checkTrackCoverage } from "gen/routes/v1/routes-RouteService_connectquery";
import type { CheckTrackCoverageRequest } from "gen/routes/v1/routes_pb";
import { formatPercent } from "helpers/format";
import React from "react";
import { Link } from "react-router";

// The share of a track that may be outside the map data before warning.
const MAX_UNCOVERED = 0.01;

interface CoverageWarningProps {
  // The track that would be snapped, or undefined if none is chosen yet.
  source: CheckTrackCoverageRequest["source"] | undefined;
}

/**
 * CoverageWarning warns when a track that would be snapped to the map lies
 * outside the areas map data has been downloaded for, where snapping can only
 * follow it with straight lines.
 */
const CoverageWarning: React.FC<CoverageWarningProps> = ({ source }) => {
  const coverage = useQuery(
    checkTrackCoverage,
    source === undefined ? skipToken : { source }
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
      {uncovered === 1
        ? "This track is outside the downloaded map areas"
        : `About ${formatPercent(uncovered)} of this track is outside the downloaded map areas`}
      , so snapping can only follow it with straight lines there. To snap it to
      trails, have an administrator download the area on the{" "}
      <Link to="/map-data" className="font-medium underline">
        Map data
      </Link>{" "}
      page first.
    </p>
  );
};

export default CoverageWarning;
