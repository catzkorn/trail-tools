import React from "react";

interface AvatarProps {
  avatarUrl: string;
  // The user's name, for initials when there's no photo.
  name: string;
  // The size and text size classes, e.g. "size-20 text-2xl". Defaults to
  // "size-8 text-xs".
  className?: string;
}

// Initials returns up to two initials of a name.
const initials = (name: string): string =>
  name
    .split(/\s+/u)
    .filter((part) => part !== "")
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join("");

// Avatar shows the user's photo, or their initials without one.
export const Avatar: React.FC<AvatarProps> = ({
  avatarUrl,
  name,
  className = "size-8 text-xs",
}) => {
  if (avatarUrl === "") {
    return (
      <span
        className={`${className} flex items-center justify-center rounded-full bg-neutral-800 font-semibold text-neutral-50`}
      >
        {initials(name)}
      </span>
    );
  }
  return (
    <img
      alt=""
      src={avatarUrl}
      className={`${className} rounded-full object-cover`}
    />
  );
};

export default Avatar;
