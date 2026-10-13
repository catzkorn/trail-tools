import { iconButton } from "@components/ui/styles";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import React from "react";

interface ListControlsProps {
  index: number;
  count: number;
  // Moves the item by delta positions.
  onMove?: (index: number, delta: number) => void;
  onRemove: (index: number) => void;
}

// ListControls are the move and remove buttons of an editable list row.
const ListControls: React.FC<ListControlsProps> = ({
  index,
  count,
  onMove,
  onRemove,
}) => (
  <div className="flex shrink-0 items-center">
    {onMove !== undefined && (
      <>
        <button
          type="button"
          className={iconButton}
          disabled={index === 0}
          aria-label="Move up"
          onClick={() => {
            onMove(index, -1);
          }}
        >
          <ArrowUpIcon className="size-4" />
        </button>
        <button
          type="button"
          className={iconButton}
          disabled={index === count - 1}
          aria-label="Move down"
          onClick={() => {
            onMove(index, 1);
          }}
        >
          <ArrowDownIcon className="size-4" />
        </button>
      </>
    )}
    <button
      type="button"
      className={`${iconButton} hover:text-red-600`}
      aria-label="Remove"
      onClick={() => {
        onRemove(index);
      }}
    >
      <TrashIcon className="size-4" />
    </button>
  </div>
);

export default ListControls;

interface AddButtonProps {
  label: string;
  onClick: () => void;
}

export const AddButton: React.FC<AddButtonProps> = ({ label, onClick }) => (
  <button
    type="button"
    className="self-start rounded-md border border-dashed border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:border-brand-500 hover:text-brand-600"
    onClick={onClick}
  >
    + {label}
  </button>
);
