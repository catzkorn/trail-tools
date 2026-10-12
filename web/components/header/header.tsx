import UserNav from "@components/header/user-nav";
import type { User } from "gen/users/v1/users_pb";
import React from "react";
import { Link, NavLink } from "react-router";

interface HeaderProps {
  user: User | null;
}

interface NavItem {
  to: string;
  label: string;
  end: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { end: true, label: "Dashboard", to: "/" },
  { end: false, label: "Calendar", to: "/calendar" },
  { end: false, label: "Training plans", to: "/plans" },
  { end: false, label: "Routes", to: "/routes" },
  { end: false, label: "Races", to: "/races" },
  { end: false, label: "Weather", to: "/forecast" },
  { end: false, label: "Activities", to: "/activities" },
  { end: false, label: "Trip reports", to: "/trip-reports" },
  { end: false, label: "Settings", to: "/settings" },
];

const navClass = ({ isActive }: { isActive: boolean }): string =>
  `rounded-md px-3 py-1.5 text-sm font-medium whitespace-nowrap ${
    isActive
      ? "bg-gray-900 text-white"
      : "text-gray-300 hover:bg-gray-700 hover:text-white"
  }`;

const Header: React.FC<HeaderProps> = ({ user }) => (
  <nav className="bg-gray-800 print:hidden">
    <div className="mx-auto max-w-7xl px-4">
      <div className="flex h-14 items-center justify-between gap-4">
        <Link to="/" className="flex shrink-0 items-center gap-2">
          <img
            className="h-8 w-auto"
            src="/img/logo_transparent_small.png"
            alt=""
          />
          <span className="text-lg font-bold text-white">Trail Tools</span>
        </Link>
        {user !== null && (
          <div className="hidden grow items-center gap-1 lg:flex">
            {NAV_ITEMS.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={navClass}
              >
                {item.label}
              </NavLink>
            ))}
          </div>
        )}
        {user !== null && <UserNav user={user} />}
      </div>
      {user !== null && (
        <div className="-mx-1 flex gap-1 overflow-x-auto pb-2 lg:hidden">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={navClass}
            >
              {item.label}
            </NavLink>
          ))}
        </div>
      )}
    </div>
  </nav>
);

export default Header;
