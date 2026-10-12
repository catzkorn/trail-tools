import Loading from "@components/loading";
import { ReportCard } from "@components/trips/trip-bits";
import { EmptyState } from "@components/ui/misc";
import QueryError from "@components/ui/query-error";
import { Code } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { getPublicProfile } from "gen/trips/v1/trips-TripReportService_connectquery";
import React, { useEffect } from "react";
import { useParams } from "react-router";

// Initials returns up to two initials of a name, for the avatar.
const initials = (name: string): string =>
  name
    .split(/\s+/u)
    .filter((part) => part !== "")
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join("");

/**
 * PublicProfilePage lists a user's published trip reports, at /u/<handle>.
 * It works without signing in.
 */
const PublicProfilePage: React.FC = () => {
  const { handle = "" } = useParams();
  const { isPending, isError, data, error, refetch } = useQuery(
    getPublicProfile,
    { handle },
    {
      retry: (failureCount, err) =>
        err.code !== Code.NotFound && failureCount < 3,
    }
  );
  const name = data?.profile?.displayName;

  useEffect(() => {
    if (name === undefined) {
      return undefined;
    }
    const previous = document.title;
    document.title = `Trip reports by ${name}`;
    return () => {
      document.title = previous;
    };
  }, [name]);

  if (isPending) {
    return <Loading />;
  }
  if (isError) {
    if (error.code === Code.NotFound) {
      return (
        <div className="py-16 text-center">
          <h1 className="text-xl font-semibold">Profile not found</h1>
          <p className="mt-2 text-gray-600">
            There is no one called @{handle} here.
          </p>
        </div>
      );
    }
    return (
      <QueryError
        error={error}
        onRetry={() => {
          void refetch();
        }}
      />
    );
  }
  const { profile, reports } = data;
  if (profile === undefined) {
    return null;
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-8 pb-16">
      <header className="flex flex-col items-center gap-3 pt-6 text-center sm:pt-10">
        <div className="flex size-20 items-center justify-center rounded-full bg-gradient-to-br from-emerald-600 to-sky-700 text-2xl font-bold text-white shadow-md">
          {initials(profile.displayName)}
        </div>
        <h1 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
          {profile.displayName}
        </h1>
        <p className="text-gray-500">
          @{profile.handle} ·{" "}
          {reports.length === 1
            ? "1 trip report"
            : `${reports.length} trip reports`}
        </p>
      </header>
      {reports.length === 0 ? (
        <EmptyState title="No trip reports yet">
          Published trip reports will show up here.
        </EmptyState>
      ) : (
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {reports.map((report) => (
            <ReportCard
              key={report.id}
              report={report}
              to={`/u/${profile.handle}/${report.slug}`}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default PublicProfilePage;
