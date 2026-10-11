import { CopyField } from "@components/ui/misc";
import QrCode from "@components/ui/qr-code";
import React from "react";

interface InvitationLinkProps {
  url: string;
  description: string;
  onDone: () => void;
}

// InvitationLink shows a new invitation link to copy or scan.
const InvitationLink: React.FC<InvitationLinkProps> = ({
  url,
  description,
  onDone,
}) => (
  <div className="flex flex-col gap-3 rounded-md border border-emerald-300 bg-emerald-50 p-3">
    <p className="text-sm text-emerald-900">{description}</p>
    <CopyField value={url} />
    <div className="flex flex-wrap items-end gap-3">
      <QrCode value={url} />
      <p className="max-w-60 text-xs text-gray-600">
        Scan the code with the other device&apos;s camera. The link can only be
        used once and is only shown now.
      </p>
    </div>
    <button
      type="button"
      className="self-start text-sm font-medium text-emerald-800 hover:underline"
      onClick={onDone}
    >
      Done
    </button>
  </div>
);

export default InvitationLink;
