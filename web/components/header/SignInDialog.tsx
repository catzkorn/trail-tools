import React, { useEffect, useState } from "react";
import SignInButton, { GoogleLogo } from "./SignInButton";
import WebAuthnForm from "./WebAuthnForm";

const SignInDialog: React.FC = () => {
  const [isWebAuthnSupported, setIsWebAuthnSupported] = useState(false);

  // Perform WebAuthn feature detection
  useEffect(() => {
    if (typeof PublicKeyCredential === "undefined") {
      return;
    }
    void Promise.all([
      PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable(),
      PublicKeyCredential.isConditionalMediationAvailable(),
    ]).then(() => {
      setIsWebAuthnSupported(true);
    });
  });

  return (
    <div className="flex justify-center items-center">
      <div className="bg-white p-5 flex rounded-lg shadow-md flex-col gap-4 text-center">
        <h3 className="text-base font-semibold text-gray-900">Sign in</h3>
        <div className="flex flex-col gap-2">
          <a href="/oidc/login">
            <SignInButton logo={GoogleLogo} text="Sign in with Google" />
          </a>
          {/* Conditionally display the WebAuthn registration options */}
          {isWebAuthnSupported && (
            <div className="flex flex-col gap-2">
              <p>Or</p>
              <WebAuthnForm />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default SignInDialog;
