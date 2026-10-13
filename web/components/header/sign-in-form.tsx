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
 * SignInForm signs in with a passkey, offered both as a button and in the
 * browser's autofill.
 */
const SignInForm: React.FC = () => {
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
            "Signing in didn't finish: it was cancelled, or this device has no passkey for trailrunn.ing. Try again, or use a device with your passkey."
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
    <div className="flex w-full max-w-sm flex-col gap-4">
      <h2 className="text-2xl font-bold text-ink">Sign in</h2>
      {passkeysSupported() ? (
        <>
          <label className="flex flex-col gap-1.5 text-[13px] font-medium text-stone-600">
            Passkey
            <input
              type="text"
              name="username"
              autoComplete="username webauthn"
              placeholder="Choose a passkey"
              className="block w-full rounded-lg border border-stone-300 bg-white px-3 py-2.5 text-[15px] font-normal text-ink focus:border-brand-500 focus:ring-brand-500"
            />
          </label>
          <button
            type="button"
            disabled={pending}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-ink px-4 py-3 text-[15px] font-semibold text-neutral-50 hover:bg-neutral-800 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-ember focus-visible:ring-offset-2 disabled:opacity-50"
            onClick={signIn}
          >
            <KeyIcon className="size-[18px] text-ember" />
            {pending ? "Waiting for passkey…" : "Sign in with a passkey"}
          </button>
        </>
      ) : (
        <p className="text-sm text-stone-600">
          This browser doesn&apos;t support passkeys.
        </p>
      )}
      {error !== "" && <p className="text-[13px] text-brand-700">{error}</p>}
      <p className="border-t border-stone-200 pt-3 text-[13px] leading-normal text-stone-500">
        trailrunn.ing is invite only. To create an account, ask for an
        invitation link.
      </p>
    </div>
  );
};

export default SignInForm;
