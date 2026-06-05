import Header from "@components/header/header";
import Loading from "@components/loading";
import { Code } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { getCurrentUser } from "gen/users/v1/users-UserService_connectquery";
import type { User } from "gen/users/v1/users_pb";
import AthletesPage from "pages/athletes";
import SettingsPage from "pages/settings";
import React from "react";
import { Route, Routes } from "react-router-dom";

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
    if (error.code === Code.Unauthenticated) {
      // No logged in user, OK!
    } else {
      // Some other error, lets pretend the user is not logged in
      console.log("Failed to get current user:", error);
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

  return (
    <div className="min-h-full flex flex-col">
      <Header user={user} />
      <main className="flex grow justify-center bg-blue-50">
        <div className="flex grow max-w-5xl px-4 py-6">
          <Routes>
            <Route path="/" element={<AthletesPage user={user} />} />
            <Route path="/athletes" element={<AthletesPage user={user} />} />
            <Route path="/settings" element={<SettingsPage user={user} />} />
          </Routes>
        </div>
      </main>
    </div>
  );
};

export default App;
