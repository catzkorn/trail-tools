import { LogoMark, Wordmark } from "@components/header/logo";
import React from "react";
import { Link } from "react-router";

interface PublicFooterProps {
  // The author whose other reports to link to, if any.
  author?: { handle: string; displayName: string };
}

/**
 * PublicFooter ends a public page with the app's mark and a link to the
 * author's other reports.
 */
const PublicFooter: React.FC<PublicFooterProps> = ({ author }) => (
  <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-stone-200 pt-6 text-sm text-stone-500">
    <span className="flex items-center gap-2">
      <LogoMark summit="#111111" className="h-[17px] w-[26px]" />
      <span>
        Shared with{" "}
        <Link to="/" className="hover:underline">
          <Wordmark className="text-[12px] text-ink" />
        </Link>
        , a trail running planner.
      </span>
    </span>
    {author !== undefined && (
      <Link
        to={`/u/${author.handle}`}
        className="font-medium text-brand-600 hover:underline"
      >
        More reports by {author.displayName}
      </Link>
    )}
  </footer>
);

export default PublicFooter;
