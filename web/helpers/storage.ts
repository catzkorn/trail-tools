/**
 * Wrappers around localStorage, which can throw in private browsing modes or
 * when storage is disabled.
 */

export const readStorage = (key: string): string | null => {
  try {
    return globalThis.localStorage.getItem(key);
  } catch {
    return null;
  }
};

export const writeStorage = (key: string, value: string): void => {
  try {
    globalThis.localStorage.setItem(key, value);
  } catch {
    // Storage is unavailable; the value is simply not remembered.
  }
};

export const readStorageFlag = (key: string): boolean =>
  readStorage(key) === "true";

export const writeStorageFlag = (key: string, value: boolean): void => {
  writeStorage(key, value ? "true" : "false");
};
