import { UserIcon } from "@heroicons/react/24/outline";
import React from "react";

interface AvatarProps {
  avatarUrl: string;
}

export const Avatar: React.FC<AvatarProps> = ({ avatarUrl }) => {
  if (avatarUrl === "") {
    return <UserIcon className="size-8 rounded-full bg-white p-1" />;
  }
  return <img alt="" src={avatarUrl} className="size-8 rounded-full" />;
};

export default Avatar;
