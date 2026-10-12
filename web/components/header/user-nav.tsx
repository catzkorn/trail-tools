import Avatar from "@components/header/avatar";
import { Menu, MenuButton, MenuItem, MenuItems } from "@headlessui/react";
import type { User } from "gen/users/v1/users_pb";
import React from "react";
import { Link } from "react-router";

interface UserNavProps {
  user: User;
}

const UserNav: React.FC<UserNavProps> = ({ user }) => (
  <div className="ml-4 flex items-center gap-3 md:ml-6">
    <span className="hidden items-center gap-2 text-sm text-gray-200 sm:flex">
      {user.name}
      {user.admin && (
        <span className="rounded bg-amber-400 px-1.5 py-0.5 text-[10px] font-bold tracking-wide text-gray-900 uppercase">
          Admin
        </span>
      )}
    </span>
    <Menu as="div" className="relative">
      <MenuButton className="relative flex max-w-xs items-center rounded-full bg-gray-800 text-sm focus:outline-hidden focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-gray-800">
        <span className="absolute -inset-1.5" />
        <span className="sr-only">Open user menu</span>
        <Avatar avatarUrl={user.avatarUrl} />
      </MenuButton>
      <MenuItems
        transition
        className="absolute right-0 z-[1100] mt-2 w-56 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black/5 transition focus:outline-hidden data-[closed]:scale-95 data-[closed]:transform data-[closed]:opacity-0 data-[enter]:duration-100 data-[leave]:duration-75 data-[enter]:ease-out data-[leave]:ease-in"
      >
        <div className="border-b border-gray-100 px-4 py-2">
          <p className="truncate text-sm font-medium text-gray-900">
            {user.name}
          </p>
          {user.email !== "" && (
            <p className="truncate text-xs text-gray-500">{user.email}</p>
          )}
          {user.admin && (
            <p className="mt-1 inline-block rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-bold text-amber-900 uppercase">
              Admin
            </p>
          )}
        </div>
        <MenuItem key="settings">
          <Link
            to="/settings"
            className="block px-4 py-2 text-sm text-gray-700 data-[focus]:bg-gray-100 data-[focus]:outline-hidden"
          >
            Settings
          </Link>
        </MenuItem>
        {user.admin && (
          <MenuItem key="map-data">
            <Link
              to="/map-data"
              className="block px-4 py-2 text-sm text-gray-700 data-[focus]:bg-gray-100 data-[focus]:outline-hidden"
            >
              Map data
            </Link>
          </MenuItem>
        )}
        <MenuItem key="logout">
          <a
            href="/logout"
            className="block px-4 py-2 text-sm text-gray-700 data-[focus]:bg-gray-100 data-[focus]:outline-hidden"
          >
            Sign out
          </a>
        </MenuItem>
      </MenuItems>
    </Menu>
  </div>
);

export default UserNav;
