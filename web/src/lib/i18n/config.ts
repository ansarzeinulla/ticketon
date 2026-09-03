/**
 * Internationalisation, customer-facing side (SRS 8, §685: "The customer-facing
 * interface shall initially support Kazakh and Russian, with English available
 * as an additional locale.").
 *
 * Kazakh is the default, not English: the platform operates in Kazakhstan, and
 * a Kazakh visitor arriving with no preference set should see Kazakh, not have
 * to go and find it. English exists but is the additional locale the SRS
 * describes, not the fallback the rest of the world's software assumes.
 */

export const locales = ["kk", "ru", "en"] as const;

export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = "kk";

/** The cookie the visitor's choice is remembered in. Readable by the server. */
export const LOCALE_COOKIE = "biletflow_locale";

/** Each locale's name in its own language, for the switcher. */
export const localeNames: Record<Locale, string> = {
  kk: "Қазақша",
  ru: "Русский",
  en: "English",
};

/** Narrow an arbitrary string to a supported locale, or null. */
export function asLocale(value: string | undefined | null): Locale | null {
  return value != null && (locales as readonly string[]).includes(value)
    ? (value as Locale)
    : null;
}
