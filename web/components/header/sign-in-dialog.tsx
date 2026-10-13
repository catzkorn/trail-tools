import { createClient } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import { KeyIcon } from "@heroicons/react/24/outline";
import { useQueryClient } from "@tanstack/react-query";
import { AuthService } from "gen/auth/v1/auth_pb";
import { UserService } from "gen/users/v1/users_pb";
import {
  conditionalMediationAvailable,
  errorMessage,
  isCancelled,
  passkeysSupported,
  signInWithPasskey,
} from "helpers/passkeys";
import { invalidateServices } from "helpers/query";
import React, {
  useEffect,
  useEffectEvent,
  useMemo,
  useRef,
  useState,
} from "react";

/**
 * SignInDialog signs in with a passkey, offered both as a button and in the
 * browser's autofill.
 */
const SignInDialog: React.FC = () => {
  const queryClient = useQueryClient();
  const transport = useTransport();
  const client = useMemo(
    () => createClient(AuthService, transport),
    [transport]
  );
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const autofillAbort = useRef<AbortController | null>(null);

  const onSignedIn = async () => {
    await invalidateServices(queryClient, [UserService]);
  };

  // BeginAutofill offers passkeys in the autofill of the input below.
  const beginAutofill = () => {
    const controller = new AbortController();
    autofillAbort.current = controller;
    conditionalMediationAvailable()
      .then(async (available) => {
        if (!available || controller.signal.aborted) {
          return;
        }
        await signInWithPasskey(client, "conditional", controller.signal);
        await onSignedIn();
      })
      .catch((error_: unknown) => {
        if (!isCancelled(error_)) {
          setError(errorMessage(error_));
        }
      });
  };
  const startAutofill = useEffectEvent(beginAutofill);

  useEffect(() => {
    startAutofill();
    return () => {
      autofillAbort.current?.abort();
    };
  }, []);

  const signIn = () => {
    // Only one WebAuthn request can be pending, so stop the autofill one.
    autofillAbort.current?.abort();
    setPending(true);
    setError("");
    signInWithPasskey(client, "optional", undefined)
      .then(onSignedIn)
      .catch((error_: unknown) => {
        if (
          error_ instanceof DOMException &&
          error_.name === "NotAllowedError"
        ) {
          /**
           * Browsers report a cancelled prompt and a passkey that couldn't
           * be used alike, so say both.
           */
          setError(
            "Signing in didn't finish: it was cancelled, or this device has no passkey for Trail Tools. Try again, or use a device with your passkey."
          );
        } else if (!isCancelled(error_)) {
          setError(errorMessage(error_));
        }
        beginAutofill();
      })
      .finally(() => {
        setPending(false);
      });
  };

  return (
    <div className="flex items-center justify-center">
      <div className="flex w-80 max-w-full flex-col gap-4 rounded-lg bg-white p-5 text-center shadow-md">
        <h3 className="text-base font-semibold text-gray-900">Sign in</h3>
        {passkeysSupported() ? (
          <>
            <input
              type="text"
              name="username"
              autoComplete="username webauthn"
              placeholder="Choose a passkey"
              aria-label="Passkey"
              className="block w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-900"
            />
            <button
              type="button"
              disabled={pending}
              className="inline-flex items-center justify-center gap-2 rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-neutral-800 disabled:opacity-50"
              onClick={signIn}
            >
              <KeyIcon className="size-5" />
              {pending ? "Waiting for passkey…" : "Sign in with a passkey"}
            </button>
          </>
        ) : (
          <p className="text-sm text-gray-600">
            This browser doesn&apos;t support passkeys.
          </p>
        )}
        {error !== "" && <p className="text-sm text-red-700">{error}</p>}
        <p className="text-xs text-gray-500">
          Trail Tools is invite only. To create an account, ask for an
          invitation link.
        </p>
      </div>
    </div>
  );
};

export default SignInDialog;
