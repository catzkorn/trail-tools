import { TransportProvider } from "@connectrpc/connect-query";
import { createConnectTransport } from "@connectrpc/connect-web";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";

const rootDiv = document.getElementById("root");
if (rootDiv === null) {
  throw new Error("Root div not found");
}
const root = ReactDOM.createRoot(rootDiv);

const transport = createConnectTransport({
  baseUrl: ".",
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
