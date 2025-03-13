import { Button, Input } from "@headlessui/react";
import { KeyIcon } from "@heroicons/react/24/outline";
import { base64URLStringToBuffer } from "helpers/base64URL";
import React, { ChangeEvent, useEffect, useState, useTransition } from "react";
import SignInButton from "./SignInButton";

interface WebAuthnResponse {
  publicKey: PublicKey;
}

interface PublicKey {
  pubKeyCredParams: PublicKeyCredentialParameters[];
  rp: PublicKeyCredentialRpEntity;
  user: User;
  challenge: string;

  authenticatorSelection?: AuthenticatorSelectionCriteria;
  excludeCredentials?: PublicKeyCredentialDescriptor[];
  timeout?: number;
  hints?: string[];
  attestation?: AttestationConveyancePreference;
  attestationFormats?: string[];
  extensions?: object;
}

interface User {
  name: string;
  displayName: string;
  id: string;
}

const WebAuthnForm: React.FC = () => {
  const [isPending, startTransition] = useTransition();
  const [username, setUsername] = useState("");

  useEffect(() => {
    const triggerPasskeyRetrieval = async () => {
      try {
        const resp = await fetch("/webauthn/login/begin");
        const data = (await resp.json()) as WebAuthnResponse;
        const cred = await navigator.credentials.get({
          publicKey: {
            ...data.publicKey,
            challenge: base64URLStringToBuffer(data.publicKey.challenge),
          },
          // Note: this component is only rendered if conditional mediation is available.
          mediation: "conditional",
        });
        if (cred === null) {
          console.error("failed to retrieve credential");
          return;
        }
        await fetch("/webauthn/login/finish", {
          body: JSON.stringify(cred),
          headers: {
            "Content-Type": "application/json",
          },
          method: "POST",
        });
        // Reload the window to reload with the session cookie set
        window.location.reload();
      } catch (err: unknown) {
        // Note: React strict mode will cause the first AbortError
        // from this component in dev mode. This does not happen when
        // building minified code.
        console.error(err);
      }
    };
    void triggerPasskeyRetrieval();
  });

  const handleSubmit = () => {
    startTransition(async () => {
      if (username === "") {
        return;
      }
      try {
        const resp = await fetch("/webauthn/register/begin?name=" + username);
        const data = (await resp.json()) as WebAuthnResponse;
        const cred = await navigator.credentials.create({
          publicKey: {
            ...data.publicKey,
            user: {
              ...data.publicKey.user,
              id: base64URLStringToBuffer(data.publicKey.user.id),
            },
            challenge: base64URLStringToBuffer(data.publicKey.challenge),
          },
        });
        if (cred === null) {
          console.error("failed to create credential");
          return;
        }
        await fetch("/webauthn/register/finish", {
          body: JSON.stringify(cred),
          headers: {
            "Content-Type": "application/json",
          },
          method: "POST",
        });
        // Reload the window to reload with the session cookie set
        window.location.reload();
      } catch (err) {
        console.error(err);
      }
    });
  };
  return (
    <div>
      <form className="flex flex-col gap-2">
        <Input
          type="text"
          placeholder="Enter your username"
          className="block w-full rounded-md border border-button-border-light rounded-md bg-white px-3 py-1.5 text-base text-gray-900"
          autoComplete="username webauthn"
          onChange={(e: ChangeEvent<HTMLInputElement>) => {
            setUsername(e.target.value);
          }}
        />
        <Button onClick={handleSubmit} disabled={isPending}>
          <SignInButton logo={<KeyIcon />} text="Register with a passkey" />
        </Button>
      </form>
    </div>
  );
};

export default WebAuthnForm;
