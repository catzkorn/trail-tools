const MAX_SLUG_LENGTH = 80;

/**
 * Slugify turns a title into the slug the server would choose: lowercase
 * ASCII letters and digits separated by single dashes, without accents,
 * e.g. "Säntis via the Lisengrat!" becomes "santis-via-the-lisengrat".
 */
export const slugify = (title: string): string => {
  let slug = title
    .toLowerCase()
    .normalize("NFKD")
    .replaceAll(/\p{Mn}/gu, "")
    .replaceAll("ß", "ss")
    .replaceAll(/[^a-z0-9]+/gu, "-")
    .replaceAll(/^-+|-+$/gu, "");
  if (slug.length > MAX_SLUG_LENGTH) {
    slug = slug.slice(0, MAX_SLUG_LENGTH);
    const dash = slug.lastIndexOf("-");
    if (dash > MAX_SLUG_LENGTH / 2) {
      slug = slug.slice(0, dash);
    }
    slug = slug.replaceAll(/-+$/gu, "");
  }
  return slug;
};

const SLUG_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/u;

export const isValidSlug = (slug: string): boolean =>
  slug.length <= MAX_SLUG_LENGTH && SLUG_PATTERN.test(slug);

const HANDLE_PATTERN = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/u;

export const isValidHandle = (handle: string): boolean =>
  handle.length >= 3 && handle.length <= 30 && HANDLE_PATTERN.test(handle);
