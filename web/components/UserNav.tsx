import Avatar from "@components/Avatar";
import { Menu, MenuButton, MenuItem, MenuItems } from "@headlessui/react";
import { User } from "gen/users/v1/users_pb";
import React from "react";

interface UserNavProps {
  user: User | undefined;
}

const userNavigation = [{ name: "Sign out", href: "/oidc/logout" }];

const UserNav: React.FC<UserNavProps> = ({ user }) => {
  if (user === undefined) {
    // Do not render button or drop down if the user is not logged in
    return <></>;
  }
  return (
    <>
      <div className="hidden md:block">
        <div className="ml-4 flex items-center md:ml-6">
          <Menu as="div" className="relative ml-3">
            <div>
              <MenuButton className="relative flex max-w-xs items-center rounded-full bg-gray-800 text-sm focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-gray-800">
                <span className="absolute -inset-1.5" />
                <span className="sr-only">Open user menu</span>
                <Avatar avatarUrl={user.avatarUrl} />
              </MenuButton>
            </div>
            <MenuItems
              transition
              className="absolute right-0 z-10 mt-2 w-48 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black/5 transition focus:outline-none data-[closed]:scale-95 data-[closed]:transform data-[closed]:opacity-0 data-[enter]:duration-100 data-[leave]:duration-75 data-[enter]:ease-out data-[leave]:ease-in"
            >
              {userNavigation.map((item) => (
                <MenuItem key={item.name}>
                  <a
                    href={item.href}
                    className="block px-4 py-2 text-sm text-gray-700 data-[focus]:bg-gray-100 data-[focus]:outline-none"
                  >
                    {item.name}
                  </a>
                </MenuItem>
              ))}
            </MenuItems>
          </Menu>
        </div>
      </div>
    </>
  );
};

export default UserNav;
