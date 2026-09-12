"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";
import {
  ORGANIZER_STATUSES,
  STATUS_LABELS,
  STATUS_STYLES,
  categoryLabel,
  formatMessageTime,
} from "@/lib/support";
import type { SupportThread as Thread, SupportStatus } from "@/lib/types";

/**
 * How often an open thread re-fetches itself.
 *
 * Polling, not a socket: SRS 4.13 explicitly does not require real-time
 * delivery for the MVP, and a support reply is not something anyone waits on
 * for seconds. A resolved thread stops polling entirely.
 */
const POLL_INTERVAL_MS = 10_000;

export function SupportThreadView({
  caseID,
  onChanged,
  onClose,
}: {
  caseID: string;
  /** Lets the surrounding list refresh its summary when the thread moves. */
  onChanged?: (thread: Thread) => void;
  onClose?: () => void;
}) {
  const [thread, setThread] = useState<Thread | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [reply, setReply] = useState("");
  const [internalNote, setInternalNote] = useState(false);
  const [sending, setSending] = useState(false);
  const [busyStatus, setBusyStatus] = useState(false);

  // Kept in a ref so the polling effect does not restart on every message.
  const onChangedRef = useRef(onChanged);
  useEffect(() => {
    onChangedRef.current = onChanged;
  });

  const adopt = useCallback((next: Thread) => {
    setThread(next);
    setError(null);
    onChangedRef.current?.(next);
  }, []);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        adopt(await api.getSupportThread(caseID, signal));
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load the conversation.");
      } finally {
        setLoading(false);
      }
    },
    [caseID, adopt],
  );

  useEffect(() => {
    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  // Poll while the conversation is still live.
  const resolved = thread?.case.status === "resolved";
  useEffect(() => {
    if (resolved) return;

    const timer = setInterval(() => void load(), POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [resolved, load]);

  async function handleReply(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const body = reply.trim();
    if (!body) return;

    setSending(true);
    try {
      adopt(await api.replyToSupportCase(caseID, body, internalNote));
      setReply("");
      setInternalNote(false);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "The message could not be sent.");
    } finally {
      setSending(false);
    }
  }

  async function changeStatus(status: SupportStatus) {
    setBusyStatus(true);
    try {
      adopt(await api.setSupportCaseStatus(caseID, status));
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "The status could not be changed.");
    } finally {
      setBusyStatus(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface p-6 text-sm text-foreground-muted">
        <Spinner />
        Loading the conversation…
      </div>
    );
  }

  if (!thread) {
    return <Alert tone="error">{error ?? "This conversation could not be loaded."}</Alert>;
  }

  const { case: supportCase, messages, can_moderate: canModerate } = thread;

  return (
    <div className="space-y-4 rounded-xl border border-border-subtle bg-surface p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="font-semibold">{supportCase.subject}</h3>
            <span
              className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLES[supportCase.status]}`}
              data-testid="case-status"
            >
              {STATUS_LABELS[supportCase.status]}
            </span>
          </div>
          <p className="mt-1 text-xs text-foreground-muted">
            {supportCase.case_number} · {categoryLabel(supportCase.category)}
            {supportCase.order_number ? ` · order ${supportCase.order_number}` : ""}
            {canModerate ? ` · from ${supportCase.requester_name}` : ""}
          </p>
        </div>

        {onClose && (
          <Button size="sm" variant="ghost" onClick={onClose}>
            Close
          </Button>
        )}
      </div>

      {error && <Alert tone="error">{error}</Alert>}

      <ul className="space-y-3" data-testid="case-messages">
        {messages.map((message) => {
          const fromStaff = message.sender_role === "staff";
          return (
            <li
              key={message.id}
              className={`rounded-lg border p-3 ${
                message.is_internal_note
                  ? "border-warning/40 bg-warning-soft"
                  : fromStaff
                    ? "border-brand/30 bg-brand-soft/40"
                    : "border-border-subtle bg-surface-muted/40"
              }`}
            >
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="text-xs font-semibold">
                  {message.sender_name}
                  {message.is_internal_note && (
                    <span className="ml-2 font-normal text-warning">
                      internal note — not shown to the attendee
                    </span>
                  )}
                </span>
                <span className="text-xs text-foreground-muted">
                  {formatMessageTime(message.created_at)}
                </span>
              </div>
              <p className="mt-1 whitespace-pre-line text-sm">{message.body}</p>
            </li>
          );
        })}
      </ul>

      {canModerate && (
        <div className="flex flex-wrap items-center gap-2 border-t border-border-subtle pt-4">
          <span className="text-xs text-foreground-muted">Set status:</span>
          {ORGANIZER_STATUSES.map((status) => (
            <Button
              key={status}
              size="sm"
              variant={supportCase.status === status ? "primary" : "secondary"}
              disabled={busyStatus || supportCase.status === status}
              onClick={() => void changeStatus(status)}
              data-testid={`set-status-${status}`}
            >
              {STATUS_LABELS[status]}
            </Button>
          ))}
        </div>
      )}

      <form onSubmit={handleReply} className="space-y-2 border-t border-border-subtle pt-4">
        <label htmlFor={`reply-${caseID}`} className="text-sm font-medium">
          {canModerate ? "Reply to the attendee" : "Add a message"}
        </label>
        <textarea
          id={`reply-${caseID}`}
          rows={3}
          value={reply}
          onChange={(event) => setReply(event.target.value)}
          placeholder={canModerate ? "Parking is in Zone B" : "Describe the problem…"}
          disabled={sending}
          className="w-full resize-y rounded-lg border border-border-subtle bg-surface px-3 py-2 text-sm placeholder:text-foreground-muted/70"
          data-testid="reply-input"
        />

        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit" loading={sending} disabled={!reply.trim()} data-testid="reply-send">
            Send
          </Button>

          {canModerate && (
            <label className="flex items-center gap-2 text-xs text-foreground-muted">
              <input
                type="checkbox"
                checked={internalNote}
                onChange={(event) => setInternalNote(event.target.checked)}
                disabled={sending}
              />
              Internal note (the attendee will not see this)
            </label>
          )}
        </div>
      </form>
    </div>
  );
}
