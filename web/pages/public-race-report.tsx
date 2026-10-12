import Loading from "@components/loading";
import {
  Paragraphs,
  RaceComparisonView,
  SECTIONS,
} from "@components/races/race-report-bits";
import QueryError from "@components/ui/query-error";
import { Code } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { PencilSquareIcon, TrophyIcon } from "@heroicons/react/24/outline";
import { getPublicRaceReport } from "gen/races/v1/race_reports-RaceReportService_connectquery";
import { formatDate, formatDurationLong, formatKm } from "helpers/format";
import React, { useEffect } from "react";
import { Link, useParams } from "react-router";

const NotAvailable: React.FC = () => (
  <div className="py-16 text-center">
    <h1 className="text-xl font-semibold">Race report not found</h1>
    <p className="mt-2 text-gray-600">
      This race report doesn&apos;t exist or isn&apos;t published.
    </p>
  </div>
);

/**
 * PublicRaceReportPage shows a published race report, at
 * /u/<handle>/races/<slug>: the result against the plan, and the story.
 */
const PublicRaceReportPage: React.FC = () => {
  const { handle = "", slug = "" } = useParams();
  const { isPending, isError, data, error, refetch } = useQuery(
    getPublicRaceReport,
    { handle, slug },
    {
      retry: (failureCount, err) =>
        err.code !== Code.NotFound && failureCount < 3,
    }
  );
  const report = data?.report;
  useEffect(() => {
    if (report !== undefined) {
      document.title = `${report.title} · ${data?.authorName ?? ""}`;
    }
  }, [report, data?.authorName]);

  if (isPending) {
    return <Loading />;
  }
  if (isError) {
    if (error.code === Code.NotFound) {
      return <NotAvailable />;
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
  if (report === undefined) {
    return <NotAvailable />;
  }
  const facts = [
    formatDate(report.raceDate),
    report.distanceMeters > 0 ? formatKm(report.distanceMeters) : "",
    report.elevationGainMeters > 0
      ? `${Math.round(report.elevationGainMeters).toLocaleString()} m gain`
      : "",
  ].filter((part) => part !== "");
  const sections = SECTIONS.filter(
    (section) => section.key !== "summary" && report[section.key].trim() !== ""
  );

  return (
    <article className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-16">
      {!report.published && (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <span>
            This is a preview of a draft. Only you can see it until you publish
            it.
          </span>
          <Link
            to={`/race-reports/${report.id}`}
            className="inline-flex items-center gap-1 font-medium hover:underline"
          >
            <PencilSquareIcon className="size-4" />
            Edit
          </Link>
        </div>
      )}
      <header className="flex flex-col gap-2 text-center">
        <p className="text-sm font-medium tracking-wide text-emerald-700 uppercase">
          Race report · {report.raceName}
        </p>
        <h1 className="text-3xl font-bold text-gray-900 sm:text-4xl">
          {report.title}
        </h1>
        <p className="text-gray-600">
          by{" "}
          <Link to={`/u/${handle}`} className="font-medium hover:underline">
            {data.authorName}
          </Link>{" "}
          · {facts.join(" · ")}
        </p>
        {(report.finishSeconds > 0 || report.placement !== "") && (
          <p className="mt-2 inline-flex items-center justify-center gap-2 text-lg font-semibold text-gray-900">
            <TrophyIcon className="size-5 text-amber-500" />
            {[
              report.finishSeconds > 0
                ? formatDurationLong(report.finishSeconds)
                : "",
              report.placement,
            ]
              .filter((part) => part !== "")
              .join(" · ")}
          </p>
        )}
      </header>
      {report.summary.trim() !== "" && (
        <div className="flex flex-col gap-4 text-lg leading-8 text-gray-800">
          <Paragraphs text={report.summary} />
        </div>
      )}
      {data.comparison?.hasPlan === true && (
        <section className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-gray-200 sm:p-6">
          <h2 className="mb-3 text-lg font-semibold text-gray-900">
            The plan against the race
          </h2>
          <RaceComparisonView
            comparison={data.comparison}
            finishSeconds={report.finishSeconds}
          />
        </section>
      )}
      {sections.map((section) => (
        <section key={section.key} className="flex flex-col gap-3">
          <h2 className="text-xl font-semibold text-gray-900">
            {section.label}
          </h2>
          <div className="flex flex-col gap-4 text-base leading-7 text-gray-800">
            <Paragraphs text={report[section.key]} />
          </div>
        </section>
      ))}
    </article>
  );
};

export default PublicRaceReportPage;
