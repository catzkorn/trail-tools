import React from "react";
import Sidebar from "components/Sidebar";
import HomeContent from "components/HomeContent";

const HomePage: React.FC = () => {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <HomeContent />
    </div>
  );
};

export default HomePage;
