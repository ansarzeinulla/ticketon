"use client";

import { createContext, useContext, useMemo, type ReactNode } from "react";

import type { Locale } from "./config";
import { getDictionaryFor } from "./dictionaries";
import { translatorFor, type Translator } from "./translate";

/**
 * The locale, made available to Client Components (SRS §685).
 *
 * The server has already decided the locale for the request; this carries that
 * decision - just the string - down to the interactive pieces (the ticket
 * selector, the checkout dialog) so they translate to the same language as the
 * server-rendered page around them, with no second detection and no flash of
 * the wrong language.
 *
 * The dictionary itself is rebuilt on the client from the locale rather than
 * serialised across the boundary: it is static data that already ships in the
 * bundle, so sending it again would be waste.
 */
interface I18nValue {
  locale: Locale;
  t: Translator;
}

const I18nContext = createContext<I18nValue | null>(null);

export function I18nProvider({ locale, children }: { locale: Locale; children: ReactNode }) {
  const value = useMemo<I18nValue>(
    () => ({ locale, t: translatorFor(getDictionaryFor(locale)) }),
    [locale],
  );
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const value = useContext(I18nContext);
  if (!value) {
    throw new Error("useI18n must be used within an I18nProvider");
  }
  return value;
}

/** The common case: just the translator. */
export function useT(): Translator {
  return useI18n().t;
}
