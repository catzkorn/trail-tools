import Loading from "@components/loading";
import { ReportCard } from "@components/trips/trip-bits";
import { EmptyState, PageHeader } from "@components/ui/misc";
import QueryError from "@components/ui/query-error";
import { linkClass, secondaryButton } from "@components/ui/styles";
import { useQuery } from "@connectrpc/connect-query";
import { GlobeAltIcon } from "@heroicons/react/24/outline";
import {
  getProfile,
  listTripReports,
} from "gen/trips/v1/trips-TripReportService_connectquery";
import React from "react";
import { Link } from "react-router";

// TripReportsPage lists the user's trip reports, drafts and published.
const TripReportsPage: React.FC = () => {
  const { isPending, isError, data, error, refetch } =
    useQuery(listTripReports);
  const profile = useQuery(getProfile);
  const handle = profile.data?.profile?.handle;

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
      <EmptyState title="No trip reports yet">
        Open one of your{" "}
        <Link to="/activities" className={linkClass}>
          activities
        </Link>{" "}
        and choose &quot;Write trip report&quot; to turn it into a story with
        photos.
      </EmptyState>
    );
  } else {
    content = (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {data.reports.map((report) => (
          <ReportCard
            key={report.id}
            report={report}
            to={`/trip-reports/${report.id}`}
            showStatus
          />
        ))}
      </div>
    );
  }

  let subtitle: React.ReactNode = "Stories of your adventures, with photos.";
  if (profile.data !== undefined && handle === undefined) {
    subtitle = (
      <>
        Choose a public handle in{" "}
        <Link to="/settings" className={linkClass}>
          Settings
        </Link>{" "}
        to publish your reports.
      </>
    );
  }

  return (
    <div>
      <PageHeader
        title="Trip reports"
        subtitle={subtitle}
        actions={
          handle === undefined ? undefined : (
            <Link to={`/u/${handle}`} className={secondaryButton}>
              <GlobeAltIcon className="size-4" />
              Your public page
            </Link>
          )
        }
      />
      {content}
    </div>
  );
};

export default TripReportsPage;
