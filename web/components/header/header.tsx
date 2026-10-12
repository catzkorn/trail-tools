import UserNav from "@components/header/user-nav";
import {
  CloseButton,
  Menu,
  MenuButton,
  MenuItem,
  MenuItems,
  Popover,
  PopoverButton,
  PopoverPanel,
} from "@headlessui/react";
import {
  Bars3Icon,
  ChevronDownIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import type { User } from "gen/users/v1/users_pb";
import React from "react";
import { Link, NavLink, matchPath, useLocation } from "react-router";

interface HeaderProps {
  user: User | null;
}

interface NavItem {
  to: string;
  label: string;
  // A short description, shown in menus.
  description?: string;
  end?: boolean;
}

interface NavGroup {
  label: string;
  items: NavItem[];
}

// The pages linked directly from the header.
const TOP_ITEMS: NavItem[] = [
  { end: true, label: "Dashboard", to: "/" },
  { label: "Calendar", to: "/calendar" },
  { label: "Activities", to: "/activities" },
];

// The pages grouped in menus.
const GROUPS: NavGroup[] = [
  {
    items: [
      {
        description: "Build up to your races week by week",
        label: "Training plans",
        to: "/plans",
      },
      {
        description: "Draw routes and import courses",
        label: "Routes",
        to: "/routes",
      },
      {
        description: "Races you could run and race day plans",
        label: "Races",
        to: "/races",
      },
      {
        description: "Forecasts along your routes",
        label: "Weather",
        to: "/forecast",
      },
    ],
    label: "Planning",
  },
  {
    items: [
      {
        description: "Stories of your trips, with photos",
        label: "Trip reports",
        to: "/trip-reports",
      },
      {
        description: "How your races went against the plan",
        label: "Race reports",
        to: "/race-reports",
      },
    ],
    label: "Reports",
  },
];

const linkClass = (active: boolean): string =>
  `inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-sm font-medium whitespace-nowrap ${
    active
      ? "bg-gray-900 text-white"
      : "text-gray-300 hover:bg-gray-700 hover:text-white"
  }`;

// UseIsActive returns whether a nav item's page is shown.
const useIsActive = (): ((item: NavItem) => boolean) => {
  const { pathname } = useLocation();
  return (item) =>
    matchPath({ end: item.end ?? false, path: item.to }, pathname) !== null;
};

// GroupMenu is a dropdown menu of a group of pages.
const GroupMenu: React.FC<{ group: NavGroup }> = ({ group }) => {
  const isActive = useIsActive();
  const active = group.items.some(isActive);
  return (
    <Menu as="div" className="relative">
      <MenuButton className={linkClass(active)}>
        {group.label}
        <ChevronDownIcon className="size-4" />
      </MenuButton>
      <MenuItems
        transition
        anchor="bottom start"
        className="z-[1100] mt-1 w-72 rounded-md bg-white p-1 shadow-lg ring-1 ring-black/5 transition focus:outline-hidden data-[closed]:scale-95 data-[closed]:opacity-0 data-[enter]:duration-100 data-[leave]:duration-75"
      >
        {group.items.map((item) => (
          <MenuItem key={item.to}>
            <Link
              to={item.to}
              className={`block rounded px-3 py-2 data-[focus]:bg-gray-100 ${isActive(item) ? "bg-gray-50" : ""}`}
            >
              <span className="block text-sm font-medium text-gray-900">
                {item.label}
              </span>
              {item.description !== undefined && (
                <span className="block text-xs text-gray-500">
                  {item.description}
                </span>
              )}
            </Link>
          </MenuItem>
        ))}
      </MenuItems>
    </Menu>
  );
};

// MobileMenu is a menu of all pages for small screens.
const MobileMenu: React.FC = () => {
  const isActive = useIsActive();
  const itemClass = (item: NavItem): string =>
    `block rounded-md px-3 py-2 text-sm font-medium ${
      isActive(item)
        ? "bg-gray-900 text-white"
        : "text-gray-200 hover:bg-gray-700 hover:text-white"
    }`;
  return (
    <Popover className="md:hidden">
      <PopoverButton className="rounded-md p-1.5 text-gray-300 hover:bg-gray-700 hover:text-white focus:outline-hidden focus-visible:ring-2 focus-visible:ring-white">
        {({ open }) => (
          <>
            <span className="sr-only">{open ? "Close menu" : "Open menu"}</span>
            {open ? (
              <XMarkIcon className="size-6" />
            ) : (
              <Bars3Icon className="size-6" />
            )}
          </>
        )}
      </PopoverButton>
      <PopoverPanel className="absolute inset-x-0 top-14 z-[1100] border-t border-gray-700 bg-gray-800 px-4 pt-2 pb-4 shadow-lg">
        <div className="flex flex-col gap-1">
          {TOP_ITEMS.map((item) => (
            <CloseButton
              key={item.to}
              as={Link}
              to={item.to}
              className={itemClass(item)}
            >
              {item.label}
            </CloseButton>
          ))}
        </div>
        {GROUPS.map((group) => (
          <div key={group.label} className="mt-3">
            <p className="px-3 pb-1 text-xs font-semibold tracking-wide text-gray-400 uppercase">
              {group.label}
            </p>
            <div className="flex flex-col gap-1">
              {group.items.map((item) => (
                <CloseButton
                  key={item.to}
                  as={Link}
                  to={item.to}
                  className={itemClass(item)}
                >
                  {item.label}
                </CloseButton>
              ))}
            </div>
          </div>
        ))}
      </PopoverPanel>
    </Popover>
  );
};

const Header: React.FC<HeaderProps> = ({ user }) => (
  <nav className="relative bg-gray-800 print:hidden">
    <div className="mx-auto max-w-7xl px-4">
      <div className="flex h-14 items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          {user !== null && <MobileMenu />}
          <Link to="/" className="flex shrink-0 items-center gap-2">
            <img
              className="h-8 w-auto"
              src="/img/logo_transparent_small.png"
              alt=""
            />
            <span className="text-lg font-bold text-white">Trail Tools</span>
          </Link>
        </div>
        {user !== null && (
          <div className="hidden grow items-center gap-1 md:flex">
            {TOP_ITEMS.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end ?? false}
                className={({ isActive }) => linkClass(isActive)}
              >
                {item.label}
              </NavLink>
            ))}
            {GROUPS.map((group) => (
              <GroupMenu key={group.label} group={group} />
            ))}
          </div>
        )}
        {user !== null && <UserNav user={user} />}
      </div>
    </div>
  </nav>
);

export default Header;
