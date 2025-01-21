import { Button } from "@headlessui/react";
import { KeyIcon } from "@heroicons/react/24/outline";
import { base64URLStringToBuffer } from "helpers/base64URL";
import React, { useTransition } from "react";
import SignInButton from "./SignInButton";

interface WebAuthnLoginResponse {
  publicKey: PublicKey;
}

interface PublicKey {
  challenge: string;

  timeout?: number;
  rpId?: string;
  allowedCredentials?: PublicKeyCredentialDescriptor[];
  user: User;
  hints?: string[];
  extensions?: object;
}

interface User {
  name: string;
  displayName: string;
  id: string;
}

interface WebAuthnLoginButtonProps {
  closeModal: () => void;
}

const WebAuthnLoginButton: React.FC<WebAuthnLoginButtonProps> = ({
  closeModal,
}) => {
  const [isPending, startTransition] = useTransition();

  const handleSubmit = () => {
    startTransition(async () => {
      try {
        const resp = await fetch("/webauthn/login/begin");
        const data = (await resp.json()) as WebAuthnLoginResponse;
        const cred = await navigator.credentials.get({
          publicKey: {
            ...data.publicKey,
            challenge: base64URLStringToBuffer(data.publicKey.challenge),
          },
        });
        if (cred === null) {
          console.error("failed to retrieve credential");
          return;
        }
        const resp2 = await fetch("/webauthn/login/finish", {
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
      <SignInButton logo={<KeyIcon />} text="Sign in with passkey" />
    </Button>
  );
};

export default WebAuthnLoginButton;
