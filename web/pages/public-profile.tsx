import Loading from "@components/loading";
import { RaceReportCard } from "@components/races/race-report-bits";
import { ReportCard } from "@components/trips/trip-bits";
import { EmptyState } from "@components/ui/misc";
import PublicFooter from "@components/ui/public-footer";
import QueryError from "@components/ui/query-error";
import { Code } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { listPublicRaceReports } from "gen/races/v1/race_reports-RaceReportService_connectquery";
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
 * PublicProfilePage lists a user's published race and trip reports, at
 * /u/<handle>.
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
  const raceReports = useQuery(listPublicRaceReports, { handle });
  const name = data?.profile?.displayName;

  useEffect(() => {
    if (name === undefined) {
      return undefined;
    }
    const previous = document.title;
    document.title = `Reports by ${name}`;
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
  const races = raceReports.data?.reports ?? [];
  const counts = [
    races.length === 1 ? "1 race report" : `${races.length} race reports`,
    reports.length === 1 ? "1 trip report" : `${reports.length} trip reports`,
  ];

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-8 pb-16">
      <header className="flex flex-col items-center gap-3 pt-6 text-center sm:pt-10">
        <div className="flex size-20 items-center justify-center rounded-full bg-ink text-2xl font-bold text-white shadow-[0_0_0_4px_#ffffff,0_0_0_8px_var(--color-ember)]">
          {initials(profile.displayName)}
        </div>
        <h1 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
          {profile.displayName}
        </h1>
        <p className="text-gray-500">
          @{profile.handle} · {counts.join(" · ")}
        </p>
      </header>
      {races.length > 0 && (
        <section className="flex flex-col gap-3">
          <h2 className="text-xl font-semibold text-gray-900">Race reports</h2>
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {races.map((report) => (
              <RaceReportCard
                key={report.id}
                report={report}
                to={`/u/${profile.handle}/races/${report.slug}`}
              />
            ))}
          </div>
        </section>
      )}
      {races.length > 0 && reports.length > 0 && (
        <h2 className="-mb-5 text-xl font-semibold text-gray-900">
          Trip reports
        </h2>
      )}
      {reports.length === 0 && races.length === 0 ? (
        <EmptyState title="No reports yet">
          Published race and trip reports will show up here.
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
      <PublicFooter />
    </div>
  );
};

export default PublicProfilePage;
