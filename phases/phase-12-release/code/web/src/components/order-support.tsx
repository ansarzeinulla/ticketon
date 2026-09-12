"use client";

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";

import { SupportThreadView } from "@/components/support-thread";
import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { STATUS_LABELS, STATUS_STYLES, categoryLabel } from "@/lib/support";
import type { SupportCase } from "@/lib/types";

/**
 * The support panel on an attendee's order confirmation.
 *
 * Opening a case needs an account: support_cases.requester_user_id is NOT NULL,
 * and a conversation needs somewhere to come back to. A guest who registers
 * with the address they bought under is matched to their order by email, so
 * they do not lose access to it.
 */
export function OrderSupport({
  orderID,
  buyerEmail,
}: {
  orderID: string;
  buyerEmail: string;
}) {
  const { status: authStatus } = useAuth();

  const [cases, setCases] = useState<SupportCase[]>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [openCaseID, setOpenCaseID] = useState<string | null>(null);
  const [composing, setComposing] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [category, setCategory] = useState("event_information");
  const [subject, setSubject] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const signedIn = authStatus === "authenticated";

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const [list, cats] = await Promise.all([
          signedIn ? api.mySupportCases(signal) : Promise.resolve([]),
          api.supportCategories(signal),
        ]);
        setCases(list.filter((item) => item.order_id === orderID));
        setCategories(cats);
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        if (cause instanceof ApiError && cause.status === 401) {
          setCases([]);
          return;
        }
        setError(cause instanceof ApiError ? cause.message : "Could not load your support cases.");
      } finally {
        setLoading(false);
      }
    },
    [orderID, signedIn],
  );

  useEffect(() => {
    if (authStatus === "loading") return;

    const controller = new AbortController();
    // Every setState in `load` happens after its await; the rule cannot see that.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal);
    return () => controller.abort();
  }, [authStatus, load]);

  async function handleOpen(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFieldErrors({});
    setError(null);

    const errors: Record<string, string> = {};
    if (!subject.trim()) errors.subject = "Give your question a short subject.";
    if (!message.trim()) errors.message = "Describe the problem.";
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setSubmitting(true);
    try {
      const thread = await api.openSupportCase({
        category,
        subject: subject.trim(),
        message: message.trim(),
        order_id: orderID,
      });
      setCases((current) => [thread.case, ...current]);
      setOpenCaseID(thread.case.id);
      setComposing(false);
      setSubject("");
      setMessage("");
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        setError(
          Object.keys(cause.fields).length > 0
            ? "Please correct the highlighted fields."
            : cause.message,
        );
      } else {
        setError("The case could not be opened.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (authStatus === "loading" || loading) {
    return (
      <section className="mt-8">
        <h2 className="text-sm font-semibold">Need help?</h2>
        <div className="mt-2 flex items-center gap-3 text-sm text-foreground-muted">
          <Spinner />
          Loading…
        </div>
      </section>
    );
  }

  // A guest buyer has no account to hold a conversation against.
  if (!signedIn) {
    return (
      <section className="mt-8 rounded-xl border border-border-subtle bg-surface p-5">
        <h2 className="text-sm font-semibold">Need help with this order?</h2>
        <p className="mt-1 text-sm text-foreground-muted">
          Sign in with <span className="font-medium">{buyerEmail}</span> — the address
          this order was bought under — to open a support case with the organizer.
        </p>
        <Link
          href={`/login?next=/orders/${orderID}`}
          className="mt-4 inline-flex rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-strong"
        >
          Sign in to get help
        </Link>
      </section>
    );
  }

  if (openCaseID) {
    return (
      <section className="mt-8 space-y-3">
        <h2 className="text-sm font-semibold">Support</h2>
        <SupportThreadView
          caseID={openCaseID}
          onClose={() => setOpenCaseID(null)}
          onChanged={(thread) =>
            setCases((current) =>
              current.map((item) => (item.id === thread.case.id ? thread.case : item)),
            )
          }
        />
      </section>
    );
  }

  return (
    <section className="mt-8 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-sm font-semibold">Support</h2>
        {!composing && (
          <Button size="sm" variant="secondary" onClick={() => setComposing(true)}>
            Contact the organizer
          </Button>
        )}
      </div>

      {error && <Alert tone="error">{error}</Alert>}

      {cases.length > 0 && (
        <ul className="space-y-2">
          {cases.map((item) => (
            <li key={item.id}>
              <button
                type="button"
                onClick={() => setOpenCaseID(item.id)}
                className="flex w-full flex-wrap items-center justify-between gap-3 rounded-lg border border-border-subtle bg-surface px-4 py-3 text-left hover:bg-surface-muted"
                data-testid="open-case"
              >
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium">{item.subject}</span>
                  <span className="text-xs text-foreground-muted">
                    {item.case_number} · {categoryLabel(item.category)} ·{" "}
                    {item.message_count} message{item.message_count === 1 ? "" : "s"}
                  </span>
                </span>
                <span
                  className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLES[item.status]}`}
                >
                  {STATUS_LABELS[item.status]}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {composing && (
        <form
          onSubmit={handleOpen}
          noValidate
          className="space-y-4 rounded-xl border border-border-subtle bg-surface p-5"
        >
          <h3 className="text-sm font-semibold">Ask the organizer</h3>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <label htmlFor="support-category" className="block text-sm font-medium">
                What is it about?
              </label>
              <select
                id="support-category"
                value={category}
                onChange={(event) => setCategory(event.target.value)}
                disabled={submitting}
                className="w-full rounded-lg border border-border-subtle bg-surface px-3 py-2 text-sm"
                data-testid="support-category"
              >
                {categories.map((option) => (
                  <option key={option} value={option}>
                    {categoryLabel(option)}
                  </option>
                ))}
              </select>
            </div>

            <TextField
              label="Subject"
              name="subject"
              placeholder="Parking"
              required
              value={subject}
              error={fieldErrors.subject}
              disabled={submitting}
              onChange={(event) => setSubject(event.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="support-message" className="block text-sm font-medium">
              Your question
            </label>
            <textarea
              id="support-message"
              rows={3}
              value={message}
              onChange={(event) => setMessage(event.target.value)}
              placeholder="Where is parking?"
              disabled={submitting}
              className="w-full resize-y rounded-lg border border-border-subtle bg-surface px-3 py-2 text-sm placeholder:text-foreground-muted/70"
              data-testid="support-message"
            />
            {fieldErrors.message && (
              <p className="text-xs text-danger">{fieldErrors.message}</p>
            )}
          </div>

          <div className="flex gap-3">
            <Button type="submit" loading={submitting} data-testid="support-submit">
              Send to the organizer
            </Button>
            <Button
              type="button"
              variant="secondary"
              disabled={submitting}
              onClick={() => setComposing(false)}
            >
              Cancel
            </Button>
          </div>
        </form>
      )}

      {!composing && cases.length === 0 && (
        <p className="rounded-xl border border-dashed border-border-subtle bg-surface p-6 text-center text-sm text-foreground-muted">
          Something wrong with this order? Message the organizer and they will reply here.
        </p>
      )}
    </section>
  );
}
