"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";

import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { formatKZT, formatTiyn, toTiyn } from "@/lib/money";
import type { PromoPreview, TicketType } from "@/lib/types";

interface Line {
  type: TicketType;
  quantity: number;
}

/** How long to wait for the basket to settle before re-pricing a discount. */
const REPRICE_DELAY_MS = 250;

/**
 * The promo code panel on the public event page.
 *
 * Two ways in, one behaviour: an attendee types a code, or arrives through a
 * campaign QR whose link carries `?c=CMP_...` and the code applies itself.
 *
 * Every figure shown here is priced by the server. The component sends the
 * basket and renders the discount it is given back; it never computes one, so
 * what appears on screen is exactly what checkout will charge.
 */
export function PromoBox({
  eventID,
  campaignToken,
  lines,
  promo,
  onChange,
}: {
  eventID: string;
  campaignToken?: string;
  lines: Line[];
  promo: PromoPreview | null;
  onChange: (next: PromoPreview | null) => void;
}) {
  const [code, setCode] = useState("");
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Once an attendee removes a code, it stays removed - otherwise arriving via
  // a campaign link would re-apply it the moment they changed their basket.
  const [dismissed, setDismissed] = useState(false);

  const basket = lines.map((line) => ({
    ticket_type_id: line.type.id,
    quantity: line.quantity,
  }));
  // The array is rebuilt on every render, so its contents are the dependency.
  const basketKey = JSON.stringify(basket);

  // The basket a discount was last priced against.
  const pricedFor = useRef<string | null>(null);

  const apply = useCallback(
    async (input: { code?: string; campaign_token?: string }, items: typeof basket) => {
      setChecking(true);
      setError(null);
      try {
        const preview = await api.previewPromo(eventID, { ...input, items });
        onChange(preview);
      } catch (cause) {
        onChange(null);
        setError(
          cause instanceof ApiError ? cause.message : "That code could not be applied.",
        );
      } finally {
        setChecking(false);
      }
    },
    [eventID, onChange],
  );

  /**
   * Keeps the shown discount honest.
   *
   * A percentage discount depends on the basket, so a figure priced against a
   * different selection would be a lie. This applies the campaign token on
   * arrival and re-prices whenever the selection changes - and clears the
   * discount when the basket empties.
   */
  useEffect(() => {
    const items = JSON.parse(basketKey) as typeof basket;

    // An empty basket is cleared by the selector that emptied it, in the event
    // handler that did so - not with a setState from inside this effect.
    if (items.length === 0) {
      pricedFor.current = null;
      return;
    }
    if (pricedFor.current === basketKey) return;

    const active = promo
      ? { code: promo.code }
      : !dismissed && campaignToken
        ? { campaign_token: campaignToken }
        : null;
    if (!active) return;

    pricedFor.current = basketKey;

    // Deferred rather than called straight from the effect, for two reasons:
    // the request's state updates then land after this render instead of
    // cascading one, and tapping "+" three times in a row re-prices once
    // instead of three times.
    const timer = setTimeout(() => void apply(active, items), REPRICE_DELAY_MS);
    return () => clearTimeout(timer);
  }, [basketKey, promo, campaignToken, dismissed, apply]);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const typed = code.trim();
    if (!typed) return;

    if (basket.length === 0) {
      setError("Choose your tickets first, then the discount can be calculated.");
      return;
    }

    setDismissed(false);
    pricedFor.current = basketKey;
    void apply({ code: typed }, basket);
  }

  function clear() {
    onChange(null);
    setCode("");
    setError(null);
    setDismissed(true);
    pricedFor.current = null;
  }

  if (promo) {
    const discountTiyn = toTiyn(promo.discount_kzt);

    return (
      <div
        className="rounded-xl border border-success/40 bg-success-soft p-5"
        data-testid="promo-applied"
      >
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm font-semibold text-success">
              Promo code {promo.code} applied
            </p>
            <p className="mt-0.5 text-xs text-success/80">
              {promo.campaign_name}
              {promo.discount_type === "percentage"
                ? ` · ${Number(promo.discount_value)}% off`
                : ` · ${formatKZT(promo.discount_value)} off`}
              {!promo.applies_to_all && " · applies to selected ticket types only"}
            </p>
          </div>
          <Button size="sm" variant="secondary" onClick={clear}>
            Remove
          </Button>
        </div>

        <dl className="mt-4 space-y-1.5 text-sm">
          <div className="flex justify-between">
            <dt className="text-success/80">Subtotal</dt>
            <dd className="tabular-nums text-success/80">{formatKZT(promo.subtotal_kzt)}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-success/80">Discount</dt>
            <dd className="tabular-nums font-medium text-success" data-testid="promo-discount">
              −{formatTiyn(discountTiyn)}
            </dd>
          </div>
          <div className="flex justify-between border-t border-success/30 pt-1.5">
            <dt className="font-semibold text-success">New total</dt>
            <dd
              className="text-base font-bold tabular-nums text-success"
              data-testid="promo-total"
            >
              {formatKZT(promo.total_kzt)}
            </dd>
          </div>
        </dl>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-xl border border-border-subtle bg-surface p-5"
    >
      <label htmlFor="promo-code" className="text-sm font-medium">
        Have a promo code?
      </label>

      <div className="mt-2 flex gap-2">
        <input
          id="promo-code"
          name="promo_code"
          value={code}
          onChange={(event) => setCode(event.target.value.toUpperCase())}
          placeholder="SPRING20"
          autoCapitalize="characters"
          autoCorrect="off"
          spellCheck={false}
          disabled={checking}
          aria-invalid={error ? true : undefined}
          className={`min-w-0 flex-1 rounded-lg border bg-surface px-3 py-2 font-mono text-sm uppercase placeholder:font-sans placeholder:normal-case placeholder:text-foreground-muted/70 ${
            error ? "border-danger" : "border-border-subtle"
          }`}
          data-testid="promo-input"
        />
        <Button type="submit" variant="secondary" disabled={checking || !code.trim()}>
          {checking ? <Spinner /> : "Apply"}
        </Button>
      </div>

      {error && (
        <p className="mt-2 text-xs text-danger" role="alert" data-testid="promo-error">
          {error}
        </p>
      )}
      {checking && !error && (
        <p className="mt-2 text-xs text-foreground-muted">Checking the code…</p>
      )}
    </form>
  );
}
