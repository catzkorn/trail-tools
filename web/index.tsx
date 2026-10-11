import { TransportProvider } from "@connectrpc/connect-query";
import { createConnectTransport } from "@connectrpc/connect-web";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router";

import App from "./app";

const rootDiv = document.querySelector("#root");
if (rootDiv === null) {
  throw new Error("Root div not found");
}
const root = ReactDOM.createRoot(rootDiv);

/**
 * Use the origin rather than a relative URL, so that RPCs work from nested
 * client side routes such as /routes/123.
 */
const transport = createConnectTransport({
  baseUrl: globalThis.location.origin,
});

const queryClient = new QueryClient();

root.render(
  <React.StrictMode>
    <BrowserRouter>
      <TransportProvider transport={transport}>
        <QueryClientProvider client={queryClient}>
          <App />
        </QueryClientProvider>
      </TransportProvider>
    </BrowserRouter>
  </React.StrictMode>
);
