import type { Client } from "@connectrpc/connect";
import { ConnectError } from "@connectrpc/connect";
import type { AuthService } from "gen/auth/v1/auth_pb";

/**
 * Helpers for passkey sign in and registration with the WebAuthn browser API.
 * The server returns the WebAuthn options as protobuf bytes (Uint8Array), and
 * the browser returns ArrayBuffers, which are converted back to Uint8Array.
 */

export type AuthClient = Client<typeof AuthService>;

// Bytes copies protobuf bytes into a buffer the WebAuthn API accepts.
const bytes = (value: Uint8Array): Uint8Array<ArrayBuffer> =>
  new Uint8Array(value);

export const passkeysSupported = (): boolean =>
  typeof globalThis.PublicKeyCredential !== "undefined";

/**
 * IsCancelled reports whether the user cancelled the passkey prompt, or the
 * request was aborted. These are not errors worth showing.
 */
export const isCancelled = (error: unknown): boolean =>
  error instanceof DOMException &&
  (error.name === "NotAllowedError" || error.name === "AbortError");

// ErrorMessage returns a message to show for a failed passkey operation.
export const errorMessage = (error: unknown): string => {
  if (error instanceof ConnectError) {
    return error.rawMessage;
  }
  if (error instanceof DOMException && error.name === "InvalidStateError") {
    return "This device already has a passkey for this account.";
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Something went wrong.";
};

// SignInWithPasskey asks the browser for a passkey and signs in with it.
export const signInWithPasskey = async (client: AuthClient): Promise<void> => {
  const begin = await client.beginPasskeyLogin({});
  const credential = await navigator.credentials.get({
    publicKey: {
      challenge: bytes(begin.challenge),
      rpId: begin.rpId,
      userVerification: "required",
    },
  });
  if (
    !(credential instanceof PublicKeyCredential) ||
    !(credential.response instanceof AuthenticatorAssertionResponse)
  ) {
    throw new Error("The browser did not return a passkey.");
  }
  const { response } = credential;
  await client.finishPasskeyLogin({
    authenticatorData: new Uint8Array(response.authenticatorData),
    challengeId: begin.challengeId,
    clientDataJson: new Uint8Array(response.clientDataJSON),
    credentialId: new Uint8Array(credential.rawId),
    signature: new Uint8Array(response.signature),
    userHandle:
      response.userHandle === null
        ? new Uint8Array()
        : new Uint8Array(response.userHandle),
  });
};

/**
 * RegisterPasskey creates a passkey on this device and registers it, either
 * with an invitation token or, with an empty token, for the signed in user.
 * The name is used for new users; existing users keep their name.
 */
export const registerPasskey = async (
  client: AuthClient,
  invitationToken: string,
  name: string
): Promise<void> => {
  const begin = await client.beginPasskeyRegistration({ invitationToken });
  const userName = begin.userName === "" ? name : begin.userName;
  const credential = await navigator.credentials.create({
    publicKey: {
      attestation: "none",
      authenticatorSelection: {
        residentKey: "required",
        userVerification: "required",
      },
      challenge: bytes(begin.challenge),
      excludeCredentials: begin.excludeCredentialIds.map((id) => ({
        id: bytes(id),
        type: "public-key",
      })),
      pubKeyCredParams: [
        { alg: -7, type: "public-key" },
        { alg: -8, type: "public-key" },
        { alg: -257, type: "public-key" },
      ],
      rp: { id: begin.rpId, name: begin.rpName },
      user: {
        displayName: userName,
        id: bytes(begin.userHandle),
        name: userName,
      },
    },
  });
  if (
    !(credential instanceof PublicKeyCredential) ||
    !(credential.response instanceof AuthenticatorAttestationResponse)
  ) {
    throw new Error("The browser did not create a passkey.");
  }
  const { response } = credential;
  await client.finishPasskeyRegistration({
    attestationObject: new Uint8Array(response.attestationObject),
    challengeId: begin.challengeId,
    clientDataJson: new Uint8Array(response.clientDataJSON),
    transports:
      typeof response.getTransports === "function"
        ? response.getTransports()
        : [],
    userName,
  });
};
