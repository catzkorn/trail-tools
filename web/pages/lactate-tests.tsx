import { LactateGuide } from "@components/lactate/lactate-bits";
import LactateTestDialog from "@components/lactate/lactate-test-dialog";
import {
  definitionLabel,
  formatMmol,
  latestThreshold,
} from "@components/lactate/lactate-utils";
import Loading from "@components/loading";
import Card from "@components/ui/card";
import { EmptyState, PageHeader, Stat } from "@components/ui/misc";
import QueryError from "@components/ui/query-error";
import {
  linkClass,
  primaryButton,
  secondaryButton,
  tableClass,
  tdClass,
  thClass,
} from "@components/ui/styles";
import { useQuery } from "@connectrpc/connect-query";
import { PlayIcon, PlusIcon } from "@heroicons/react/24/outline";
import { listLactateTests } from "gen/lactate/v1/lactate-LactateTestService_connectquery";
import type { LactateTest } from "gen/lactate/v1/lactate_pb";
import { ThresholdDefinition } from "gen/lactate/v1/lactate_pb";
import { formatDate } from "helpers/format";
import React, { useState } from "react";
import { Link } from "react-router";

// HeartRate formats a heart rate, or a dash if it wasn't found.
const heartRate = (bpm: number): string => (bpm > 0 ? `${bpm} bpm` : "–");

// Estimate returns a test's heart rate by a definition, 0 if not found.
const estimate = (test: LactateTest, definition: ThresholdDefinition): number =>
  test.thresholdEstimates.find((est) => est.definition === definition)
    ?.heartRateBpm ?? 0;

/**
 * Change formats how much a test's aerobic threshold moved since the previous
 * test that found one, e.g. "+4 bpm".
 */
const change = (tests: LactateTest[], index: number): string => {
  const current = tests[index]?.aerobicThresholdHeartRateBpm ?? 0;
  const previous = tests
    .slice(index + 1)
    .find((other) => other.aerobicThresholdHeartRateBpm > 0);
  if (current === 0 || previous === undefined) {
    return "";
  }
  const delta = current - previous.aerobicThresholdHeartRateBpm;
  return `${delta > 0 ? "+" : ""}${delta} bpm`;
};

// TestsTable lists the tests, the latest first, with their thresholds.
const TestsTable: React.FC<{ tests: LactateTest[] }> = ({ tests }) => (
  <div className="overflow-x-auto">
    <table className={tableClass}>
      <thead>
        <tr>
          <th className={thClass}>Date</th>
          <th className={thClass}>Aerobic threshold</th>
          <th className={thClass}>Change</th>
          <th className={thClass}>
            {definitionLabel(ThresholdDefinition.FIXED_2_5_MMOL)}
          </th>
          <th className={thClass}>
            {definitionLabel(ThresholdDefinition.LOWEST_PLUS_1_MMOL)}
          </th>
          <th className={thClass}>Lowest reading</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-100">
        {tests.map((test, index) => (
          <tr key={test.id}>
            <td className={tdClass}>
              <Link to={`/lactate-tests/${test.id}`} className={linkClass}>
                {formatDate(test.date)}
              </Link>
            </td>
            <td className={`${tdClass} font-semibold tabular-nums`}>
              {heartRate(test.aerobicThresholdHeartRateBpm)}
            </td>
            <td className={`${tdClass} tabular-nums`}>
              {change(tests, index)}
            </td>
            <td className={`${tdClass} tabular-nums`}>
              {heartRate(estimate(test, ThresholdDefinition.FIXED_2_5_MMOL))}
            </td>
            <td className={`${tdClass} tabular-nums`}>
              {heartRate(
                estimate(test, ThresholdDefinition.LOWEST_PLUS_1_MMOL)
              )}
            </td>
            <td className={`${tdClass} tabular-nums`}>
              {test.lowestLactateMmolPerLiter > 0
                ? formatMmol(test.lowestLactateMmolPerLiter)
                : "–"}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

// LactateTestsPage lists the user's lactate tests and their aerobic threshold.
const LactateTestsPage: React.FC = () => {
  const { isPending, isError, data, error, refetch } = useQuery(
    listLactateTests,
    {}
  );
  const [adding, setAdding] = useState(false);

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
  } else if (data.tests.length === 0) {
    content = (
      <div className="flex flex-col gap-4">
        <EmptyState title="No lactate tests yet">
          Start a guided test with your lactate meter, or add the results of a
          test you've done.
        </EmptyState>
        <Card title="How to test">
          <LactateGuide />
        </Card>
      </div>
    );
  } else {
    const threshold = latestThreshold(data.tests);
    content = (
      <div className="flex flex-col gap-4">
        <Card title="Aerobic threshold">
          {threshold === undefined ? (
            <p className="text-sm text-gray-600">
              None of your tests reached 2 mmol/L from below yet. Start the next
              test at a lower heart rate, and go on until lactate rises past 2.5
              mmol/L.
            </p>
          ) : (
            <div className="flex flex-col gap-2">
              <Stat
                label="Heart rate"
                value={`${threshold.heartRateBpm} bpm`}
                sub={
                  <>
                    From your test on{" "}
                    <Link
                      to={`/lactate-tests/${threshold.test.id}`}
                      className={linkClass}
                    >
                      {formatDate(threshold.test.date)}
                    </Link>
                  </>
                }
              />
              <p className="text-sm text-gray-600">
                Keep your base workouts below it: it's the top of your easy,
                zone 2 running.
              </p>
            </div>
          )}
        </Card>
        <Card title="Tests">
          <TestsTable tests={data.tests} />
        </Card>
        <Card title="How to test">
          <details>
            <summary className="cursor-pointer text-sm font-medium text-gray-700">
              Show the guide
            </summary>
            <div className="pt-3">
              <LactateGuide />
            </div>
          </details>
        </Card>
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Lactate tests"
        subtitle="Find your aerobic threshold, the heart rate at the top of your easy running."
        actions={
          <div className="flex gap-2">
            <button
              type="button"
              className={secondaryButton}
              onClick={() => {
                setAdding(true);
              }}
            >
              <PlusIcon className="size-4" />
              Add results
            </button>
            <Link to="/lactate-tests/new" className={primaryButton}>
              <PlayIcon className="size-4" />
              Start a test
            </Link>
          </div>
        }
      />
      {content}
      {adding && (
        <LactateTestDialog
          test={undefined}
          onClose={() => {
            setAdding(false);
          }}
        />
      )}
    </div>
  );
};

export default LactateTestsPage;
