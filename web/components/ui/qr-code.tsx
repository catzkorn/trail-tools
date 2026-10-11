import React, { useMemo } from "react";
import { encode } from "uqr";

interface QrCodeProps {
  value: string;
  size?: number;
}

// QrCode renders a QR code of the value as an SVG.
const QrCode: React.FC<QrCodeProps> = ({ value, size = 192 }) => {
  const { path, modules } = useMemo(() => {
    const qr = encode(value, { border: 2, ecc: "M" });
    const commands: string[] = [];
    for (const [row, cells] of qr.data.entries()) {
      for (const [column, dark] of cells.entries()) {
        if (dark) {
          commands.push(`M${column},${row}h1v1h-1z`);
        }
      }
    }
    return { modules: qr.size, path: commands.join("") };
  }, [value]);
  return (
    <svg
      viewBox={`0 0 ${modules} ${modules}`}
      width={size}
      height={size}
      shapeRendering="crispEdges"
      role="img"
      aria-label="QR code of the link"
      className="rounded-md bg-white"
    >
      <rect width={modules} height={modules} fill="#ffffff" />
      <path d={path} fill="#111827" />
    </svg>
  );
};

export default QrCode;
