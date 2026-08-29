"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";

import {
  EventFormFields,
  validateEventForm,
  type EventFormValues,
} from "@/components/event-form-fields";
import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { isoToLocalInput, localInputToISO, slugify, timezoneOptions } from "@/lib/datetime";
import type { BiletEvent } from "@/lib/types";

/**
 * Editing an event (SRS 4.2: "Create, edit, duplicate, publish, unpublish, and
 * cancel events").
 *
 * The API has had PATCH /events/{id} since Phase 2; nothing in the web app
 * reached it, so an organizer who mistyped a venue had no way to correct it.
 *
 * Only changed fields are sent. The API's PATCH is tri-state - absent leaves a
 * column alone, explicit null clears it - so sending the whole form back would
 * turn every untouched blank optional field into a deliberate erasure.
 */
export function EditEventForm({ eventID }: { eventID: string }) {
  const router = useRouter();
  const zones = useMemo(() => timezoneOptions(), []);

  const [event, setEvent] = useState<BiletEvent | null>(null);
  const [values, setValues] = useState<EventFormValues | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const loaded = await api.getEvent(eventID, signal);
        setEvent(loaded);
        setValues(toFormValues(loaded));
        setLoadError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setLoadError(
          cause instanceof ApiError ? cause.message : "Could not load this event.",
        );
      } finally {
        setLoading(false);
      }
    },
    [eventID],
  );

  useEffect(() => {
    const controller = new AbortController();
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  function update<K extends keyof EventFormValues>(key: K, value: EventFormValues[K]) {
    setValues((current) => (current ? { ...current, [key]: value } : current));
    setSaved(false);
  }

  async function handleSubmit(submitEvent: FormEvent<HTMLFormElement>) {
    submitEvent.preventDefault();
    if (!values || !event) return;

    setFormError(null);
    setFieldErrors({});
    setSaved(false);

    const { errors, startsISO, endsISO } = validateEventForm(values, localInputToISO);
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    const original = toFormValues(event);
    const patch: Record<string, unknown> = {};

    // Scalars: send only what moved.
    if (values.title.trim() !== original.title) patch.title = values.title.trim();
    if (values.timezone !== original.timezone) patch.timezone = values.timezone;
    if (values.visibility !== original.visibility) patch.visibility = values.visibility;
    if (startsISO && startsISO !== event.starts_at) patch.starts_at = startsISO;
    if (endsISO && endsISO !== event.ends_at) patch.ends_at = endsISO;
    if (!isPublished(event) && slugify(values.slug) !== original.slug) {
      patch.slug = slugify(values.slug);
    }

    if (values.capacity.trim() !== original.capacity) {
      patch.capacity = values.capacity.trim() === "" ? null : Number(values.capacity);
    }

    // Optional text: an emptied field is an explicit null, which is how the
    // API clears a column. That is only correct because the comparison above
    // means an untouched blank field is never sent at all.
    for (const [key, current, before] of [
      ["description", values.description, original.description],
      ["category", values.category, original.category],
      ["venue_name", values.venueName, original.venueName],
      ["venue_address", values.venueAddress, original.venueAddress],
      ["refund_policy", values.refundPolicy, original.refundPolicy],
      ["cover_image_url", values.coverImageURL, original.coverImageURL],
    ] as const) {
      if (current.trim() !== before) {
        patch[key] = current.trim() === "" ? null : current.trim();
      }
    }

    if (Object.keys(patch).length === 0) {
      setFormError("Nothing has changed yet.");
      return;
    }

    setSubmitting(true);
    try {
      const updated = await api.updateEvent(eventID, patch);
      setEvent(updated);
      setValues(toFormValues(updated));
      setSaved(true);
      router.refresh();
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        setFormError(
          Object.keys(cause.fields).length > 0
            ? "Please correct the highlighted fields."
            : cause.message,
        );
      } else {
        setFormError("Something went wrong. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center gap-3 py-16 text-sm text-foreground-muted">
        <Spinner aria-hidden />
        Loading this event…
      </div>
    );
  }

  if (loadError || !values || !event) {
    return <Alert tone="error">{loadError ?? "Could not load this event."}</Alert>;
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <Link
          href={`/dashboard/events/${eventID}`}
          className="text-sm text-foreground-muted hover:underline"
        >
          ← Back to {event.title}
        </Link>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Edit event</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          {isPublished(event)
            ? "This event is live. Changing when or where it happens emails everybody holding a ticket."
            : "This event is not published yet, so nobody is notified of changes."}
        </p>
      </div>

      {formError && <Alert tone="error">{formError}</Alert>}
      {saved && <Alert tone="success">Saved.</Alert>}

      <form
        onSubmit={handleSubmit}
        noValidate
        className="space-y-6 rounded-xl border border-border-subtle bg-surface p-6"
      >
        <EventFormFields
          values={values}
          onChange={update}
          fieldErrors={fieldErrors}
          disabled={submitting}
          zones={zones}
          slugLocked={isPublished(event)}
          optionalOpen
        />

        <div className="flex items-center gap-3 border-t border-border-subtle pt-5">
          <Button type="submit" loading={submitting} data-testid="save-event">
            {submitting ? "Saving…" : "Save changes"}
          </Button>
          <Link
            href={`/dashboard/events/${eventID}`}
            className="rounded-lg px-4 py-2.5 text-sm font-medium text-foreground-muted hover:bg-surface-muted"
          >
            Discard
          </Link>
        </div>
      </form>
    </div>
  );
}

/** A published or suspended event's URL is already in circulation. */
function isPublished(event: BiletEvent): boolean {
  return event.status === "published" || event.status === "suspended";
}

function toFormValues(event: BiletEvent): EventFormValues {
  return {
    title: event.title,
    slug: event.slug,
    capacity: event.capacity == null ? "" : String(event.capacity),
    timezone: event.timezone,
    startsAt: isoToLocalInput(event.starts_at, event.timezone),
    endsAt: isoToLocalInput(event.ends_at, event.timezone),
    description: event.description ?? "",
    coverImageURL: event.cover_image_url ?? "",
    refundPolicy: event.refund_policy ?? "",
    category: event.category ?? "",
    venueName: event.venue_name ?? "",
    venueAddress: event.venue_address ?? "",
    visibility: event.visibility,
  };
}
