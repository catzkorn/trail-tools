import { Code, ConnectError, createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { User, UserService } from "gen/users/v1/users_pb";
import React, { useEffect, useState } from "react";
import { Route, Routes } from "react-router-dom";
import HomePage from "./pages/HomePage";

const App: React.FC = () => {
  const transport = createConnectTransport({
    baseUrl: ".",
  });

  const client = createClient(UserService, transport);
  const [user, setUser] = useState<User | undefined>(undefined);

  useEffect(
    () => {
      const abort = new AbortController();

      async function getCurrentUser() {
        try {
          const result = await client.getCurrentUser(
            {},
            { signal: abort.signal }
          );
          setUser(result.user);
        } catch (err: unknown) {
          const cErr = ConnectError.from(err);
          if (
            // Unauthenticated is fine, it just means the user is not logged in
            cErr.code !== Code.Unauthenticated &&
            // Canceled is fine, it just means the request was aborted
            cErr.code !== Code.Canceled
          ) {
            console.log("Failed to get current user:", cErr);
          }
        }
      }
      void getCurrentUser();
      // Abort the request if the component is unmounted
      return () => {
        abort.abort();
      };
    },
    [] // Empty dependencies array to run the effect only once
  );
  return (
    <Routes>
      <Route path="/" element={<HomePage user={user} />} />
    </Routes>
  );
};

export default App;
