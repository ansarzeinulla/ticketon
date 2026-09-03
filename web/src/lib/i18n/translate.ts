import type { Dictionary, MessageKey } from "./dictionaries";

/**
 * Resolve a dotted message key against a dictionary and fill `{name}`
 * placeholders.
 *
 * A missing key returns the key itself rather than an empty string: a visible
 * "checkout.title" on the page is a bug that gets fixed, whereas a blank space
 * is one that ships. In practice the typed `MessageKey` stops it happening at
 * all.
 */
export function translate(
  dict: Dictionary,
  key: MessageKey,
  params?: Record<string, string | number>,
): string {
  const raw = key.split(".").reduce<unknown>((node, part) => {
    if (node && typeof node === "object" && part in node) {
      return (node as Record<string, unknown>)[part];
    }
    return undefined;
  }, dict);

  if (typeof raw !== "string") return key;
  if (!params) return raw;

  return raw.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  );
}

/** A bound translator: `t("checkout.title")`. */
export type Translator = (key: MessageKey, params?: Record<string, string | number>) => string;

export function translatorFor(dict: Dictionary): Translator {
  return (key, params) => translate(dict, key, params);
}
