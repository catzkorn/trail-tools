import React from "react";

import { UserIcon } from "@heroicons/react/24/outline";

interface AvatarProps {
  avatarUrl: string;
}

export const Avatar: React.FC<AvatarProps> = ({ avatarUrl }) => {
  if (avatarUrl === "") {
    return <UserIcon className="size-8 rounded-full" />;
  }
  return <img alt="" src={avatarUrl} className="size-8 rounded-full" />;
};

export default Avatar;
