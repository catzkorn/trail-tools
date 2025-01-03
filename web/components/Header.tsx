import UserNav from "@components/UserNav";
import { Disclosure } from "@headlessui/react";
import { User } from "gen/users/v1/users_pb";
import React from "react";
import { Link } from "react-router-dom";

const navigation = [{ name: "Home", href: "/", current: true }];
function classNames(...classes: string[]) {
  return classes.filter(Boolean).join(" ");
}

interface HeaderProps {
  user: User | null;
}

const Header: React.FC<HeaderProps> = ({ user }) => {
  return (
    <Disclosure as="nav" className="bg-gray-800">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center">
            <div className="block">
              <div className="ml-10 flex items-baseline space-x-4">
                {navigation.map((item) => (
                  <Link
                    key={item.name}
                    to={item.href}
                    aria-current={item.current ? "page" : undefined}
                    className={classNames(
                      item.current
                        ? "bg-gray-900 text-white"
                        : "text-gray-300 hover:bg-gray-700 hover:text-white",
                      "rounded-md px-3 py-2 text-sm font-medium"
                    )}
                  >
                    {item.name}
                  </Link>
                ))}
                {user === null && (
                  <a
                    key="login"
                    href="/oidc/login"
                    className="text-gray-300 hover:bg-gray-700 hover:text-white rounded-md px-3 py-2 text-sm font-medium"
                  >
                    Sign in
                  </a>
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
