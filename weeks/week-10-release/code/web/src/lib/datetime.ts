/**
 * Converting between the `datetime-local` inputs on the form and the RFC 3339
 * timestamps the Go API expects.
 *
 * An organizer types the wall-clock time at the venue ("19:00"), and the event
 * carries its own IANA timezone, so the two have to be combined into a real
 * instant. Doing this with `new Date("2026-12-20T19:00")` would silently use
 * the *browser's* zone, which is wrong for anyone planning an Almaty event
 * from another country.
 */

/** IANA zones offered in the form, Kazakhstan first. */
export const COMMON_TIMEZONES = [
  "Asia/Almaty",
  "Asia/Aqtau",
  "Asia/Aqtobe",
  "Asia/Atyrau",
  "Asia/Oral",
  "Asia/Qostanay",
  "Asia/Qyzylorda",
  "Asia/Tashkent",
  "Asia/Bishkek",
  "Europe/Moscow",
  "Europe/London",
  "Europe/Berlin",
  "America/New_York",
  "UTC",
] as const;

/**
 * Every zone the browser knows, with the common ones pinned to the top.
 * Falls back to the short list on engines without `supportedValuesOf`.
 */
export function timezoneOptions(): string[] {
  const common = [...COMMON_TIMEZONES];

  let all: string[] = [];
  try {
    const supported = (
      Intl as typeof Intl & {
        supportedValuesOf?: (key: string) => string[];
      }
    ).supportedValuesOf;
    if (typeof supported === "function") all = supported("timeZone");
  } catch {
    all = [];
  }

  if (all.length === 0) return common;
  return [...common, ...all.filter((zone) => !common.includes(zone as never))];
}

/** The browser's own zone, when it is one we can offer as a default. */
export function browserTimezone(): string | null {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || null;
  } catch {
    return null;
  }
}

/**
 * How far `timeZone` is ahead of UTC at the given instant, in milliseconds.
 * Derived from Intl rather than a table, so DST is handled by the platform.
 */
function zoneOffsetMs(instant: Date, timeZone: string): number {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    hour12: false,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).formatToParts(instant);

  const read = (type: Intl.DateTimeFormatPartTypes): number => {
    const part = parts.find((p) => p.type === type);
    return part ? Number(part.value) : 0;
  };

  // Some engines render midnight as hour 24; normalise it.
  const hour = read("hour") % 24;

  const asIfUTC = Date.UTC(
    read("year"),
    read("month") - 1,
    read("day"),
    hour,
    read("minute"),
    read("second"),
  );

  return asIfUTC - instant.getTime();
}

/**
 * Turn a `datetime-local` value ("2026-12-20T19:00") plus an IANA zone into an
 * RFC 3339 UTC timestamp.
 *
 * Returns null when the input is empty or unparseable, so callers can show a
 * validation message instead of sending a bad request.
 */
export function localInputToISO(value: string, timeZone: string): string | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/.exec(value);
  if (!match) return null;

  const [, year, month, day, hour, minute, second = "0"] = match;
  const naiveUTC = Date.UTC(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
    Number(second),
  );

  // The offset depends on the instant we are solving for, so guess once using
  // the naive value, then refine. The second pass settles DST boundaries.
  let instant = naiveUTC - zoneOffsetMs(new Date(naiveUTC), timeZone);
  instant = naiveUTC - zoneOffsetMs(new Date(instant), timeZone);

  const result = new Date(instant);
  return Number.isNaN(result.getTime()) ? null : result.toISOString();
}

/**
 * The inverse of localInputToISO: an RFC 3339 instant rendered as the
 * wall-clock `datetime-local` value at the event's venue.
 *
 * The edit form needs this so an organizer sees the time they typed, not the
 * same instant translated into whatever zone their laptop is in. Formatting in
 * en-CA gives ISO-ordered date parts, which is what the input expects.
 */
export function isoToLocalInput(iso: string, timeZone: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";

  try {
    const parts = new Intl.DateTimeFormat("en-CA", {
      timeZone,
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    }).formatToParts(date);

    const get = (type: string) => parts.find((part) => part.type === type)?.value ?? "";
    // Intl renders midnight as "24" in some environments; the input wants "00".
    const hour = get("hour") === "24" ? "00" : get("hour");
    return `${get("year")}-${get("month")}-${get("day")}T${hour}:${get("minute")}`;
  } catch {
    return date.toISOString().slice(0, 16);
  }
}

/** Render an API timestamp in the event's own timezone. */
export function formatInTimezone(iso: string, timeZone: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;

  try {
    return new Intl.DateTimeFormat("en-GB", {
      timeZone,
      dateStyle: "medium",
      timeStyle: "short",
    }).format(date);
  } catch {
    return date.toISOString();
  }
}

/** A `datetime-local` value for `daysFromNow`, rounded to the hour. */
export function defaultLocalInput(daysFromNow: number, atHour: number): string {
  const date = new Date();
  date.setDate(date.getDate() + daysFromNow);
  date.setHours(atHour, 0, 0, 0);

  const pad = (n: number) => String(n).padStart(2, "0");
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}`
  );
}

/** "Almaty Winter Jazz" -> "almaty-winter-jazz", for the slug preview. */
/**
 * Cyrillic to ASCII, mirroring internal/store/slug.go.
 *
 * Kazakh and Russian titles are the common case in Kazakhstan (SRS 7), and
 * without this every one of them slugified to an empty string - so the form
 * previewed "/your-event" while the API was about to mint a real transliterated
 * slug. The preview has to agree with what the server will do.
 */
const CYRILLIC: Record<string, string> = {
  а: "a", ә: "a", б: "b", в: "v", г: "g", ғ: "g", д: "d",
  е: "e", ё: "e", ж: "zh", з: "z", и: "i", й: "i", к: "k",
  қ: "q", л: "l", м: "m", н: "n", ң: "ng", о: "o", ө: "o",
  п: "p", р: "r", с: "s", т: "t", у: "u", ұ: "u", ү: "u",
  ф: "f", х: "h", һ: "h", ц: "ts", ч: "ch", ш: "sh", щ: "sch",
  ъ: "", ы: "y", і: "i", ь: "", э: "e", ю: "yu", я: "ya",
};

export function slugify(input: string): string {
  const transliterated = Array.from(input.toLowerCase())
    .map((character) => CYRILLIC[character] ?? character)
    .join("");

  return transliterated
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80)
    .replace(/-+$/g, "");
}
