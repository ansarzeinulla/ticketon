/**
 * Money helpers.
 *
 * Amounts cross the wire as decimal strings ("5000.00") so PostgreSQL's
 * numeric(14,2) is never rounded by a JavaScript float. These helpers format
 * for display and do the small amount of arithmetic the ticket selector needs,
 * in integer tiyn (1/100 KZT) rather than floats.
 */

/** Parse "5000.00" into 500000 tiyn. Returns 0 for anything unparseable. */
export function toTiyn(amount: string): number {
  const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(amount.trim());
  if (!match) return 0;

  const whole = Number(match[1]);
  const fraction = Number((match[2] ?? "0").padEnd(2, "0"));
  return whole * 100 + fraction;
}

/** Render tiyn as "5 000 ₸". */
export function formatTiyn(tiyn: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "KZT",
    currencyDisplay: "narrowSymbol",
    minimumFractionDigits: tiyn % 100 === 0 ? 0 : 2,
    maximumFractionDigits: 2,
  })
    .format(tiyn / 100)
    // Thin spaces read better than commas for KZT.
    .replace(/,/g, " ");
}

/** Render an API amount string as "5 000 ₸". */
export function formatKZT(amount: string): string {
  return formatTiyn(toTiyn(amount));
}

/** Total for `quantity` tickets at `unitPrice`, formatted. */
export function lineTotal(unitPrice: string, quantity: number): string {
  return formatTiyn(toTiyn(unitPrice) * quantity);
}
