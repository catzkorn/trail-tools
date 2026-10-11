import React, { useEffect, useState } from "react";
import { useSearchParams } from "react-router";

export interface ConnectResult {
  connected: boolean;
  message: string;
}

/**
 * UseConnectResult returns the result of connecting Strava, which the OAuth
 * callback passes to the settings page in the "strava" and "message" query
 * parameters. It removes them from the URL, so reloading the page doesn't
 * show the result again.
 */
export const useConnectResult = (): ConnectResult | null => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [result] = useState<ConnectResult | null>(() => {
    const status = searchParams.get("strava");
    if (status === null) {
      return null;
    }
    return {
      connected: status === "connected",
      message: searchParams.get("message") ?? "",
    };
  });
  useEffect(() => {
    if (searchParams.has("strava")) {
      setSearchParams(
        (params) => {
          params.delete("strava");
          params.delete("message");
          return params;
        },
        { replace: true }
      );
    }
  }, [searchParams, setSearchParams]);
  return result;
};

interface ConnectResultNoticeProps {
  result: ConnectResult | null;
}

// ConnectResultNotice shows whether connecting Strava worked.
export const ConnectResultNotice: React.FC<ConnectResultNoticeProps> = ({
  result,
}) => {
  if (result === null) {
    return null;
  }
  if (result.connected) {
    return (
      <p className="rounded-md bg-emerald-50 p-2 text-sm text-emerald-900">
        Strava is connected. Your activities are syncing.
      </p>
    );
  }
  return (
    <p className="rounded-md bg-red-50 p-2 text-sm text-red-800">
      {result.message === "" ? "Failed to connect Strava." : result.message}
    </p>
  );
};
