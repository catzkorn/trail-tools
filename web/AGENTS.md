# Web app

A React single-page app in TypeScript, using Connect Query for the API,
Tailwind CSS v4 and Leaflet. The server embeds the bundle in `dist/` and serves
`index.html` for every page.

## Building

- `dist/` and `gen/` are committed, and CI fails if they're out of date. After
  changing the app, run `make esbuild tailwindcss` (or `make gen`) and commit
  the result. `make watch` rebuilds them while you work.
- `make web-format` formats with oxfmt, and `make web-lint` type checks and
  lints with oxlint. Both run in a Node container; `make web-deps` installs the
  packages.
- The type checker and linter are strict, so code has to be explicit:
  `x !== ""` and `x !== undefined` rather than truthiness, no `any` or non-null
  assertions, object keys in alphabetical order, `void` for promises you don't
  await, and exhaustive `switch` statements with braced cases.
- Optional props that may be `undefined` are typed `onX?: (() => void) |
  undefined`, because of `exactOptionalPropertyTypes`.

## Files

- `pages/<name>.tsx` has a page per route, default exporting `XxxPage`.
- `components/<feature>/` has the components of a feature, and `components/ui/`
  the shared ones.
- `helpers/` has plain functions, with named exports: dates, formatting,
  parsing, labels, queries and storage.
- File names are kebab-case. A file's main component is its default export;
  files of several small components (`*-bits.tsx`) use named exports.
- Import across folders with the aliases `@components/...`, `gen/...`,
  `helpers/...` and `pages/...`, and use `import type` for types.

## Components

- Write components as `const X: React.FC<XProps> = (...) => ...`, with an
  `interface XProps` above it.
- Doc comments start with the component's name, capitalized ("RacePlanCard
  shows ..."). Comment props above each field. Use `//` for one line and
  `/** */` for more; no trailing comments.
- Keep state in `useState`, one per form field, initialized from the existing
  value. There's no form library or global store.
- Use one dialog for creating and editing, taking the resource or `undefined`,
  and mount it only while it's open.

## Data

- Query with `useQuery(listRaces)` and the descriptors in
  `gen/<domain>/v1/*_connectquery`. Pass `skipToken` as the input to skip a
  query.
- Start independent queries together in the parent, not in children that only
  render once the parent's data has loaded.
- After a mutation, invalidate the affected services with
  `invalidateServices(queryClient, [RaceService, ...])` from
  `helpers/query.ts`, importing the service from `*_pb`. When a mutation
  returns what a query shows, write it into the cache with `setQueryData`
  instead.
- Send updates with an `updateMask` listing the changed fields in snake_case.
- Handle every state: `<Loading />` while pending, `<QueryError error={error}
  onRetry={...} />` on failure, an `EmptyState` when there's nothing, then the
  content. Show mutation errors with `<ErrorText error={...} />`, and disable
  buttons while their mutation is pending.
- Poll with `refetchInterval` only while something is changing, e.g. a sync in
  progress.

## Styling

- Use Tailwind classes, composed with template literals. Reuse the class
  strings in `components/ui/styles.ts` (`primaryButton`, `secondaryButton`,
  `inputClass`, `cardClass`, ...) and the components in `components/ui/`:
  `Card`, `PageHeader`, `EmptyState`, `Badge`, `Stat`, the fields in
  `fields.tsx`, `Modal` and `ConfirmButton` for destructive actions.
- Design for phones first, then larger screens with `sm:`, `md:` and `lg:`.
  Wrap wide tables in `overflow-x-auto`.
- There's no dark mode. Pages print cleanly: hide controls with
  `print:hidden`.
- Icons are Heroicons, outline unless filled reads better, sized with
  `size-4` to `size-6`. Numbers use `tabular-nums`.
- Give icon-only buttons an `aria-label` and a `title`. Buttons that don't
  submit a form have `type="button"`. External links open in a new tab with
  `rel="noreferrer"`.

## Formatting and wording

- Format with the helpers in `helpers/format.ts` (`formatKm`, `formatMeters`,
  `formatDuration`, `formatPace`, `formatDate`, ...) and parse input with
  `helpers/parse.ts`. Units are metric only.
- Calendar dates are `YYYY-MM-DD` strings in local time, handled by
  `helpers/dates.ts`. Never use `toISOString` for them.
- Store preferences with `helpers/storage.ts`, under keys starting with
  `trail-tools.`.
- Write text in sentence case and American spelling, for trail runners:
  "New plan", "Delete race plan?". Subtitles, hints and validation messages
  are full sentences. Pending labels end with an ellipsis character:
  "Saving…". Separate items with " · ".
- The app is called "trailrunn.ing", in lowercase.

## Maps

Maps use Leaflet directly, not react-leaflet. `MapView` creates the map and
`useMap()` gives it to children. Layers are components that render nothing and
add and remove Leaflet objects in an effect with cleanup, like those in
`components/map/layers.tsx`. Wrap tooltip text in `textElement()` so it isn't
parsed as HTML. Charts are SVG, drawn by hand.

## Routes

Routes are declared in `app.tsx`: public pages for everyone, and the rest only
when signed in. Collections are plural (`/races`, `/races/:id`), with `/new`
for creating. Keep view state like filters in the URL's search parameters. Add
new pages to the header's navigation in `components/header/`.
