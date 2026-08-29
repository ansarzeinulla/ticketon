"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState, type FormEvent } from "react";

import {
  EventFormFields,
  validateEventForm,
  type EventFormValues,
} from "@/components/event-form-fields";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import {
  browserTimezone,
  defaultLocalInput,
  localInputToISO,
  slugify,
  timezoneOptions,
} from "@/lib/datetime";
import type { CreateEventInput } from "@/lib/types";

export function CreateEventForm() {
  const router = useRouter();
  const { refresh } = useAuth();

  const zones = useMemo(() => timezoneOptions(), []);
  const defaultZone = useMemo(() => {
    const browser = browserTimezone();
    return browser && zones.includes(browser) ? browser : "Asia/Almaty";
  }, [zones]);

  const [values, setValues] = useState<EventFormValues>(() => ({
    title: "",
    slug: "",
    capacity: "200",
    timezone: defaultZone,
    startsAt: defaultLocalInput(30, 19),
    endsAt: defaultLocalInput(30, 22),
    description: "",
    coverImageURL: "",
    refundPolicy: "",
    category: "",
    venueName: "",
    venueAddress: "",
    visibility: "public",
  }));

  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  function update<K extends keyof EventFormValues>(key: K, value: EventFormValues[K]) {
    setValues((current) => ({ ...current, [key]: value }));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    setFieldErrors({});

    const { errors, startsISO, endsISO } = validateEventForm(values, localInputToISO);
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    const capacityValue = values.capacity.trim() === "" ? undefined : Number(values.capacity);

    const payload: CreateEventInput = {
      title: values.title.trim(),
      starts_at: startsISO as string,
      ends_at: endsISO as string,
      timezone: values.timezone,
      visibility: values.visibility as CreateEventInput["visibility"],
      ...(values.slug.trim() ? { slug: slugify(values.slug) } : {}),
      ...(capacityValue !== undefined ? { capacity: capacityValue } : {}),
      ...(values.description.trim() ? { description: values.description.trim() } : {}),
      ...(values.category.trim() ? { category: values.category.trim() } : {}),
      ...(values.venueName.trim() ? { venue_name: values.venueName.trim() } : {}),
      ...(values.venueAddress.trim() ? { venue_address: values.venueAddress.trim() } : {}),
      ...(values.coverImageURL ? { cover_image_url: values.coverImageURL } : {}),
      ...(values.refundPolicy.trim() ? { refund_policy: values.refundPolicy.trim() } : {}),
    };

    setSubmitting(true);
    try {
      await api.createEvent(payload);

      // Creating an event grants the organizer role, so pick up the new roles.
      void refresh();

      router.push("/dashboard");
      router.refresh();
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fields);
        setFormError(
          Object.keys(error.fields).length > 0
            ? "Please correct the highlighted fields."
            : error.message,
        );
      } else {
        setFormError("Something went wrong. Please try again.");
      }
      setSubmitting(false);
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <Link href="/dashboard" className="text-sm text-foreground-muted hover:underline">
          ← Back to events
        </Link>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Create an event</h1>
        <p className="mt-1 text-sm text-foreground-muted">
          Your event is saved as a draft. Publish it from the dashboard when it is ready.
        </p>
      </div>

      {formError && <Alert tone="error">{formError}</Alert>}

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
        />

        <div className="flex items-center gap-3 border-t border-border-subtle pt-5">
          <Button type="submit" loading={submitting}>
            {submitting ? "Creating…" : "Create event"}
          </Button>
          <Link
            href="/dashboard"
            className="rounded-lg px-4 py-2.5 text-sm font-medium text-foreground-muted hover:bg-surface-muted"
          >
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}
