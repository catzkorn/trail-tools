import React from "react";

const Sidebar: React.FC = () => {
  return (
    <div className="bg-gray-800 text-white h-screen w-64 p-4">
      <h2 className="text-2xl font-bold mb-6">My App</h2>
      <nav>
        <ul className="space-y-4">
          <li>
            <a href="#" className="hover:text-gray-400">
              Dashboard
            </a>
          </li>
          <li>
            <a href="#" className="hover:text-gray-400">
              Profile
            </a>
          </li>
          <li>
            <a href="#" className="hover:text-gray-400">
              Settings
            </a>
          </li>
          <li>
            <a href="#" className="hover:text-gray-400">
              Logout
            </a>
          </li>
        </ul>
      </nav>
    </div>
  );
};

export default Sidebar;
