import { TripTag } from "gen/trips/v1/trips_pb";
import React from "react";

interface TagGroup {
  label: string;
  // Whether a report has at most one of the group's tags.
  exclusive: boolean;
  tags: TripTag[];
}

// TAG_GROUPS are the tags a report can have, grouped by what they describe.
export const TAG_GROUPS: TagGroup[] = [
  {
    exclusive: true,
    label: "Difficulty",
    tags: [TripTag.BEGINNER_FRIENDLY, TripTag.MODERATE, TripTag.CHALLENGING],
  },
  {
    exclusive: false,
    label: "Terrain",
    tags: [
      TripTag.RUNNABLE,
      TripTag.TECHNICAL,
      TripTag.SCRAMBLING,
      TripTag.SNOW,
    ],
  },
  {
    exclusive: true,
    label: "Route",
    tags: [TripTag.LOOP, TripTag.OUT_AND_BACK, TripTag.POINT_TO_POINT],
  },
];

// ALL_TAGS are the tags in the order of their groups.
export const ALL_TAGS = TAG_GROUPS.flatMap((group) => group.tags);

const TAG_LABELS: Record<TripTag, string> = {
  [TripTag.UNSPECIFIED]: "",
  [TripTag.BEGINNER_FRIENDLY]: "Good for beginners",
  [TripTag.MODERATE]: "Moderate",
  [TripTag.CHALLENGING]: "Challenging",
  [TripTag.RUNNABLE]: "Runnable",
  [TripTag.TECHNICAL]: "Technical",
  [TripTag.SCRAMBLING]: "Scrambling",
  [TripTag.SNOW]: "Snow",
  [TripTag.LOOP]: "Loop",
  [TripTag.OUT_AND_BACK]: "Out and back",
  [TripTag.POINT_TO_POINT]: "Point to point",
};

export const tagLabel = (tag: TripTag): string => TAG_LABELS[tag];

// TAG_SLUGS name tags in URLs, e.g. ?tag=beginner-friendly.
const TAG_SLUGS = new Map(
  TAG_GROUPS.flatMap((group) =>
    group.tags.map((tag): [TripTag, string] => [
      tag,
      TripTag[tag].toLowerCase().replaceAll("_", "-"),
    ])
  )
);

export const tagSlug = (tag: TripTag): string => TAG_SLUGS.get(tag) ?? "";

// TagFromSlug returns the tag named in a URL, if any.
export const tagFromSlug = (slug: string | null): TripTag | undefined =>
  [...TAG_SLUGS].find(([, value]) => value === slug)?.[0];

const tagClass =
  "inline-flex items-center rounded-full bg-stone-100 px-2.5 py-0.5 text-xs font-medium text-stone-700 ring-1 ring-stone-200 ring-inset";

// TagList shows a report's tags.
export const TagList: React.FC<{ tags: TripTag[]; className?: string }> = ({
  tags,
  className = "",
}) =>
  tags.length > 0 && (
    <ul className={`flex flex-wrap gap-1.5 ${className}`}>
      {tags.map((tag) => (
        <li key={tag} className={tagClass}>
          {tagLabel(tag)}
        </li>
      ))}
    </ul>
  );

interface TagPickerProps {
  tags: TripTag[];
  onChange: (tags: TripTag[]) => void;
}

// TagPicker chooses a report's tags, at most one of each exclusive group.
export const TagPicker: React.FC<TagPickerProps> = ({ tags, onChange }) => {
  const toggle = (group: TagGroup, tag: TripTag) => {
    if (tags.includes(tag)) {
      onChange(tags.filter((other) => other !== tag));
      return;
    }
    const kept = group.exclusive
      ? tags.filter((other) => !group.tags.includes(other))
      : tags;
    const next = new Set([...kept, tag]);
    onChange(ALL_TAGS.filter((other) => next.has(other)));
  };
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-1 text-sm font-medium text-gray-700">Tags</legend>
      {TAG_GROUPS.map((group) => (
        <div
          key={group.label}
          className="flex flex-wrap items-center gap-1.5"
          role="group"
          aria-label={group.label}
        >
          <span className="w-20 text-xs text-gray-500">{group.label}</span>
          {group.tags.map((tag) => {
            const selected = tags.includes(tag);
            return (
              <button
                key={tag}
                type="button"
                aria-pressed={selected}
                className={`rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset focus:outline-hidden focus-visible:ring-2 focus-visible:ring-brand-500 ${
                  selected
                    ? "bg-ink text-white ring-ink"
                    : "bg-white text-stone-700 ring-stone-300 hover:bg-stone-50"
                }`}
                onClick={() => {
                  toggle(group, tag);
                }}
              >
                {tagLabel(tag)}
              </button>
            );
          })}
        </div>
      ))}
    </fieldset>
  );
};

// FilterPill is the class of a filter button.
const filterPill = (active: boolean): string =>
  `rounded-full px-3 py-1 text-sm font-medium ring-1 ring-inset focus:outline-hidden focus-visible:ring-2 focus-visible:ring-brand-500 ${
    active
      ? "bg-ink text-white ring-ink"
      : "bg-white text-stone-700 ring-stone-300 hover:bg-stone-50"
  }`;

interface TagFilterProps {
  // The tags to filter by.
  tags: TripTag[];
  selected: TripTag | undefined;
  onSelect: (tag: TripTag | undefined) => void;
}

// TagFilter filters a list of reports by one of their tags.
export const TagFilter: React.FC<TagFilterProps> = ({
  tags,
  selected,
  onSelect,
}) => (
  <div
    className="flex flex-wrap justify-center gap-2"
    role="group"
    aria-label="Filter by tag"
  >
    <button
      type="button"
      aria-pressed={selected === undefined}
      className={filterPill(selected === undefined)}
      onClick={() => {
        onSelect(undefined);
      }}
    >
      All
    </button>
    {tags.map((tag) => (
      <button
        key={tag}
        type="button"
        aria-pressed={tag === selected}
        className={filterPill(tag === selected)}
        onClick={() => {
          onSelect(tag === selected ? undefined : tag);
        }}
      >
        {tagLabel(tag)}
      </button>
    ))}
  </div>
);
