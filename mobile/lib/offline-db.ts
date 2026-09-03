import * as SQLite from "expo-sqlite";

import type { Roster, RosterEntry } from "./types";

/**
 * The device-local store behind offline check-in (SRS 4.8).
 *
 * Two tables, and the split matters:
 *
 *   - `roster` is the downloaded guest list. It holds the SHA-256 of each
 *     admission token, never the token, so a phone left on a table at the door
 *     cannot be turned into a ticket forge. It is disposable: it can be
 *     re-downloaded any time there is a network.
 *
 *   - `queue` is the admissions this device made while offline. It is *not*
 *     disposable - until it has been synced it is the only record that those
 *     people were let in - so re-downloading the roster must never touch it.
 *
 * Everything is scoped by event id, because one shared scanner works several
 * doors across a season.
 */

let dbPromise: Promise<SQLite.SQLiteDatabase> | null = null;

async function getDB(): Promise<SQLite.SQLiteDatabase> {
  if (!dbPromise) {
    dbPromise = (async () => {
      const db = await SQLite.openDatabaseAsync("biletflow-offline.db");
      // WAL keeps writes from blocking the reads the scanner does on every
      // frame; foreign_keys off because the two tables are deliberately
      // independent - the queue outlives any roster it was built against.
      await db.execAsync(`
        PRAGMA journal_mode = WAL;
        CREATE TABLE IF NOT EXISTS roster (
          event_id         TEXT NOT NULL,
          ticket_id        TEXT NOT NULL,
          token_hash       TEXT NOT NULL,
          ticket_code      TEXT NOT NULL,
          attendee_name    TEXT NOT NULL,
          ticket_type_name TEXT NOT NULL,
          seat_label       TEXT NOT NULL DEFAULT '',
          status           TEXT NOT NULL,
          PRIMARY KEY (event_id, ticket_id)
        );
        CREATE INDEX IF NOT EXISTS roster_hash_idx ON roster (event_id, token_hash);
        CREATE TABLE IF NOT EXISTS queue (
          event_id     TEXT NOT NULL,
          ticket_id    TEXT NOT NULL,
          scanned_at   TEXT NOT NULL,
          device_label TEXT NOT NULL,
          PRIMARY KEY (event_id, ticket_id)
        );
      `);
      return db;
    })();
  }
  return dbPromise;
}

/**
 * Replace an event's roster with a freshly downloaded one.
 *
 * The queue is left untouched, and any ticket that still has a pending
 * admission queued on this device is re-marked checked-in in the new roster.
 * Without that, a re-download partway through a shift would make already-
 * admitted people look valid again and let them in twice.
 */
