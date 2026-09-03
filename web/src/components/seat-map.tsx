"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { formatKZT, formatTiyn, toTiyn } from "@/lib/money";
import type { Seat, SeatMap as SeatMapData, SeatSection, SeatStatus } from "@/lib/types";

/**
 * How the six seat states are drawn (SRS 4.3.1).
 *
 * The SRS requires them to be distinguished "with both colors and text or
 * symbols", and that is not decoration: roughly one man in twelve cannot tell
 * the green from the red. Every state therefore carries a shape or a letter as
 * well as a fill, and the legend spells all of them out.
 */
type SeatState = SeatStatus | "selected";

const SEAT_STYLE: Record<
  SeatState,
  { fill: string; stroke: string; label: string; symbol: string }
> = {
  available: {
    fill: "var(--color-surface)",
    stroke: "var(--color-brand)",
    label: "Available",
    symbol: "",
  },
  selected: {
    fill: "var(--color-brand)",
    stroke: "var(--color-brand-strong)",
    label: "Selected",
    symbol: "✓",
  },
  held: {
    fill: "var(--color-warning-soft)",
    stroke: "var(--color-warning)",
    label: "Held by someone",
    symbol: "⏳",
  },
  sold: {
    fill: "var(--color-surface-muted)",
    stroke: "var(--color-border-subtle)",
    label: "Sold",
    symbol: "×",
  },
  unavailable: {
    fill: "transparent",
    stroke: "var(--color-border-subtle)",
    label: "Not for sale",
    symbol: "−",
  },
};

/** Seat geometry, in the same units the API reports coordinates in. */
const SEAT_RADIUS = 9;
const PADDING = 26;

/**
 * The interactive seat map (SRS 4.3.1).
 *
 * SVG rather than canvas: every seat is a real DOM node, so it can be a
 * `<button>` with a label a screen reader can read and a keyboard can reach.
 * A canvas would have meant reimplementing hit-testing, focus and
 * accessibility by hand, and doing all three worse.
 */
