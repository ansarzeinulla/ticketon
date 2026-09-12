"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { ModerationAction } from "@/components/moderation-action";
import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { EventReport, ReportStatus } from "@/lib/types";

const FILTERS: { value: string; label: string }[] = [
  { value: "open", label: "Open" },
  { value: "reviewing", label: "Reviewing" },
  { value: "upheld", label: "Upheld" },
  { value: "dismissed", label: "Dismissed" },
  { value: "", label: "Everything" },
];

const REASON_LABELS: Record<string, string> = {
  fraud: "Fraud",
  misleading: "Misleading",
  inappropriate: "Inappropriate",
  spam: "Spam",
  copyright: "Copyright",
  other: "Other",
};

const STATUS_TONES: Record<ReportStatus, string> = {
  open: "bg-warning-soft text-warning",
  reviewing: "bg-brand/10 text-brand",
  upheld: "bg-danger-soft text-danger",
  dismissed: "bg-surface-muted text-foreground-muted",
};

/**
 * The moderation queue (SRS 4.12: "Review reported events").
 *
 * Deciding a report and suspending an event are deliberately two actions.
 * Upholding a complaint says "this was justified"; whether that warrants
 * taking the event down is a second, explicit choice — one report of
 * "misleading" against a hundred happy attendees is not the same call as a
 * fraud complaint, and collapsing the two would make the queue a trigger
 * rather than a review.
 */
export function ReportsQueue() {
  const { user } = useAuth();
  const isAdmin = user?.roles.includes("platform_admin") ?? false;

  const [status, setStatus] = useState("open");
  const [reports, setReports] = useState<EventReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (filter: string, signal?: AbortSignal) => {
      try {
        const data = await api.listEventReports(filter || undefined, signal);
        setReports(data.reports);
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load the queue.");
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    if (!isAdmin) return;
    const controller = new AbortController();
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(status, controller.signal);
    return () => controller.abort();
  }, [status, load, isAdmin]);

  if (!isAdmin) {
    return (
      <Alert tone="error" title="Not available">
        The moderation queue is restricted to platform administrators.
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <Link href="/admin" className="text-sm text-foreground-muted hover:underline">
            ← Back to administration
          </Link>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight">Reported events</h1>
          <p className="mt-1 text-sm text-foreground-muted">
            Complaints raised by attendees, newest first.
          </p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void load(status)}>
          Refresh
        </Button>
      </div>

      <div className="flex flex-wrap gap-2">
        {FILTERS.map((filter) => (
          <button
            key={filter.value || "all"}
            type="button"
            onClick={() => setStatus(filter.value)}
            aria-pressed={status === filter.value}
            data-testid={`report-filter-${filter.value || "all"}`}
            className={`rounded-full px-3 py-1.5 text-xs font-medium ${
              status === filter.value
                ? "bg-brand text-white"
                : "border border-border-subtle hover:bg-surface-muted"
            }`}
          >
            {filter.label}
          </button>
        ))}
      </div>

      {error && <Alert>{error}</Alert>}

      {loading ? (
        <p className="flex items-center gap-2 text-sm text-foreground-muted">
          <Spinner aria-hidden /> Loading…
        </p>
      ) : reports.length === 0 ? (
        <p
          className="rounded-xl border border-dashed border-border-subtle px-6 py-12 text-center text-sm text-foreground-muted"
          data-testid="queue-empty"
        >
          Nothing here. {status === "open" ? "No open complaints." : "No reports match."}
        </p>
      ) : (
        <ul className="space-y-4" data-testid="report-queue">
          {reports.map((report) => (
            <li
              key={report.id}
              className="rounded-xl border border-border-subtle bg-surface p-5"
              data-testid={`report-${report.id}`}
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <Link
                      href={`/events/${report.event_slug}`}
                      className="font-semibold text-brand hover:underline"
                    >
                      {report.event_title}
                    </Link>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_TONES[report.status]}`}
                    >
                      {report.status}
                    </span>
                    <span className="rounded-full bg-surface-muted px-2 py-0.5 text-xs">
                      {REASON_LABELS[report.reason] ?? report.reason}
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-foreground-muted">
                    Organizer {report.organizer_email}
                    {report.reporter_email && ` · reported by ${report.reporter_email}`}
                  </p>
                </div>
                <time className="text-xs text-foreground-muted">
                  {new Date(report.created_at).toLocaleDateString()}
                </time>
              </div>

              {report.details && (
                <blockquote className="mt-3 border-l-2 border-border-subtle pl-3 text-sm text-foreground-muted">
                  {report.details}
                </blockquote>
              )}

              {report.resolution && (
                <p className="mt-3 text-sm">
                  <span className="text-foreground-muted">Resolution: </span>
                  {report.resolution}
                </p>
              )}

              {(report.status === "open" || report.status === "reviewing") && (
                <div className="mt-4 flex flex-wrap gap-2">
                  <ModerationAction
                    label="Uphold"
                    confirmLabel="Uphold this report"
                    consequence="Records that the complaint was justified. Suspending the event is a separate decision, taken from the administration page."
                    onConfirm={(reason) =>
                      api.reviewEventReport(report.id, "upheld", reason)
                    }
                    onDone={() => void load(status)}
                  />
                  <ModerationAction
                    label="Dismiss"
                    confirmLabel="Dismiss this report"
                    consequence="Records that the complaint was not justified. The event is untouched and the reporter is not told."
                    tone="secondary"
                    onConfirm={(reason) =>
                      api.reviewEventReport(report.id, "dismissed", reason)
                    }
                    onDone={() => void load(status)}
                  />
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
