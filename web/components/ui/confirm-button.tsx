import type { ReactNode } from "react";
import React, { useState } from "react";

import Modal from "./modal";
import { dangerButton, secondaryButton, secondaryDangerButton } from "./styles";

interface ConfirmButtonProps {
  // The button content.
  children: ReactNode;
  className?: string;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  disabled?: boolean;
  onConfirm: () => void;
}

// ConfirmButton asks for confirmation before running a destructive action.
const ConfirmButton: React.FC<ConfirmButtonProps> = ({
  children,
  className = secondaryDangerButton,
  title,
  message,
  confirmLabel = "Delete",
  disabled = false,
  onConfirm,
}) => {
  const [open, setOpen] = useState(false);
  const close = () => {
    setOpen(false);
  };
  return (
    <>
      <button
        type="button"
        className={className}
        disabled={disabled}
        onClick={() => {
          setOpen(true);
        }}
      >
        {children}
      </button>
      <Modal open={open} onClose={close} title={title}>
        <div className="flex flex-col gap-4">
          <div className="text-sm text-gray-700">{message}</div>
          <div className="flex justify-end gap-2">
            <button type="button" className={secondaryButton} onClick={close}>
              Cancel
            </button>
            <button
              type="button"
              className={dangerButton}
              onClick={() => {
                setOpen(false);
                onConfirm();
              }}
            >
              {confirmLabel}
            </button>
          </div>
        </div>
      </Modal>
    </>
  );
};

export default ConfirmButton;
