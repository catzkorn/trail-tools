import { Button } from "@headlessui/react";
import { KeyIcon } from "@heroicons/react/24/outline";
import { base64URLStringToBuffer } from "helpers/base64URL";
import React, { useTransition } from "react";
import SignInButton from "./SignInButton";

interface WebAuthnRegistrationResponse {
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

interface WebAuthnRegistrationButtonProps {
  closeModal: () => void;
}

const WebAuthnRegistrationButton: React.FC<WebAuthnRegistrationButtonProps> = ({
  closeModal,
}) => {
  const [isPending, startTransition] = useTransition();

  const handleSubmit = () => {
    startTransition(async () => {
      try {
        const resp = await fetch("/webauthn/register/begin?name=johan");
        const data = (await resp.json()) as WebAuthnRegistrationResponse;
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
        const resp2 = await fetch("/webauthn/register/finish", {
          body: JSON.stringify(cred),
          headers: {
            "Content-Type": "application/json",
          },
          method: "POST",
        });
        console.log(resp2.status);
        closeModal();
      } catch (err) {
        console.error(err);
      }
    });
  };
  return (
    <Button onClick={handleSubmit} disabled={isPending}>
      <SignInButton logo={<KeyIcon />} text="Register a passkey" />
    </Button>
  );
};

export default WebAuthnRegistrationButton;
