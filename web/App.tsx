import Header from "@components/header/Header";
import { Code, ConnectError, createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AthleteService } from "gen/athletes/v1/athletes_pb";
import { User, UserService } from "gen/users/v1/users_pb";
import AthletesPage from "pages/Athletes";
import SettingsPage from "pages/Settings";
import React, { useEffect, useState } from "react";
import { Route, Routes } from "react-router-dom";

const App: React.FC = () => {
  const transport = createConnectTransport({
    baseUrl: ".",
  });
  const userServiceClient = createClient(UserService, transport);
  const athleteServiceClient = createClient(AthleteService, transport);
  const [user, setUser] = useState<User | null | undefined>(undefined);

  useEffect(
    () => {
      async function getCurrentUser() {
        try {
          const result = await userServiceClient.getCurrentUser({});
          setUser(result.user);
        } catch (err: unknown) {
          const cErr = ConnectError.from(err);
          switch (cErr.code) {
            case Code.Unauthenticated:
              // Unauthenticated is fine, it just means the user is not logged in
              setUser(null);
              break;
            case Code.Canceled:
              // Canceled is fine, it just means the request was aborted
              break;
            default:
              console.log("Failed to get current user:", cErr);
          }
        }
      }
      void getCurrentUser();
    },
    [] // Empty dependencies array to run the effect only once
  );

  if (user === undefined) {
    return (
      <div className="h-screen flex items-center justify-center">
        <h1>Loading...</h1>
      </div>
    );
  }

  return (
    <div className="min-h-full flex flex-col">
      <Header user={user} />
      <main className="flex grow justify-center bg-blue-50">
        <div className="flex grow max-w-5xl px-4 py-6">
          <Routes>
            <Route
              path="/"
              element={
                <AthletesPage client={athleteServiceClient} user={user} />
              }
            />
            <Route
              path="/athletes"
              element={
                <AthletesPage client={athleteServiceClient} user={user} />
              }
            />
            <Route path="/settings" element={<SettingsPage user={user} />} />
          </Routes>
        </div>
      </main>
    </div>
  );
};

export default App;
