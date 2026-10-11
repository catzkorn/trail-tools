import React, { useEffect, useState } from "react";

import WebAuthnForm from "./web-authn-form";

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
          {isWebAuthnSupported ? (
            <WebAuthnForm />
          ) : (
            <p>This browser doesn&apos;t support passkeys.</p>
          )}
        </div>
      </div>
    </div>
  );
};

export default SignInDialog;
