"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { SelectField, TextField } from "@/components/ui/field";
import { ApiError, api, campaignQRURL } from "@/lib/api";
import { formatKZT } from "@/lib/money";
import type { Campaign, DiscountType } from "@/lib/types";

const DISCOUNT_TYPES = ["percentage", "fixed_amount"] as const;

/** The organizer's promo-code and campaign-QR workspace for one event. */
export function CampaignManager({ eventID }: { eventID: string }) {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [discountType, setDiscountType] = useState<DiscountType>("percentage");
  const [discountValue, setDiscountValue] = useState("20");
  const [maxRedemptions, setMaxRedemptions] = useState("");

  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [busyID, setBusyID] = useState<string | null>(null);
  const [copiedID, setCopiedID] = useState<string | null>(null);

  const load = useCallback(async (signal?: AbortSignal) => {
    try {
      setCampaigns(await api.listCampaigns(eventID, signal));
      setLoadError(null);
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setLoadError(cause instanceof ApiError ? cause.message : "Could not load campaigns.");
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
    if (!name.trim()) errors.name = "Give the campaign a name.";
    if (!/^[A-Za-z0-9_-]{3,32}$/.test(code.trim())) {
      errors.code = "Use 3-32 letters, digits, hyphens or underscores.";
    }
    if (!/^\d+(\.\d{1,2})?$/.test(discountValue.trim()) || Number(discountValue) <= 0) {
      errors.discount_value = "Enter a discount greater than zero.";
    } else if (discountType === "percentage" && Number(discountValue) > 100) {
      errors.discount_value = "A percentage cannot exceed 100.";
    }

    let limit: number | undefined;
    if (maxRedemptions.trim() !== "") {
      limit = Number(maxRedemptions);
      if (!Number.isInteger(limit) || limit <= 0) {
        errors.max_redemptions = "The limit must be a whole number of at least 1.";
      }
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setSubmitting(true);
    try {
      const created = await api.createCampaign(eventID, {
        name: name.trim(),
        code: code.trim().toUpperCase(),
        discount_type: discountType,
        // Sent as a string so the decimal reaches numeric(14,2) untouched.
        discount_value: discountValue.trim(),
        ...(limit !== undefined ? { max_redemptions: limit } : {}),
      });
      setCampaigns((current) => [created, ...current]);
      setName("");
      setCode("");
      setMaxRedemptions("");
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        setFormError(
          Object.keys(cause.fields).length > 0
            ? "Please correct the highlighted fields."
            : cause.message,
        );
      } else {
        setFormError("Could not create the campaign.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  async function toggleActive(campaign: Campaign) {
    setBusyID(campaign.id);
    setLoadError(null);
    try {
      const updated = await api.setCampaignActive(campaign.id, campaign.status !== "active");
      setCampaigns((current) => current.map((c) => (c.id === updated.id ? updated : c)));
    } catch (cause) {
      setLoadError(cause instanceof ApiError ? cause.message : "The change failed.");
    } finally {
      setBusyID(null);
    }
  }

  async function remove(campaign: Campaign) {
    setBusyID(campaign.id);
    setLoadError(null);
    try {
      await api.deleteCampaign(campaign.id);
      setCampaigns((current) => current.filter((c) => c.id !== campaign.id));
    } catch (cause) {
      setLoadError(cause instanceof ApiError ? cause.message : "The campaign could not be deleted.");
    } finally {
      setBusyID(null);
    }
  }

  async function copyLink(campaign: Campaign) {
    try {
      await navigator.clipboard.writeText(campaign.campaign_url);
      setCopiedID(campaign.id);
      setTimeout(() => setCopiedID(null), 2000);
    } catch {
      setLoadError("Could not copy the link. Select it from the field instead.");
    }
  }

  return (
    <section className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">Promo codes &amp; campaigns</h2>
        <p className="mt-1 text-sm text-foreground-muted">
          Attendees can type the code at checkout, or scan the campaign QR to open the
          event with the discount already applied. The QR carries only an opaque token —
          the discount is always decided by the server.
        </p>
      </div>

      {loadError && <Alert tone="error">{loadError}</Alert>}

      {loading ? (
        <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface p-6 text-sm text-foreground-muted">
          <Spinner />
          Loading campaigns…
        </div>
      ) : campaigns.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border-subtle bg-surface p-6 text-center text-sm text-foreground-muted">
          No campaigns yet. Create a promo code below to start tracking attributed sales.
        </p>
      ) : (
        <ul className="space-y-3">
          {campaigns.map((campaign) => (
            <li
              key={campaign.id}
              className="rounded-xl border border-border-subtle bg-surface p-5"
            >
              <div className="flex flex-wrap items-start gap-5">
                <div className="shrink-0 rounded-lg bg-white p-2">
                  {/* A per-campaign API image on another origin, served no-store:
                      routing it through the image optimizer would cache and
                      re-encode it, and a re-encoded QR may not scan. */}
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={campaignQRURL(campaign.id)}
                    alt={`Campaign QR code for ${campaign.code}`}
                    width={112}
                    height={112}
                    className="h-28 w-28"
                  />
                </div>

                <div className="min-w-0 flex-1 space-y-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-base font-semibold">{campaign.code}</span>
                    <StatusPill status={campaign.status} />
                    <span className="text-sm text-foreground-muted">{campaign.name}</span>
                  </div>

                  <p className="text-sm">
                    <span className="font-medium">
                      {campaign.discount_type === "percentage"
                        ? `${Number(campaign.discount_value)}% off`
                        : `${formatKZT(campaign.discount_value)} off`}
                    </span>
                    <span className="ml-3 text-foreground-muted">
                      {campaign.max_redemptions
                        ? `${campaign.redemption_count} of ${campaign.max_redemptions} used`
                        : `${campaign.redemption_count} used · unlimited`}
                    </span>
                  </p>

                  <dl className="flex flex-wrap gap-x-8 gap-y-2 text-sm">
                    <div>
                      <dt className="text-xs text-foreground-muted">Orders</dt>
                      <dd className="mt-0.5 font-medium">{campaign.orders_count}</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-foreground-muted">Tickets</dt>
                      <dd className="mt-0.5 font-medium">{campaign.tickets_sold}</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-foreground-muted">Revenue</dt>
                      <dd className="mt-0.5 font-medium">
                        {formatKZT(campaign.gross_revenue_kzt)}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs text-foreground-muted">Discounted</dt>
                      <dd className="mt-0.5 font-medium">
                        {formatKZT(campaign.discount_given_kzt)}
                      </dd>
                    </div>
                  </dl>

                  <div className="flex flex-wrap items-center gap-2">
                    <input
                      readOnly
                      value={campaign.campaign_url}
                      aria-label={`Campaign link for ${campaign.code}`}
                      onFocus={(event) => event.currentTarget.select()}
                      className="min-w-0 flex-1 rounded-lg border border-border-subtle bg-surface-muted/50 px-3 py-2 font-mono text-xs text-foreground-muted"
                    />
                    <Button size="sm" variant="secondary" onClick={() => void copyLink(campaign)}>
                      {copiedID === campaign.id ? "Copied" : "Copy link"}
                    </Button>
                    <a
                      href={campaignQRURL(campaign.id)}
                      download={`biletflow-campaign-${campaign.code}.png`}
                      className="inline-flex items-center rounded-lg border border-border-subtle px-3 py-1.5 text-xs font-medium hover:bg-surface-muted"
                    >
                      Download QR
                    </a>
                    <Button
                      size="sm"
                      variant="secondary"
                      loading={busyID === campaign.id}
                      disabled={busyID !== null || campaign.status === "exhausted"}
                      onClick={() => void toggleActive(campaign)}
                    >
                      {campaign.status === "active" ? "Disable" : "Enable"}
                    </Button>
                    {campaign.redemption_count === 0 && (
                      <Button
                        size="sm"
                        variant="danger"
                        disabled={busyID !== null}
                        onClick={() => void remove(campaign)}
                      >
                        Delete
                      </Button>
                    )}
                  </div>
                </div>
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
        <h3 className="text-sm font-semibold">Create a promo code</h3>

        {formError && <Alert tone="error">{formError}</Alert>}

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          <TextField
            label="Campaign name"
            name="name"
            placeholder="Spring Promotion"
            required
            value={name}
            error={fieldErrors.name}
            disabled={submitting}
            onChange={(event) => setName(event.target.value)}
          />
          <TextField
            label="Promo code"
            name="code"
            placeholder="SPRING20"
            hint="Shown to attendees."
            required
            value={code}
            error={fieldErrors.code}
            disabled={submitting}
            onChange={(event) => setCode(event.target.value.toUpperCase())}
          />
          <SelectField
            label="Discount type"
            name="discount_type"
            options={DISCOUNT_TYPES}
            value={discountType}
            error={fieldErrors.discount_type}
            disabled={submitting}
            onChange={(event) => setDiscountType(event.target.value as DiscountType)}
          />
          <TextField
            label={discountType === "percentage" ? "Percent off" : "KZT off"}
            name="discount_value"
            inputMode="decimal"
            placeholder={discountType === "percentage" ? "20" : "1500"}
            required
            value={discountValue}
            error={fieldErrors.discount_value}
            disabled={submitting}
            onChange={(event) => setDiscountValue(event.target.value)}
          />
          <TextField
            label="Max redemptions"
            name="max_redemptions"
            type="number"
            min={1}
            step={1}
            placeholder="Unlimited"
            hint="Leave blank for no limit."
            value={maxRedemptions}
            error={fieldErrors.max_redemptions}
            disabled={submitting}
            onChange={(event) => setMaxRedemptions(event.target.value)}
          />
        </div>

        <Button type="submit" loading={submitting}>
          {submitting ? "Creating…" : "Create promo code"}
        </Button>
      </form>
    </section>
  );
}

const STATUS_STYLES: Record<string, string> = {
  active: "bg-success-soft text-success",
  disabled: "bg-surface-muted text-foreground-muted",
  exhausted: "bg-warning-soft text-warning",
  expired: "bg-danger-soft text-danger",
  draft: "bg-surface-muted text-foreground-muted",
};

function StatusPill({ status }: { status: string }) {
  return (
    <span
      className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${
        STATUS_STYLES[status] ?? STATUS_STYLES.draft
      }`}
    >
      {status}
    </span>
  );
}