export function SeatMapPicker({
  eventID,
  onHeld,
}: {
  eventID: string;
  /** Called once the seats are reserved, with the basket to check out. */
  onHeld: (orderID: string) => void;
}) {
  const [plan, setPlan] = useState<SeatMapData | null>(null);
  const [selected, setSelected] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [holding, setHolding] = useState(false);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        setPlan(await api.seatMap(eventID, signal));
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load the seat map.");
      } finally {
        setLoading(false);
      }
    },
    [eventID],
  );

  useEffect(() => {
    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  /** Where each seat is priced, so a click can show a total immediately. */
  const priceBySeat = useMemo(() => {
    const prices = new Map<string, { section: SeatSection; price: string }>();
    for (const section of plan?.sections ?? []) {
      for (const row of section.rows) {
        for (const seat of row.seats) {
          prices.set(seat.id, { section, price: section.price_kzt ?? "0" });
        }
      }
    }
    return prices;
  }, [plan]);

  const seatLabel = useCallback(
    (seat: Seat, section: SeatSection, row: string) =>
      `${section.name}, row ${row}, seat ${seat.number}` +
      (seat.accessible ? ", accessible" : "") +
      (section.price_kzt ? `, ${formatKZT(section.price_kzt)}` : ""),
    [],
  );

  function toggle(seat: Seat) {
    if (seat.status !== "available") return;
    setSelected((current) =>
      current.includes(seat.id)
        ? current.filter((id) => id !== seat.id)
        : [...current, seat.id],
    );
  }

  const totalTiyn = selected.reduce(
    (sum, id) => sum + toTiyn(priceBySeat.get(id)?.price ?? "0"),
    0,
  );

  async function reserve() {
    setHolding(true);
    setError(null);
    try {
      const hold = await api.holdSeats(eventID, selected);
      onHeld(hold.order_id);
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.message
          : "Could not reserve those seats. Try again.",
      );
      // Somebody else may have taken one while this browser was deciding, so
      // the map is re-read rather than left showing a seat that is now gone.
      await load();
      setSelected([]);
    } finally {
      setHolding(false);
    }
  }

  if (loading) {
    return <p className="text-sm text-foreground-muted">Loading the seat map…</p>;
  }
  if (!plan) {
    return <Alert>{error ?? "No seat map for this event."}</Alert>;
  }

  const width = plan.max_x - plan.min_x + PADDING * 2;
  const height = plan.max_y - plan.min_y + PADDING * 2;

  return (
    <section className="space-y-4" aria-labelledby="seat-map-heading">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="seat-map-heading" className="text-lg font-semibold tracking-tight">
            Choose your seats
          </h2>
          <p className="mt-1 text-sm text-foreground-muted">
            {plan.venue_name} · {plan.available_seats} of {plan.total_seats} seats free
          </p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void load()}>
          Refresh
        </Button>
      </div>

      {error && <Alert>{error}</Alert>}

      <div className="overflow-x-auto rounded-xl border border-border-subtle bg-surface p-4">
        <svg
          viewBox={`${plan.min_x - PADDING} ${plan.min_y - PADDING} ${width} ${height}`}
          className="mx-auto block h-auto w-full max-w-2xl"
          role="group"
          aria-label={`Seat map for ${plan.venue_name}`}
        >
          {/* The stage, so the map has an orientation. */}
          <rect
            x={plan.min_x - PADDING / 2}
            y={plan.min_y - PADDING}
            width={width - PADDING}
            height={12}
            rx={4}
            fill="var(--color-surface-muted)"
          />
          <text
            x={plan.min_x + (width - PADDING * 2) / 2}
            y={plan.min_y - PADDING + 9}
            textAnchor="middle"
            fontSize={7}
            fill="var(--color-foreground-muted)"
          >
            STAGE
          </text>

          {plan.sections.map((section) =>
            section.rows.map((row) =>
              row.seats.map((seat) => {
                const state: SeatState = selected.includes(seat.id)
                  ? "selected"
                  : seat.status;
                const style = SEAT_STYLE[state];
                const clickable = seat.status === "available" || state === "selected";

                return (
                  <g key={seat.id}>
                    <circle
                      cx={seat.x}
                      cy={seat.y}
                      r={SEAT_RADIUS}
                      fill={style.fill}
                      stroke={style.stroke}
                      strokeWidth={state === "selected" ? 2.5 : 1.5}
                      // An accessible seat is dashed as well as marked, so it
                      // reads as different at a glance and without colour.
                      strokeDasharray={seat.accessible ? "3 2" : undefined}
                      className={clickable ? "cursor-pointer" : "cursor-not-allowed"}
                      onClick={() => toggle(seat)}
                      role="button"
                      tabIndex={clickable ? 0 : -1}
                      aria-disabled={!clickable}
                      aria-pressed={state === "selected"}
                      aria-label={`${seatLabel(seat, section, row.label)} — ${style.label}`}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          toggle(seat);
                        }
                      }}
                    />
                    {/* The symbol carries the state for anybody who cannot
                        rely on the fill colour. */}
                    <text
                      x={seat.x}
                      y={seat.y + 3}
                      textAnchor="middle"
                      fontSize={8}
                      pointerEvents="none"
                      fill={
                        state === "selected"
                          ? "#fff"
                          : "var(--color-foreground-muted)"
                      }
                    >
                      {style.symbol || (seat.accessible ? "♿" : seat.number)}
                    </text>
                  </g>
                );
              }),
            ),
          )}
        </svg>
      </div>

      {/* The legend names every state in words, which is the other half of
          "colors and text or symbols". */}
      <ul className="flex flex-wrap gap-x-5 gap-y-2 text-xs text-foreground-muted">
        {(Object.keys(SEAT_STYLE) as SeatState[]).map((state) => (
          <li key={state} className="flex items-center gap-1.5">
            <span
              aria-hidden
              className="inline-block size-3 rounded-full border"
              style={{
                background: SEAT_STYLE[state].fill,
                borderColor: SEAT_STYLE[state].stroke,
              }}
            />
            <span>
              {SEAT_STYLE[state].symbol && (
                <span aria-hidden className="mr-1">
                  {SEAT_STYLE[state].symbol}
                </span>
              )}
              {SEAT_STYLE[state].label}
            </span>
          </li>
        ))}
        <li className="flex items-center gap-1.5">
          <span aria-hidden>♿</span>
          <span>Accessible (dashed outline)</span>
        </li>
      </ul>

      <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border-subtle bg-surface-muted p-4">
        <div>
          <p className="text-sm font-medium">
            {selected.length === 0
              ? "No seats selected"
              : `${selected.length} seat${selected.length === 1 ? "" : "s"} selected`}
          </p>
          {selected.length > 0 && (
            <p className="mt-0.5 text-xs text-foreground-muted">
              {selected
                .map((id) => {
                  const entry = priceBySeat.get(id);
                  return entry ? `${entry.section.name} ${formatKZT(entry.price)}` : "";
                })
                .filter(Boolean)
                .join(" · ")}
            </p>
          )}
        </div>

        <div className="flex items-center gap-3">
          {/* The price before continuing, which SRS 4.3.1 asks for by name. */}
          <span className="text-lg font-semibold tabular-nums">
            {formatTiyn(totalTiyn)}
          </span>
          <Button
            disabled={selected.length === 0}
            loading={holding}
            onClick={() => void reserve()}
          >
            Hold these seats
          </Button>
        </div>
      </div>
    </section>
  );
}
