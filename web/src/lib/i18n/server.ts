import { cookies, headers } from "next/headers";

import { asLocale, defaultLocale, LOCALE_COOKIE, locales, type Locale } from "./config";
import { getDictionaryFor } from "./dictionaries";
import { translatorFor, type Translator } from "./translate";

/**
 * The locale for this request, decided on the server (SRS §685).
 *
 * A saved choice wins - it is the visitor telling us in as many words. Failing
 * that we read Accept-Language, so a first visit from a Russian-configured
 * browser is Russian without anyone touching the switcher. Only when neither
 * says anything do we fall to Kazakh, because the platform is Kazakhstani and
 * that is the right default here rather than English.
 */
export async function getLocale(): Promise<Locale> {
  const cookieStore = await cookies();
  const saved = asLocale(cookieStore.get(LOCALE_COOKIE)?.value);
  if (saved) return saved;

  const header = (await headers()).get("accept-language");
  return negotiate(header);
}

/** Pick the best supported locale from an Accept-Language header. */
export function negotiate(header: string | null): Locale {
  if (!header) return defaultLocale;

  const ranked = header
    .split(",")
    .map((part) => {
      const [tag, q] = part.trim().split(";q=");
      return { tag: tag.toLowerCase().split("-")[0], weight: q ? Number(q) : 1 };
    })
    .sort((a, b) => b.weight - a.weight);

  for (const { tag } of ranked) {
    const match = asLocale(tag);
    if (match && (locales as readonly string[]).includes(match)) return match;
  }
  return defaultLocale;
}

/** The dictionary and a bound translator for a server component. */
export async function getTranslations(): Promise<{ locale: Locale; t: Translator }> {
  const locale = await getLocale();
  return { locale, t: translatorFor(getDictionaryFor(locale)) };
}
