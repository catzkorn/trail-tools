import Header from "@components/Header";
import HomeContent from "@components/HomeContent";
import { User } from "gen/users/v1/users_pb";
import React from "react";

interface HomePageProps {
  user: User | null;
}

const HomePage: React.FC<HomePageProps> = ({ user }) => {
  return (
    <div className="min-h-full">
      <Header user={user} />
      <main>
        <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
          <HomeContent />
        </div>
      </main>
    </div>
  );
};

export default HomePage;
