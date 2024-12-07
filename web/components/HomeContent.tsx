import React from "react";
import CreateAthleteForm from "components/CreateAthleteForm";

const HomeContent: React.FC = () => {
  return (
    <div className="flex-1 p-6 bg-gray-100">
      <h1 className="text-3xl font-bold mb-4">Welcome to the Home Page</h1>
      <CreateAthleteForm />
    </div>
  );
};

export default HomeContent;
