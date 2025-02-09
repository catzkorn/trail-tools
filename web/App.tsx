import Header from "@components/Header";
import { Code, ConnectError, createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import HomePage from "@pages/HomePage";
import ProfilePage from "@pages/Profile";
import { User, UserService } from "gen/users/v1/users_pb";
import React, { useEffect, useState } from "react";
import { Route, Routes } from "react-router-dom";

const App: React.FC = () => {
  const transport = createConnectTransport({
    baseUrl: ".",
  });
  const client = createClient(UserService, transport);
  const [user, setUser] = useState<User | null | undefined>(undefined);

  useEffect(
    () => {
      async function getCurrentUser() {
        try {
          const result = await client.getCurrentUser({});
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
    <div className="min-h-full">
      <Header user={user} />
      <main>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/profile" element={<ProfilePage user={user} />} />
        </Routes>
      </main>
    </div>
  );
};

export default App;
