import UserNav from "@components/UserNav";
import { Disclosure } from "@headlessui/react";
import { User } from "gen/users/v1/users_pb";
import React from "react";
import { Link } from "react-router-dom";
import SignInDialog from "./SignInDialog";

interface HeaderProps {
  user: User | null;
}

const Header: React.FC<HeaderProps> = ({ user }) => {
  const currentHighlight = "bg-gray-900 text-white";
  const regularHighlight = "text-gray-300 hover:bg-gray-700 hover:text-white";
  const sharedClasses = "rounded-md px-3 py-2 text-sm font-medium";

  return (
    <Disclosure as="nav" className="bg-gray-800">
      <div className="mx-auto max-w-7xl px-4">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center">
            <div className="block">
              <div className="ml-10 flex items-baseline space-x-4">
                <Link
                  key="home"
                  to="/"
                  aria-current="page"
                  className={currentHighlight + " " + sharedClasses}
                >
                  Home
                </Link>
                {user === null && (
                  <SignInDialog
                    className={regularHighlight + " " + sharedClasses}
                  />
                )}
              </div>
            </div>
          </div>
          <div className="block">
            <UserNav user={user} />
          </div>
        </div>
      </div>
    </Disclosure>
  );
};

export default Header;