export async function saveRoster(roster: Roster): Promise<void> {
  const db = await getDB();
  await db.withTransactionAsync(async () => {
    await db.runAsync(`DELETE FROM roster WHERE event_id = ?`, roster.event_id);
    for (const t of roster.tickets) {
      await db.runAsync(
        `INSERT INTO roster
           (event_id, ticket_id, token_hash, ticket_code, attendee_name,
            ticket_type_name, seat_label, status)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
        roster.event_id,
        t.ticket_id,
        t.token_hash,
        t.ticket_code,
        t.attendee_name,
        t.ticket_type_name,
        t.seat_label ?? "",
        t.status,
      );
    }
    // Re-apply this device's own un-synced admissions over the fresh statuses.
    await db.runAsync(
      `UPDATE roster SET status = 'checked_in'
        WHERE event_id = ?1
          AND ticket_id IN (SELECT ticket_id FROM queue WHERE event_id = ?1)`,
      roster.event_id,
    );
  });
}

/** How many tickets the roster holds and how many are marked admitted. */
export async function rosterStats(
  eventID: string,
): Promise<{ total: number; checkedIn: number; hasRoster: boolean }> {
  const db = await getDB();
  const row = await db.getFirstAsync<{ total: number; checked_in: number }>(
    `SELECT count(*) AS total,
            sum(CASE WHEN status = 'checked_in' THEN 1 ELSE 0 END) AS checked_in
       FROM roster WHERE event_id = ?`,
    eventID,
  );
  const total = row?.total ?? 0;
  return { total, checkedIn: row?.checked_in ?? 0, hasRoster: total > 0 };
}

/** Look one ticket up by the hash of its scanned token. */
export async function findByHash(
  eventID: string,
  tokenHash: string,
): Promise<RosterEntry | null> {
  const db = await getDB();
  const row = await db.getFirstAsync<RosterEntry>(
    `SELECT ticket_id, token_hash, ticket_code, attendee_name,
            ticket_type_name, seat_label, status
       FROM roster WHERE event_id = ? AND token_hash = ?`,
    eventID,
    tokenHash,
  );
  return row ?? null;
}

/** Look one ticket up by its id, for admitting somebody found by name. */
export async function findByTicketID(
  eventID: string,
  ticketID: string,
): Promise<RosterEntry | null> {
  const db = await getDB();
  const row = await db.getFirstAsync<RosterEntry>(
    `SELECT ticket_id, token_hash, ticket_code, attendee_name,
            ticket_type_name, seat_label, status
       FROM roster WHERE event_id = ? AND ticket_id = ?`,
    eventID,
    ticketID,
  );
  return row ?? null;
}

/**
 * Record an offline admission: mark the roster row and queue it for sync.
 *
 * Both writes are one transaction and both are idempotent on (event, ticket),
 * so a double scan of the same ticket at the same door changes nothing and
 * queues nothing new - the local `already checked in` answer comes from the
 * roster status this sets.
 */
export async function recordLocalCheckIn(
  eventID: string,
  ticketID: string,
  scannedAt: string,
  deviceLabel: string,
): Promise<void> {
  const db = await getDB();
  await db.withTransactionAsync(async () => {
    await db.runAsync(
      `UPDATE roster SET status = 'checked_in' WHERE event_id = ? AND ticket_id = ?`,
      eventID,
      ticketID,
    );
    await db.runAsync(
      `INSERT OR IGNORE INTO queue (event_id, ticket_id, scanned_at, device_label)
       VALUES (?, ?, ?, ?)`,
      eventID,
      ticketID,
      scannedAt,
      deviceLabel,
    );
  });
}

/**
 * Undo an admission this device made offline (SRS 4.8: "undo an accidental
 * check-in where authorized").
 *
 * Both the queued record and the local status go back, so the person is off
 * the list again and the next scan of that ticket admits them properly. Only
 * an un-synced admission can be undone this way - once the server has it, the
 * reversal has to go through the online endpoint.
 */
export async function undoLocalCheckIn(eventID: string, ticketID: string): Promise<void> {
  const db = await getDB();
  await db.withTransactionAsync(async () => {
    await db.runAsync(
      `DELETE FROM queue WHERE event_id = ? AND ticket_id = ?`,
      eventID,
      ticketID,
    );
    await db.runAsync(
      `UPDATE roster SET status = 'valid' WHERE event_id = ? AND ticket_id = ?`,
      eventID,
      ticketID,
    );
  });
}

export interface QueuedCheckIn {
  ticket_id: string;
  scanned_at: string;
  device_label: string;
}

/** The admissions still waiting to reach the server. */
export async function pendingCheckIns(eventID: string): Promise<QueuedCheckIn[]> {
  const db = await getDB();
  return db.getAllAsync<QueuedCheckIn>(
    `SELECT ticket_id, scanned_at, device_label
       FROM queue WHERE event_id = ? ORDER BY scanned_at`,
    eventID,
  );
}

export async function pendingCount(eventID: string): Promise<number> {
  const db = await getDB();
  const row = await db.getFirstAsync<{ n: number }>(
    `SELECT count(*) AS n FROM queue WHERE event_id = ?`,
    eventID,
  );
  return row?.n ?? 0;
}

/**
 * Drop admissions the server has now accounted for, and correct the roster
 * where the server disagreed.
 *
 * `reconciled` is every ticket the server returned a verdict on - recorded,
 * already-checked-in, unknown or refused. All of them leave the queue: the
 * server has the last word, and a ticket it called `not_valid` must stop
 * showing as admitted here too, so the next scan tells staff the truth.
 */
export async function applySyncResults(
  eventID: string,
  reconciled: { ticket_id: string; status: string }[],
): Promise<void> {
  if (reconciled.length === 0) return;
  const db = await getDB();
  await db.withTransactionAsync(async () => {
    for (const r of reconciled) {
      await db.runAsync(
        `DELETE FROM queue WHERE event_id = ? AND ticket_id = ?`,
        eventID,
        r.ticket_id,
      );
      await db.runAsync(
        `UPDATE roster SET status = ? WHERE event_id = ? AND ticket_id = ?`,
        r.status,
        eventID,
        r.ticket_id,
      );
    }
  });
}
