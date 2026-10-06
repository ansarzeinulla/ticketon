import { describe, expect, it } from "vitest";

import { formatKZT, formatTiyn, lineTotal, toTiyn } from "./money";

/**
 * Money never becomes a float (SRS 7: prices and refunds are KZT; the schema
 * stores numeric(14,2)).
 *
 * These are the inputs a browser walkthrough would never think to type: a
 * one-digit fraction, a value large enough to lose precision if it went
 * through a float, and the malformed strings a misbehaving API could return.
 */

describe("toTiyn", () => {
  it("parses whole and fractional amounts", () => {
    expect(toTiyn("5000.00")).toBe(500_000);
    expect(toTiyn("0.00")).toBe(0);
    expect(toTiyn("0")).toBe(0);
    expect(toTiyn("12345")).toBe(1_234_500);
  });

  it("pads a single decimal place rather than reading it as tiyn", () => {
    // "5000.5" is five thousand and fifty tiyn, not five thousand and five.
    expect(toTiyn("5000.5")).toBe(500_050);
    expect(toTiyn("0.1")).toBe(10);
  });

  it("survives amounts a float would round", () => {
    // 99 999 999 999.99 KZT is within numeric(14,2) and beyond the range where
    // float arithmetic on tiyn stays exact only because we use integers.
    expect(toTiyn("99999999999.99")).toBe(9_999_999_999_999);
  });

  it("ignores surrounding whitespace", () => {
    expect(toTiyn("  5000.00  ")).toBe(500_000);
  });

  it("returns zero for anything it cannot parse, rather than NaN", () => {
    // NaN would propagate silently into a total; zero is visibly wrong.
    for (const bad of ["", "abc", "-100.00", "1.234", "1,000", "1e3", "٥٠٠"]) {
      expect(toTiyn(bad)).toBe(0);
    }
  });
});

describe("formatTiyn", () => {
  // Intl groups KZT with a narrow no-break space (U+202F), not an ordinary
  // one. Spelling it out keeps the expectation honest - a plain space here
  // would pass only by accident of how the file was saved.
  const NNBSP = "\u202f";

  it("drops the decimals on a round amount", () => {
    expect(formatTiyn(500_000)).toBe(`₸5${NNBSP}000`);
    expect(formatTiyn(0)).toBe("₸0");
  });

  it("keeps them when there are tiyn to show", () => {
    expect(formatTiyn(500_050)).toBe(`₸5${NNBSP}000.50`);
  });

  it("groups with spaces rather than commas", () => {
    expect(formatTiyn(123_456_700)).not.toContain(",");
    expect(formatTiyn(123_456_700)).toContain(NNBSP);
  });
});

describe("formatKZT", () => {
  it("renders what the API sends", () => {
    expect(formatKZT("5000.00")).toBe("₸5\u202f000");
    expect(formatKZT("0.00")).toBe("₸0");
    expect(formatKZT("21120.00")).toBe("₸21\u202f120");
  });
});

describe("lineTotal", () => {
  it("multiplies in integer tiyn, so four lots of a half do not drift", () => {
    expect(lineTotal("0.25", 4)).toBe("₸1");
    expect(lineTotal("5000.00", 3)).toBe("₸15\u202f000");
  });

  it("is zero for a free ticket at any quantity", () => {
    expect(lineTotal("0.00", 10)).toBe("₸0");
  });

  it("is zero for an empty basket", () => {
    expect(lineTotal("5000.00", 0)).toBe("₸0");
  });
});
