import {
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import type { ReactNode } from "react";
import React from "react";

import { iconButton } from "./styles";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  // Wide modals are used for larger forms.
  wide?: boolean;
}

const Modal: React.FC<ModalProps> = ({
  open,
  onClose,
  title,
  children,
  wide = false,
}) => (
  <Dialog open={open} onClose={onClose} className="relative z-[2000]">
    <DialogBackdrop
      transition
      className="fixed inset-0 bg-gray-900/50 transition-opacity data-[closed]:opacity-0 data-[enter]:duration-200 data-[leave]:duration-150"
    />
    <div className="fixed inset-0 w-screen overflow-y-auto">
      <div className="flex min-h-full items-start justify-center p-2 sm:items-center sm:p-4">
        <DialogPanel
          transition
          className={`relative w-full ${wide ? "max-w-3xl" : "max-w-lg"} rounded-lg bg-white shadow-xl transition-all data-[closed]:translate-y-4 data-[closed]:opacity-0 data-[enter]:duration-200 data-[leave]:duration-150`}
        >
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <DialogTitle as="h3" className="text-base font-semibold">
              {title}
            </DialogTitle>
            <button
              type="button"
              className={iconButton}
              onClick={onClose}
              aria-label="Close"
            >
              <XMarkIcon className="size-5" />
            </button>
          </div>
          <div className="px-4 py-4">{children}</div>
        </DialogPanel>
      </div>
    </div>
  </Dialog>
);

export default Modal;
