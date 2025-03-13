import UserNav from "@components/header/UserNav";
import { Disclosure } from "@headlessui/react";
import { User } from "gen/users/v1/users_pb";
import React from "react";
import { Link } from "react-router-dom";

interface HeaderProps {
  user: User | null;
}

const Header: React.FC<HeaderProps> = ({ user }) => {
  const buttonStyle =
    "text-gray-300 hover:bg-gray-700 hover:text-white rounded-md px-3 py-2 text-md font-medium";

  return (
    <Disclosure as="nav" className="bg-gray-800">
      <div className="mx-auto max-w-5xl px-4">
        <div className="flex items-center justify-between pt-3">
          <div className="flex flex-row justify-between">
            <Link to="/" className="flex flex-col items-center">
              <img
                className="h-8 w-auto"
                src="/img/logo_transparent_small.png"
                alt="Trail Tools"
              />
              <span className="ml-2 text-white text-xl font-bold">
                Trail Tools
              </span>
            </Link>
          </div>
          {user !== null && <UserNav user={user} />}
        </div>
        <div className="flex flex-row p-2">
          {user !== null && (
            <>
              <Link to="/athletes">
                <button className={buttonStyle}>Athletes</button>
              </Link>
              <Link to="/blood-lactate">
                <button className={buttonStyle}>Blood Lactate</button>
              </Link>
              <Link to="/settings">
                <button className={buttonStyle}>Settings</button>
              </Link>
            </>
          )}
        </div>
      </div>
    </Disclosure>
  );
};

export default Header;
