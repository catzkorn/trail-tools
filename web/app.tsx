import Header from "@components/header/header";
import SignInDialog from "@components/header/sign-in-dialog";
import Loading from "@components/loading";
import { Code } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { getCurrentUser } from "gen/users/v1/users-UserService_connectquery";
import type { User } from "gen/users/v1/users_pb";
import ActivitiesPage from "pages/activities";
import ActivityPage from "pages/activity";
import AthletesPage from "pages/athletes";
import DashboardPage from "pages/dashboard";
import ForecastPage from "pages/forecast";
import InvitePage from "pages/invite";
import MapDataPage from "pages/map-data";
import OAuthConsentPage from "pages/oauth-consent";
import PlanPage from "pages/plan";
import PlansPage from "pages/plans";
import PublicProfilePage from "pages/public-profile";
import PublicTripReportPage from "pages/public-trip-report";
import RacePlanPage from "pages/race-plan";
import RacesPage from "pages/races";
import RoutePage from "pages/route";
import RoutesPage from "pages/routes";
import SettingsPage from "pages/settings";
import SharedRacePlanPage from "pages/shared-race-plan";
import TripReportPage from "pages/trip-report";
import TripReportsPage from "pages/trip-reports";
import type { ReactNode } from "react";
import React from "react";
import { Link, Route, Routes } from "react-router";

interface RequireUserProps {
  user: User | null;
  children: ReactNode;
}

// RequireUser shows a sign in prompt instead of pages that need a user.
const RequireUser: React.FC<RequireUserProps> = ({ user, children }) => {
  if (user === null) {
    return (
      <div className="flex grow flex-col items-center gap-4 py-10">
        <p className="text-lg text-gray-700">Please sign in to continue.</p>
        <SignInDialog />
      </div>
    );
  }
  return children;
};

const NotFound: React.FC = () => (
  <div className="flex grow flex-col items-center gap-2 py-10">
    <h1 className="text-xl font-semibold">Page not found</h1>
    <Link to="/" className="text-emerald-700 hover:underline">
      Back to the dashboard
    </Link>
  </div>
);

const App: React.FC = () => {
  const { isPending, isError, data, error } = useQuery(
    getCurrentUser,
    {},
    {
      retry: (failureCount, err) => {
        // Disable automatic retries on Unauthenticated errors
        // Since we expect it if the user is not logged in.
        if (err.code === Code.Unauthenticated) {
          return false;
        }
        return failureCount < 3;
      },
      // 7 24 hour days
      staleTime: 1000 * 60 * 60 * 24 * 7,
    }
  );

  if (isPending) {
    return <Loading />;
  }

  // We got a response, let's check if it's an error
  // User is null if the user is not logged in.
  let user: User | null = null;
  if (isError) {
    if (error.code !== Code.Unauthenticated) {
      // Some other error, lets pretend the user is not logged in
      console.error("Failed to get current user:", error);
    }
    user = null;
  } else if (data.user === undefined) {
    // This is a programmer error, the user should always be set in the response.
    return (
      <div className="h-screen flex items-center justify-center">
        <h1>Error: No user data, please report this bug!</h1>
      </div>
    );
  } else {
    ({ user } = data);
  }

  const protectedPage = (page: ReactNode): ReactNode => (
    <RequireUser user={user}>{page}</RequireUser>
  );

  return (
    <div className="flex min-h-full flex-col">
      <Header user={user} />
      <main className="flex grow justify-center bg-slate-50 print:bg-white">
        <div className="flex w-full max-w-7xl min-w-0 grow flex-col px-3 py-5 sm:px-4 print:px-0 print:py-0">
          <Routes>
            <Route path="/" element={<DashboardPage user={user} />} />
            <Route path="/plans" element={protectedPage(<PlansPage />)} />
            <Route path="/plans/:id" element={protectedPage(<PlanPage />)} />
            <Route path="/routes" element={protectedPage(<RoutesPage />)} />
            <Route
              path="/routes/new"
              element={protectedPage(<RoutePage isNew />)}
            />
            <Route
              path="/routes/:id"
              element={protectedPage(<RoutePage isNew={false} />)}
            />
            <Route path="/races" element={protectedPage(<RacesPage />)} />
            <Route
              path="/race-plans/:id"
              element={protectedPage(<RacePlanPage />)}
            />
            <Route
              path="/share/race-plans/:token"
              element={<SharedRacePlanPage />}
            />
            <Route path="/forecast" element={protectedPage(<ForecastPage />)} />
            <Route
              path="/activities"
              element={protectedPage(<ActivitiesPage />)}
            />
            <Route
              path="/activities/:id"
              element={protectedPage(<ActivityPage />)}
            />
            <Route
              path="/trip-reports"
              element={protectedPage(<TripReportsPage />)}
            />
            <Route
              path="/trip-reports/:id"
              element={protectedPage(<TripReportPage />)}
            />
            <Route path="/u/:handle" element={<PublicProfilePage />} />
            <Route path="/u/:handle/:slug" element={<PublicTripReportPage />} />
            <Route path="/athletes" element={<AthletesPage user={user} />} />
            <Route
              path="/settings"
              element={protectedPage(<SettingsPage user={user} />)}
            />
            <Route path="/map-data" element={protectedPage(<MapDataPage />)} />
            <Route path="/invite/:token" element={<InvitePage />} />
            <Route
              path="/oauth/authorize"
              element={protectedPage(<OAuthConsentPage />)}
            />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </div>
      </main>
    </div>
  );
};

export default App;
