"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";
import { formatKZT } from "@/lib/money";
import type { TicketType } from "@/lib/types";

export function TicketTypeManager({ eventID }: { eventID: string }) {
  const [types, setTypes] = useState<TicketType[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [name, setName] = useState("");
  const [price, setPrice] = useState("5000");
  const [quantity, setQuantity] = useState("5");
  const [maxPerOrder, setMaxPerOrder] = useState("10");

  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [busyID, setBusyID] = useState<string | null>(null);

  const load = useCallback(async (signal?: AbortSignal) => {
    try {
      setTypes(await api.listTicketTypes(eventID, signal));
      setLoadError(null);
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setLoadError(cause instanceof ApiError ? cause.message : "Could not load ticket types.");
    } finally {
      setLoading(false);
    }
  }, [eventID]);

  useEffect(() => {
    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    setFieldErrors({});

    const errors: Record<string, string> = {};
    if (!name.trim()) errors.name = "Name is required.";

    const quantityValue = Number(quantity);
    if (!Number.isInteger(quantityValue) || quantityValue < 0) {
      errors.quantity_total = "Quantity must be a whole number of zero or more.";
    }
    const maxValue = Number(maxPerOrder);
    if (!Number.isInteger(maxValue) || maxValue <= 0) {
      errors.max_per_order = "The per-order limit must be at least 1.";
    }
    // Sent as a string so the decimal reaches PostgreSQL's numeric untouched.
    const priceValue = price.trim() === "" ? "0" : price.trim();
    if (!/^\d+(\.\d{1,2})?$/.test(priceValue)) {
      errors.price_kzt = "Price must be a non-negative amount, such as 5000.";
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setSubmitting(true);
    try {
      const created = await api.createTicketType(eventID, {
        name: name.trim(),
        price_kzt: priceValue,
        quantity_total: quantityValue,
        max_per_order: maxValue,
      });
      setTypes((current) => [...current, created]);
      setName("");
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        setFormError(
          Object.keys(cause.fields).length > 0
            ? "Please correct the highlighted fields."
            : cause.message,
        );
      } else {
        setFormError("Could not create the ticket type.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  async function toggleHidden(type: TicketType) {
    setBusyID(type.id);
    try {
      const updated = await api.updateTicketType(type.id, { is_hidden: !type.is_hidden });
      setTypes((current) => current.map((t) => (t.id === updated.id ? updated : t)));
    } catch (cause) {
      setLoadError(cause instanceof ApiError ? cause.message : "The change failed.");
    } finally {
      setBusyID(null);
    }
  }

  async function remove(type: TicketType) {
    setBusyID(type.id);
    try {
      await api.deleteTicketType(type.id);
      setTypes((current) => current.filter((t) => t.id !== type.id));
    } catch (cause) {
      setLoadError(cause instanceof ApiError ? cause.message : "The ticket type could not be deleted.");
    } finally {
      setBusyID(null);
    }
  }

  return (
    <section className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">Ticket types</h2>
        <p className="mt-1 text-sm text-foreground-muted">
          Attendees choose from these on the public event page. A price of zero makes
          the ticket free.
        </p>
      </div>

      {loadError && <Alert tone="error">{loadError}</Alert>}

      {loading ? (
        <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface p-6 text-sm text-foreground-muted">
          <Spinner />
          Loading ticket types…
        </div>
      ) : types.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border-subtle bg-surface p-6 text-center text-sm text-foreground-muted">
          No ticket types yet. Add one below so attendees have something to buy.
        </p>
      ) : (
        <ul className="space-y-2">
          {types.map((type) => (
            <li
              key={type.id}
              className="flex flex-wrap items-center gap-4 rounded-xl border border-border-subtle bg-surface p-4"
            >
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{type.name}</span>
                  {type.is_free && (
                    <span className="rounded-full bg-brand-soft px-2 py-0.5 text-xs font-medium text-brand-strong">
                      Free
                    </span>
                  )}
                  {type.is_hidden && (
                    <span className="rounded-full bg-surface-muted px-2 py-0.5 text-xs font-medium text-foreground-muted">
                      Hidden
                    </span>
                  )}
                </div>
                <p className="mt-0.5 text-sm text-foreground-muted">
                  {formatKZT(type.price_kzt)} · max {type.max_per_order} per order
                </p>
              </div>

              {/*
                SRS 4.3: "View the number of available, reserved, sold,
                refunded, and checked-in tickets." Reserved and refunded are
                shown only when they are non-zero - a column of permanent
                zeroes buries the three numbers an organizer actually reads.
              */}
              <dl className="flex flex-wrap gap-x-6 gap-y-2 text-sm" data-testid="ticket-type-counts">
                <div>
                  <dt className="text-xs text-foreground-muted">Sold</dt>
                  <dd className="mt-0.5 font-medium">{type.quantity_sold}</dd>
                </div>
                <div>
                  <dt className="text-xs text-foreground-muted">Available</dt>
                  <dd
                    className={`mt-0.5 font-medium ${
                      type.quantity_remaining === 0 ? "text-danger" : ""
                    }`}
                  >
                    {type.quantity_remaining}
                  </dd>
                </div>
                {type.quantity_reserved > 0 && (
                  <div>
                    <dt className="text-xs text-foreground-muted">Reserved</dt>
                    <dd className="mt-0.5 font-medium">{type.quantity_reserved}</dd>
                  </div>
                )}
                {type.quantity_refunded > 0 && (
                  <div>
                    <dt className="text-xs text-foreground-muted">Refunded</dt>
                    <dd className="mt-0.5 font-medium">{type.quantity_refunded}</dd>
                  </div>
                )}
                {type.quantity_checked_in > 0 && (
                  <div>
                    <dt className="text-xs text-foreground-muted">Checked in</dt>
                    <dd className="mt-0.5 font-medium text-success">
                      {type.quantity_checked_in}
                    </dd>
                  </div>
                )}
                <div>
                  <dt className="text-xs text-foreground-muted">Total</dt>
                  <dd className="mt-0.5 font-medium">{type.quantity_total}</dd>
                </div>
              </dl>

              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={busyID !== null}
                  loading={busyID === type.id}
                  onClick={() => void toggleHidden(type)}
                >
                  {type.is_hidden ? "Show" : "Hide"}
                </Button>
                {type.quantity_sold === 0 && (
                  <Button
                    size="sm"
                    variant="danger"
                    disabled={busyID !== null}
                    onClick={() => void remove(type)}
                  >
                    Delete
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      <form
        onSubmit={handleCreate}
        noValidate
        className="space-y-4 rounded-xl border border-border-subtle bg-surface p-5"
      >
        <h3 className="text-sm font-semibold">Add a ticket type</h3>

        {formError && <Alert tone="error">{formError}</Alert>}

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <TextField
            label="Name"
            name="name"
            placeholder="General Admission"
            required
            value={name}
            error={fieldErrors.name}
            disabled={submitting}
            onChange={(event) => setName(event.target.value)}
          />
          <TextField
            label="Price (KZT)"
            name="price_kzt"
            inputMode="decimal"
            placeholder="5000"
            hint="0 for a free ticket."
            value={price}
            error={fieldErrors.price_kzt}
            disabled={submitting}
            onChange={(event) => setPrice(event.target.value)}
          />
          <TextField
            label="Quantity"
            name="quantity_total"
            type="number"
            min={0}
            step={1}
            required
            value={quantity}
            error={fieldErrors.quantity_total}
            disabled={submitting}
            onChange={(event) => setQuantity(event.target.value)}
          />
          <TextField
            label="Max per order"
            name="max_per_order"
            type="number"
            min={1}
            step={1}
            value={maxPerOrder}
            error={fieldErrors.max_per_order}
            disabled={submitting}
            onChange={(event) => setMaxPerOrder(event.target.value)}
          />
        </div>

        <Button type="submit" loading={submitting}>
          {submitting ? "Adding…" : "Add ticket type"}
        </Button>
      </form>
    </section>
  );
}
