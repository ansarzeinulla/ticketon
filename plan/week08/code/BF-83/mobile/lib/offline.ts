import * as Crypto from "expo-crypto";

import { api, ApiError } from "./api";
import {
  applySyncResults,
  findByHash,
  findByTicketID,
  pendingCheckIns,
  recordLocalCheckIn,
  type QueuedCheckIn,
} from "./offline-db";
import type { RosterEntry, SyncSummary } from "./types";

/**
 * Hash a scanned token the same way the server does when it builds the roster:
 * SHA-256, lower-case hex. The comparison the door makes is hash against hash,
 * so this has to match `encode(digest(qr_token,'sha256'),'hex')` byte for byte.
 */
export async function hashToken(token: string): Promise<string> {
  return Crypto.digestStringAsync(Crypto.CryptoDigestAlgorithm.SHA256, token, {
    encoding: Crypto.CryptoEncoding.HEX,
  });
}

/** The verdict the offline gate reaches, mirroring the online scanner's cases. */
export type OfflineOutcome =
  | { kind: "valid"; entry: RosterEntry }
  | { kind: "already_checked_in"; entry: RosterEntry }
  | { kind: "not_valid"; entry: RosterEntry; reason: string }
  | { kind: "unknown" };

/**
 * Decide a scanned token against the local roster, and admit it if it is good.
 *
 * The rules are exactly the door's rules, run offline:
 *   - a token that matches nothing is unknown (a forgery, or the wrong event);
 *   - a cancelled or refunded ticket is refused and named as such;
 *   - a ticket already marked checked-in on this device is a repeat;
 *   - anything else is admitted, marked locally, and queued for sync.
 */
export async function verifyOffline(
  eventID: string,
  token: string,
  deviceLabel: string,
): Promise<OfflineOutcome> {
  const entry = await findByHash(eventID, await hashToken(token));
  return decide(eventID, entry, deviceLabel);
}

/** The same decision for somebody found by name rather than by camera. */
export async function admitByTicketID(
  eventID: string,
  ticketID: string,
  deviceLabel: string,
): Promise<OfflineOutcome> {
  const entry = await findByTicketID(eventID, ticketID);
  return decide(eventID, entry, deviceLabel);
}

async function decide(
  eventID: string,
  entry: RosterEntry | null,
  deviceLabel: string,
): Promise<OfflineOutcome> {
  if (!entry) return { kind: "unknown" };

  if (entry.status === "cancelled" || entry.status === "refunded") {
    return { kind: "not_valid", entry, reason: entry.status };
  }
  if (entry.status === "checked_in") {
    return { kind: "already_checked_in", entry };
  }

  const scannedAt = new Date().toISOString();
  await recordLocalCheckIn(eventID, entry.ticket_id, scannedAt, deviceLabel);
  return { kind: "valid", entry: { ...entry, status: "checked_in" } };
}

/**
 * Push every queued admission to the server and reconcile the local store with
 * the verdicts (SRS 4.8, "synchronize check-in records with the central
 * platform").
 *
 * Returns null when there was nothing to sync. Throws only when the network
 * itself fails - a per-ticket refusal is data in the summary, not an error,
 * because the twenty good admissions in the same batch still have to land.
 */
export async function syncPending(
  eventID: string,
  deviceLabel: string,
): Promise<SyncSummary | null> {
  const pending: QueuedCheckIn[] = await pendingCheckIns(eventID);
  if (pending.length === 0) return null;

  const summary = await api.syncCheckIns(
    eventID,
    pending.map((p) => ({
      ticket_id: p.ticket_id,
      scanned_at: p.scanned_at,
      device_label: p.device_label || deviceLabel,
    })),
  );

  // Every ticket the server ruled on leaves the queue; where it disagreed with
  // the device, its status overwrites ours so the roster tells the truth.
  await applySyncResults(
    eventID,
    summary.results.map((r) => ({ ticket_id: r.ticket_id, status: statusFor(r.outcome) })),
  );
  return summary;
}

/** Turn a sync outcome into the roster status it implies. */
function statusFor(outcome: string): string {
  switch (outcome) {
    case "recorded":
    case "already_checked_in":
      return "checked_in";
    case "not_valid":
      // The server voided it after the device went offline; stop showing it as
      // a good admission so the next scan of it is refused here too.
      return "refunded";
    default:
      // unknown_ticket: nothing sensible to record locally, leave as-is.
      return "checked_in";
  }
}

/** True when the failure is a lost network rather than a rejected request. */
export function isOffline(error: unknown): boolean {
  return error instanceof ApiError && error.isNetworkError;
}
