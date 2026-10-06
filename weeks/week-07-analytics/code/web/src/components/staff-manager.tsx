"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";

import { Alert } from "@/components/ui/alert";
import { Button, Spinner } from "@/components/ui/button";
import { SelectField, TextField } from "@/components/ui/field";
import { ApiError, api } from "@/lib/api";
import type { StaffMember } from "@/lib/types";

const ROLES = ["event_admin", "support_staff", "manager"] as const;

const ROLE_LABELS: Record<string, string> = {
  event_admin: "Event admin — scans tickets at the gate",
  support_staff: "Support staff — answers support cases",
  manager: "Manager — full access to this event",
};

/**
 * Naming the people who work an event (SRS 4.2: "Assign event administrators
 * who verify tickets and check attendees in through a mobile app").
 *
 * The three staff endpoints have existed since Phase 6, but only the scanner
 * app consumed the result - there was no way for an organizer to grant the
 * access in the first place without hand-writing a cURL call. That made the
 * mobile app effectively unusable by anybody who was not already the organizer.
 */
export function StaffManager({ eventID }: { eventID: string }) {
  const [staff, setStaff] = useState<StaffMember[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [email, setEmail] = useState("");
  const [role, setRole] = useState<string>("event_admin");
  const [adding, setAdding] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [revoking, setRevoking] = useState<string | null>(null);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const data = await api.listStaff(eventID, signal);
        setStaff(data.staff);
        setError(null);
      } catch (cause) {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof ApiError ? cause.message : "Could not load the staff list.");
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

  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFieldErrors({});
    setError(null);

    if (!email.trim()) {
      setFieldErrors({ email: "Enter the email address of an existing account." });
      return;
    }

    setAdding(true);
    try {
      await api.assignStaff(eventID, email.trim().toLowerCase(), role);
      setEmail("");
      await load();
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fields);
        if (Object.keys(cause.fields).length === 0) setError(cause.message);
      } else {
        setError("Could not add this person.");
      }
    } finally {
      setAdding(false);
    }
  }

  async function revoke(member: StaffMember) {
    setRevoking(member.id);
    setError(null);
    try {
      await api.revokeStaff(eventID, member.id);
      await load();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "Could not revoke this access.");
    } finally {
      setRevoking(null);
    }
  }

  const active = staff?.filter((member) => !member.revoked_at) ?? [];

  return (
    <section className="rounded-xl border border-border-subtle bg-surface p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-base font-semibold">Gate staff</h2>
        <p className="text-xs text-foreground-muted">
          Who can scan tickets for this event in the BiletFlow Scanner app.
        </p>
      </div>

      {error && (
        <div className="mt-3">
          <Alert>{error}</Alert>
        </div>
      )}

      <div className="mt-4">
        {loading ? (
          <p className="flex items-center gap-2 text-sm text-foreground-muted">
            <Spinner aria-hidden /> Loading…
          </p>
        ) : active.length === 0 ? (
          <p className="text-sm text-foreground-muted">
            Nobody yet. You can always scan your own event; add somebody here to let
            them work the door without your account.
          </p>
        ) : (
          <ul className="divide-y divide-border-subtle" data-testid="staff-list">
            {active.map((member) => (
              <li
                key={member.id}
                className="flex flex-wrap items-center justify-between gap-3 py-3"
              >
                <div className="min-w-0">
                  <div className="font-medium">{member.user_name}</div>
                  <p className="text-xs text-foreground-muted">{member.user_email}</p>
                  <p className="text-xs text-foreground-muted">
                    {ROLE_LABELS[member.role] ?? member.role}
                  </p>
                </div>
                <Button
                  size="sm"
                  variant="secondary"
                  loading={revoking === member.id}
                  disabled={revoking !== null}
                  onClick={() => revoke(member)}
                  data-testid={`revoke-${member.id}`}
                >
                  Revoke
                </Button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <form onSubmit={add} noValidate className="mt-5 space-y-4 border-t border-border-subtle pt-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField
            label="Email"
            type="email"
            name="staff_email"
            placeholder="scanner@biletflow.kz"
            hint="They need a BiletFlow account already."
            value={email}
            error={fieldErrors.email}
            disabled={adding}
            onChange={(event) => setEmail(event.target.value)}
          />
          <SelectField
            label="Role"
            name="staff_role"
            options={ROLES}
            value={role}
            error={fieldErrors.role}
            disabled={adding}
            onChange={(event) => setRole(event.target.value)}
          />
        </div>
        <Button type="submit" size="sm" loading={adding} data-testid="add-staff">
          Add to this event
        </Button>
      </form>
    </section>
  );
}
