import { describe, expect, it } from "vitest";

import {
  formatInTimezone,
  isoToLocalInput,
  localInputToISO,
  slugify,
} from "./datetime";

/**
 * Times are read and written in the event's own timezone, never the browser's
 * (SRS 7: "Calendar exports shall preserve the event's configured time zone";
 * the ticket prints the local time an attendee will turn up at).
 *
 * The cases that matter are the ones a walkthrough on one machine in one zone
 * cannot catch: a DST boundary, and the round trip that the edit form depends
 * on to show an organizer the time they typed.
 */

describe("localInputToISO", () => {
  it("reads wall-clock time in the event's zone, not the runner's", () => {
    // Almaty is UTC+5 with no DST, so 19:00 local is 14:00Z whatever the
    // machine running this is set to.
    expect(localInputToISO("2027-04-11T19:00", "Asia/Almaty")).toBe(
      "2027-04-11T14:00:00.000Z",
    );
  });

  it("handles a zone that does observe DST, on both sides of the change", () => {
    // London is UTC+0 in January and UTC+1 in July.
    expect(localInputToISO("2027-01-15T12:00", "Europe/London")).toBe(
      "2027-01-15T12:00:00.000Z",
    );
    expect(localInputToISO("2027-07-15T12:00", "Europe/London")).toBe(
      "2027-07-15T11:00:00.000Z",
    );
  });

  it("returns null for an unusable value instead of an invalid date", () => {
    for (const bad of ["", "not a date", "2027-04-11", "11/04/2027 19:00"]) {
      expect(localInputToISO(bad, "Asia/Almaty")).toBeNull();
    }
  });
});

describe("isoToLocalInput", () => {
  it("renders an instant as wall-clock time at the venue", () => {
    expect(isoToLocalInput("2027-04-11T14:00:00Z", "Asia/Almaty")).toBe(
      "2027-04-11T19:00",
    );
  });

  it("round-trips with localInputToISO, which the edit form relies on", () => {
    // If these disagreed, opening the edit form and saving without touching
    // anything would silently move the event.
    for (const [local, zone] of [
      ["2027-04-11T19:30", "Asia/Almaty"],
      ["2027-01-15T12:00", "Europe/London"],
      ["2027-07-15T23:45", "Europe/London"],
      ["2027-12-31T00:00", "Asia/Almaty"],
    ] as const) {
      const iso = localInputToISO(local, zone);
      expect(iso).not.toBeNull();
      expect(isoToLocalInput(iso as string, zone)).toBe(local);
    }
  });

  it("renders midnight as 00, not 24", () => {
    const iso = localInputToISO("2027-06-01T00:00", "Asia/Almaty");
    expect(isoToLocalInput(iso as string, "Asia/Almaty")).toBe("2027-06-01T00:00");
  });

  it("returns an empty string for an unparseable instant", () => {
    expect(isoToLocalInput("nonsense", "Asia/Almaty")).toBe("");
  });
});

describe("formatInTimezone", () => {
  it("shows the time at the venue", () => {
    const rendered = formatInTimezone("2027-04-11T14:00:00Z", "Asia/Almaty");
    expect(rendered).toContain("19:00");
    expect(rendered).toContain("2027");
  });

  it("falls back to the raw value rather than throwing on a bad zone", () => {
    expect(() => formatInTimezone("2027-04-11T14:00:00Z", "Not/AZone")).not.toThrow();
  });
});

describe("slugify", () => {
  it("makes a URL-safe slug from a title", () => {
    expect(slugify("Almaty Winter Jazz Night")).toBe("almaty-winter-jazz-night");
  });

  it("transliterates Cyrillic, so a Kazakh or Russian title still gets a URL", () => {
    // SRS 7 expects Kazakh and Russian content; a title in either must not
    // produce an empty slug.
    expect(slugify("Алматы Джаз")).not.toBe("");
    expect(slugify("Алматы Джаз")).toMatch(/^[a-z0-9-]+$/);
  });

  it("collapses punctuation and trims stray separators", () => {
    expect(slugify("  Jazz -- Night!!  ")).toBe("jazz-night");
  });
});
