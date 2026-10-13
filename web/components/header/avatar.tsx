import { UserIcon } from "@heroicons/react/24/outline";
import React from "react";

interface AvatarProps {
  avatarUrl: string;
  // The size classes, e.g. "size-20". Defaults to "size-8".
  className?: string;
}

export const Avatar: React.FC<AvatarProps> = ({
  avatarUrl,
  className = "size-8",
}) => {
  if (avatarUrl === "") {
    return (
      <UserIcon
        className={`${className} rounded-full bg-white p-1 text-gray-900`}
      />
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
