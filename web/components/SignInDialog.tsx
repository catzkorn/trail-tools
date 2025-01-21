import {
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import React, { useState } from "react";
import SignInButton, { GoogleLogo } from "./SignInButton";
import WebAuthnLoginButton from "./WebAuthnLogin";
import WebAuthnRegistrationButton from "./WebAuthnRegistration";

interface SignInDialogProps {
  className: string;
}

const SignInDialog: React.FC<SignInDialogProps> = ({ className }) => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <button
        className={className}
        onClick={() => {
          setIsOpen(true);
        }}
      >
        Sign in
      </button>
      <Dialog open={isOpen} onClose={setIsOpen} className="relative z-10">
        <DialogBackdrop
          transition
          className="fixed inset-0 bg-gray-500/80 transition-opacity data-[closed]:opacity-0 data-[enter]:duration-300 data-[leave]:duration-200 data-[enter]:ease-out data-[leave]:ease-in"
        />

        <div className="fixed inset-0 z-10 w-screen overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <DialogPanel
              transition
              className="relative transform overflow-hidden rounded-lg bg-white text-left shadow-xl transition-all data-[closed]:translate-y-4 data-[closed]:opacity-0 data-[enter]:duration-300 data-[leave]:duration-200 data-[enter]:ease-out data-[leave]:ease-in"
            >
              <div className="bg-white px-4 pb-4 pt-5">
                <div>
                  <div className="flex flex-col gap-4 text-center">
                    <DialogTitle
                      as="h3"
                      className="text-base font-semibold text-gray-900"
                    >
                      Sign in
                    </DialogTitle>
                    <div className="flex flex-col gap-2">
                      <a href="/oidc/login">
                        <SignInButton
                          logo={GoogleLogo}
                          text="Sign in with Google"
                        />
                      </a>
                      {/* TODO: Only display if supported by the browser */}
                      <WebAuthnRegistrationButton
                        closeModal={() => {
                          setIsOpen(false);
                        }}
                      />
                      <WebAuthnLoginButton
                        closeModal={() => {
                          setIsOpen(false);
                        }}
                      />
                    </div>
                  </div>
                </div>
              </div>
            </DialogPanel>
          </div>
        </div>
      </Dialog>
    </>
  );
};

export default SignInDialog;
