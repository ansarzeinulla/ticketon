# Schema draft

Working notes for the database, before the tables are final. The source of
truth is `db/init/02_schema.sql`; this file only records how we got there.

## Core entities (SRS section 6)

| Entity | Why it exists |
| --- | --- |
| `users` | one account per email; roles are a separate table so a person can be both an organizer and an attendee |
| `events` | owned by an organizer; has a lifecycle: draft, published, cancelled, completed |
| `ticket_types` | several tiers per event, each with its own price and capacity |
| `orders` / `order_items` | one checkout; money is `numeric(14,2)` in KZT, never a float |
| `tickets` | one per admission, carries the QR token |
| `check_in_records` | one row per scan, so a second entry is detectable |

## Decisions so far

- UUID primary keys (`gen_random_uuid()` from `pgcrypto`).
- Emails are `citext`, so `A@x.kz` and `a@x.kz` are the same account.
- Every table has `created_at` / `updated_at`; a trigger keeps `updated_at` honest.
- Status fields are PostgreSQL enums, not free text.

## Open questions

- Seat maps: separate `venues` / `seats` tables, or a JSON layout per event?
- Should cancelled orders keep their tickets for history, or delete them?
