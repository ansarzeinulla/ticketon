"use client";

import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { ApiError, api } from "@/lib/api";
import { formatInTimezone } from "@/lib/datetime";
import type { BiletEvent } from "@/lib/types";

export function EventCard({
  event,
  onChanged,
}: {
  event: BiletEvent;
  onChanged: (updated: BiletEvent) => void;
}) {
  const [busy, setBusy] = useState<null | "publish" | "unpublish" | "cancel">(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmingCancel, setConfirmingCancel] = useState(false);

  async function run(action: "publish" | "unpublish" | "cancel") {
    setBusy(action);
    setError(null);
    try {
      const updated =
        action === "publish"
          ? await api.publishEvent(event.id)
          : action === "unpublish"
            ? await api.unpublishEvent(event.id)
            : await api.cancelEvent(event.id);
      onChanged(updated);
      setConfirmingCancel(false);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "The action failed.");
    } finally {
      setBusy(null);
    }
  }

  const canPublish = event.status === "draft" || event.status === "unpublished";
  // SRS 4.2 lists unpublish alongside publish. It takes the page down while
  // the organizer reworks the event, and - unlike cancelling - it is
  // reversible and tells nobody.
  const canUnpublish = event.status === "published";
  const canCancel = event.status !== "cancelled" && event.status !== "completed";

  return (
    <li className="rounded-xl border border-border-subtle bg-surface p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate text-base font-semibold">{event.title}</h3>
            <StatusBadge status={event.status} />
            <span className="rounded-full bg-surface-muted px-2 py-0.5 text-xs capitalize text-foreground-muted">
              {event.lifecycle}
            </span>
          </div>
          <p className="mt-1 font-mono text-xs text-foreground-muted">/{event.slug}</p>
        </div>

        <div className="flex shrink-0 flex-wrap gap-2">
          <Link
            href={`/dashboard/events/${event.id}`}
            className="inline-flex items-center rounded-lg border border-border-subtle px-3 py-1.5 text-xs font-medium hover:bg-surface-muted"
          >
            Manage tickets
          </Link>
          {event.status === "published" && (
            <Link
              href={`/events/${event.slug}`}
              className="inline-flex items-center rounded-lg border border-border-subtle px-3 py-1.5 text-xs font-medium hover:bg-surface-muted"
            >
              Public page
            </Link>
          )}
          <Link
            href={`/dashboard/events/${event.id}/edit`}
            className="inline-flex items-center rounded-lg border border-border-subtle px-3 py-1.5 text-xs font-medium hover:bg-surface-muted"
            data-testid="edit-event"
          >
            Edit
          </Link>
          {canPublish && (
            <Button
              size="sm"
              loading={busy === "publish"}
              disabled={busy !== null}
              onClick={() => run("publish")}
            >
              Publish
            </Button>
          )}
          {canUnpublish && (
            <Button
              size="sm"
              variant="secondary"
              loading={busy === "unpublish"}
              disabled={busy !== null}
              onClick={() => run("unpublish")}
              data-testid="unpublish-event"
            >
              Unpublish
            </Button>
          )}
          {canCancel && !confirmingCancel && (
            <Button
              size="sm"
              variant="danger"
              disabled={busy !== null}
              onClick={() => setConfirmingCancel(true)}
            >
              Cancel event
            </Button>
          )}
        </div>
      </div>

      {/*
        Cancelling is final and emails every ticket holder, so it does not fire
        on a single click. Unpublishing is the reversible action, and it is
        offered right here so the destructive one is never the only way to take
        a page down.
      */}
      {confirmingCancel && (
        <div className="mt-4 space-y-2 rounded-lg border border-danger/30 bg-danger-soft/40 p-3">
          <p className="text-sm">
            Cancelling <strong>{event.title}</strong> is final. Every ticket holder is
            emailed, their tickets stop working, and the event cannot be published
            again. To take the page down temporarily, use Unpublish instead.
          </p>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="danger"
              loading={busy === "cancel"}
              disabled={busy !== null}
              onClick={() => run("cancel")}
            >
              Cancel this event
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setConfirmingCancel(false)}>
              Keep it
            </Button>
          </div>
        </div>
      )}

      <dl className="mt-4 grid grid-cols-2 gap-x-4 gap-y-3 text-sm sm:grid-cols-4">
        <div>
          <dt className="text-xs text-foreground-muted">Starts</dt>
          <dd className="mt-0.5">{formatInTimezone(event.starts_at, event.timezone)}</dd>
        </div>
        <div>
          <dt className="text-xs text-foreground-muted">Ends</dt>
          <dd className="mt-0.5">{formatInTimezone(event.ends_at, event.timezone)}</dd>
        </div>
        <div>
          <dt className="text-xs text-foreground-muted">Timezone</dt>
          <dd className="mt-0.5">{event.timezone}</dd>
        </div>
        <div>
          <dt className="text-xs text-foreground-muted">Capacity</dt>
          <dd className="mt-0.5">{event.capacity ?? "Unlimited"}</dd>
        </div>
      </dl>

      {error && (
        <p className="mt-3 text-xs text-danger" role="alert">
          {error}
        </p>
      )}
    </li>
  );
}
