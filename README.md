# BiletFlow

Self-service event ticketing platform for Kazakhstan, built to the specification
in [bilet.md](bilet.md).

| Phase | Scope | Status |
| --- | --- | --- |
| 1 | PostgreSQL schema + Docker environment | done |
| 2 | Go REST API: auth and event management — see [api/README.md](api/README.md) | done |
| 3 | Next.js organizer dashboard — see [web/README.md](web/README.md) | done |
| 4 | Ticket types + simulated KZT checkout | done |
| 5 | QR codes + printable A4 PDF tickets | done |
| 6 | React Native scanner app — see [mobile/README.md](mobile/README.md) | done |
| 7 | Promo codes + campaign QR codes | done |
| 8 | Support chat + platform moderation | done |

Everything at once, in containers:

```bash
docker compose --profile app up -d     # PostgreSQL + API + dashboard
```

→ dashboard on <http://localhost:3000>, API on <http://localhost:8080>.

Or layer by layer, for development:

```bash
make up        # 1. PostgreSQL with the schema applied
make api-run   # 2. Go API      → http://localhost:8080
make web-dev   # 3. Dashboard   → http://localhost:3000
make scan-dev  # 4. Scanner app → Expo (press i / a, or scan the QR)
```

```bash
make verify    # every offline suite in one command
```

Or individually:

```bash
make test       # 147 database assertions
make api-check  # gofmt, vet, and the Go unit + integration suite
make api-smoke  # 168 cURL acceptance checks (needs the API running)
make web-check  # frontend lint, typecheck and 32 unit tests
make web-e2e    # 31 browser end-to-end specs (needs the API and dashboard running)
make scan-check # scanner app typecheck
```

What each suite covers, and which SRS requirement it proves, is in
[docs/verification/traceability.md](docs/verification/traceability.md); the
measured results are in [docs/verification/RESULTS.md](docs/verification/RESULTS.md).

---

## Phase 1: Database & Docker Foundation

---

## Quick start

```bash
make up && make test
```

That copies `.env.example` to `.env` if needed, starts PostgreSQL, applies the
schema, waits for the health check, and runs 144 assertions against the result.

Individual steps, if you prefer them explicit:

```bash
cp .env.example .env
docker compose up -d
./db/tests/run_tests.sh
```

| Command | What it does |
| --- | --- |
| `make up` | Start PostgreSQL; the schema is applied on first start |
| `make test` | Run the database test suite |
| `make seed` | Load the demo dataset (re-runnable) |
| `make psql` | Interactive `psql` shell |
| `make tables` | List tables with approximate row counts |
| `make reset` | Destroy the volume and rebuild from `db/init` |
| `make pgadmin` | Start pgAdmin at <http://localhost:5050> |
| `make down` | Stop containers, keep the data |
| `make api-run` | Run the Phase 2 API (see [api/README.md](api/README.md)) |
| `make api-test` | Run the Go test suite |
| `make api-smoke` | cURL acceptance checks against a running API |
| `make web-dev` | Run the organizer dashboard (see [web/README.md](web/README.md)) |
| `make web-build` | Production build of the dashboard |

---

## Connecting with DBeaver / pgAdmin / TablePlus

| Setting | Value |
| --- | --- |
| Host | `localhost` |
| Port | `5433` (the container listens on 5432; 5433 avoids clashing with a locally installed PostgreSQL) |
| Database | `biletflow` |
| User | `biletflow` |
| Password | `biletflow_dev_password` |

Connection string:

```
postgresql://biletflow:biletflow_dev_password@localhost:5433/biletflow
```

Change any of these in `.env` and run `make reset`.

pgAdmin is optional and runs under a compose profile, so `docker compose up`
starts only the database:

```bash
make pgadmin
```

---

## Verifying the phase-1 success criteria by hand

**1. The database starts without errors**

```bash
docker compose up -d && docker compose logs db | tail -20
```

The last lines read `database system is ready to accept connections`, with
`running /docker-entrypoint-initdb.d/01_extensions.sql` and `02_schema.sql`
above them and no `ERROR` lines.

**2. The SQL script creates all tables**

The scripts in `db/init/` run automatically on first start. They are also
**idempotent**, so you can paste `db/init/01_extensions.sql` and
`db/init/02_schema.sql` into DBeaver or pgAdmin and execute them by hand — on an
already-initialised database they complete without error and change nothing.
Then:

```sql
\dt
```

28 tables are listed. `make schema` re-runs both scripts from the command line.

**3. Insert and select a user**

```sql
INSERT INTO users (email, password_hash, full_name, status)
VALUES ('test.user@biletflow.kz', 'not-a-real-hash', 'Test User', 'active');

SELECT id, email, full_name, status, locale, created_at
FROM users
WHERE email = 'test.user@biletflow.kz';
```

`id`, `locale`, `created_at` and `updated_at` are filled in by the database.
Clean up with `DELETE FROM users WHERE email = 'test.user@biletflow.kz';`.

---

## Tests

```bash
make test                 # everything
./db/tests/run_tests.sh 04   # only files starting with 04
VERBOSE=1 make test        # print every assertion
```

Each file runs inside a transaction that is rolled back, so the suite never
changes the contents of your database and can be run against a seeded one. The
runner exits non-zero on any failure, so it can be dropped into CI later.

| File | Covers |
| --- | --- |
| `01_environment.sql` | Server version, extensions, UTC clock — *criterion 1* |
| `02_schema_objects.sql` | Every expected table, enum, index, trigger and primary key exists — *criterion 2* |
| `03_user_crud.sql` | Insert / select / update / delete a user, cascade to roles — *criterion 3* |
| `04_constraints.sql` | 33 invalid rows the database must reject, plus generated-column checks |
| `05_business_rules.sql` | No double check-in, no double-sold seat, no forged campaign QR, append-only audit log, redemption limits, activation checklist |
| `06_end_to_end_flow.sql` | Free registration → ticket → check-in → support case → refund → the analytics figures from SRS 4.15, plus event duplication |

The suite treats a file that produces zero assertions as a failure, so a test
that silently stops running cannot report a pass.

---

## Schema

28 tables covering every entity in SRS section 6, plus three tables the entity
list implies: `user_roles`, `campaign_ticket_types` and `paid_sales_activations`.

| Area | Tables |
| --- | --- |
| Accounts (4.1) | `users`, `user_roles`, `organizer_profiles`, `payout_accounts` |
| Venue & seating (4.3.1) | `venues`, `venue_sections`, `seat_rows`, `seats`, `seat_holds` |
| Events (4.2, 4.3) | `events`, `staff_assignments`, `ticket_types` |
| Commerce (4.4, 4.6, 4.7) | `orders`, `order_items`, `attendees`, `tickets` |
| Money (3.2, 4.5, 4.9) | `payments`, `refunds`, `paid_sales_activations` |
| Campaigns (4.14) | `campaigns`, `campaign_ticket_types`, `promo_codes`, `promo_redemptions` |
| Check-in (4.8) | `check_in_records` |
| Support (4.13) | `support_cases`, `support_messages` |
| Platform (4.10, 4.16) | `notifications`, `audit_logs` |

### Rules enforced by the database, not just the application

These are the SRS requirements where an application-layer-only check would be a
real risk, so they are also constraints in PostgreSQL:

- **A ticket is admitted once.** A partial unique index on
  `check_in_records (ticket_id) WHERE reversed_at IS NULL` makes a second
  check-in impossible; an authorised reversal releases it again (4.8).
- **A seat is sold once.** A partial unique index on
  `tickets (seat_id) WHERE status IN ('valid','checked_in')` blocks two
  concurrent orders from taking the same seat, and a partial unique index on
  `seat_holds (seat_id) WHERE status = 'active'` does the same for holds (4.3.1).
- **A campaign QR is never an admission QR.** Ticket tokens must match `TKT_…`
  and campaign tokens `CMP_…`, so the two namespaces cannot overlap and a
  campaign code can never be stored as, or scanned as, a ticket (4.14).
- **The order total is arithmetic, not a client value.**
  `total_kzt = subtotal - discount + processing_fee` is a check constraint, as
  is `line_total = unit_price × quantity − discount` on each item (4.6).
- **Inventory cannot be oversold.** `quantity_sold + quantity_reserved <=
  quantity_total` on `ticket_types` (4.3).
- **Redemptions cannot exceed the campaign limit.** `redemption_count <=
  max_redemptions`, plus one redemption per order (4.14).
- **Paid sales cannot go live on an incomplete checklist.** An activation row
  may only reach `active` with identity, payout, terms and fee payment all
  recorded (4.5).
- **The audit trail is append-only.** A trigger rejects `UPDATE` and `DELETE` on
  `audit_logs` (4.16). It deliberately carries no foreign keys, since a
  cascading delete would otherwise be blocked by that same trigger.

### Conventions

- `uuid` primary keys via `gen_random_uuid()`; `audit_logs` uses a bigint identity.
- All timestamps are `timestamptz`; the server runs in UTC and each event stores
  its own IANA `timezone` so calendar exports keep the configured zone (4.11).
- Money is `numeric(14,2)` and currency is pinned to `KZT` by check constraint.
- Emails, event slugs and promo codes use `citext`, so they are unique
  regardless of letter case.
- `updated_at` is maintained by trigger, not by the application.
- Payments and refunds carry `is_simulated`, defaulting to true — the MVP moves
  no real money (4.6).
- Only password *hashes* are stored, and payout accounts keep an opaque provider
  reference plus a masked display value, never card data (section 7).

---

## Demo data

```bash
make seed
```

Loads 7 users, a 126-seat predefined venue layout, three events (one free, one
paid assigned-seating with an active campaign, one completed), orders, tickets,
a check-in, a refund, a support thread and an activity timeline. It clears its
own rows first, so running it repeatedly does not duplicate anything.

### Demo accounts

Every seeded account uses the password **`biletflow-demo`**.

| Email | Name | Roles |
| --- | --- | --- |
| `dana@biletflow.kz` | Dana Amirova | organizer, attendee |
| `timur@biletflow.kz` | Timur Bekov | organizer |
| `nurlan@example.kz` | Nurlan Sagyndyk | attendee |
| `aigerim@example.kz` | Aigerim Zhaksy | attendee |
| `olzhas@example.kz` | Olzhas Serik | attendee |
| `scanner@biletflow.kz` | Askar Kassym | event_admin — signs into the scanner app |
| `support@biletflow.kz` | Sofia Ivanova | support_staff, platform_admin — sees `/admin` |

These are for a local demonstration database and nothing else. The password is
published here on purpose: it is what makes the dataset usable.

Ticket QR codes you can type into the scanner's "Enter code by hand":

| Token | What it demonstrates |
| --- | --- |
| `TKT_DEMOSEAT00000000A1` | A valid admission — green screen |
| `TKT_DEMOFREE0000000001` | Already checked in — red, with the first-use time |
| `TKT_DEMOSEAT00000000F1` | A refunded ticket — red, refused |
| `CMP_STUDENT15ALMATYJAZZ` | A campaign QR — refused as "promo code, not a ticket" |

---

## Layout

```
docker-compose.yml         PostgreSQL 17 + optional pgAdmin
Makefile                   task shortcuts
.env.example               credentials, ports and API settings
db/init/01_extensions.sql  citext, pgcrypto, btree_gist
db/init/02_schema.sql      the full schema (idempotent)
db/seed/01_demo_data.sql   demo dataset
db/tests/run_tests.sh      test runner
db/tests/_helpers.sql      assertion helpers
db/tests/_fixture.sql      deterministic test dataset
db/tests/0*.sql            the test files
api/                       Phase 2 Go REST API
web/                       Phase 3 Next.js organizer dashboard
mobile/                    Phase 6 Expo ticket scanner
docs/                      Postman collection
```

---

## Decisions taken in this phase

- **PostgreSQL** over MongoDB (SRS 9 leaves this open). Ticketing is relational
  and the hard requirements are transactional: atomic seat reservation, no
  double admission, server-side redemption limits and an append-only audit
  trail. Those are check constraints and partial unique indexes here.
- **Money as `numeric(14,2)`**, not floating point.
- **Roles as a set** (`user_roles`) rather than one column, because a person is
  routinely both an attendee and an organizer.
- **Venue snapshots on the event** (`venue_name`, `venue_address`) and seat
  snapshots on the ticket, so a historical ticket still prints correctly after
  the venue record changes.

Still open, per SRS 12: the backend language, the activation fee amount, and
whether attendees must create accounts (`orders.buyer_user_id` is nullable so
guest checkout stays possible).


---

## Phase 2: Core API

Go REST API for registration, JWT login and event CRUD, on top of the schema
above. Full documentation in **[api/README.md](api/README.md)**; a Postman
collection is in [docs/](docs/biletflow-api.postman_collection.json).

```bash
make up && make api-run
make api-test    # Go unit + integration tests against a real PostgreSQL
make api-smoke   # cURL acceptance checks against the running API
```

Decisions taken in this phase:

- **Go with the standard library router.** Go 1.22's `net/http` ServeMux does
  method and path-parameter routing, so no third-party router is needed. `pgx`
  is the only database dependency.
- **Stateless JWTs, 24-hour lifetime.** Refresh tokens and a revocation list
  both need a storage table the Phase 1 schema does not have, so they are
  deferred rather than improvised. Suspension still takes effect immediately,
  because every authorised request re-reads the account.
- **Validation is duplicated on purpose.** The handlers check what the database
  constraints already enforce, so a bad request returns a 422 naming the field
  instead of a 500 from a constraint violation.
- **Creating an event grants the organizer role**, since publishing events and
  issuing free tickets is free and needs no approval (SRS 3.1).


---

## Phase 3: Organizer Web Dashboard

Next.js 16 App Router + React 19 + Tailwind 4. An organizer can register, sign
in, list their events and create new ones against the real Go API. Full
documentation in **[web/README.md](web/README.md)**.

```bash
make web-install   # once
make web-dev       # http://localhost:3000
```

Decisions taken in this phase:

- **The token lives in a cookie, not localStorage.** `proxy.ts` (Next.js 16's
  rename of middleware) runs on the server and can only read cookies, so route
  protection happens before render instead of flashing the dashboard and then
  bouncing. The XSS trade-off is documented in the frontend README.
- **Two gates.** The proxy checks a cookie *exists*; `AuthProvider` confirms
  with `GET /auth/me` that it is still *valid*, which also catches an account
  suspended after the token was issued.
- **The dashboard reads `/events/mine`, not `/events`.** The public catalogue
  returns only published public events, so a new draft would be invisible on
  the dashboard that just created it.
- **Field-level errors round-trip.** The API's 422 `fields` map is rendered on
  the exact inputs that caused it.
- **Times are converted in the event's own timezone,** not the browser's.


---

## Phase 4: Ticket Types & Simulated Checkout

Organizers configure ticket types; attendees browse a public event page and buy
tickets with a simulated KZT checkout that cannot oversell.

- **Backend:** ticket-type CRUD, `GET /public/events/{slug}`, and an atomic
  `POST /events/{id}/checkout` that takes inventory, records a paid order,
  issues tickets and files a simulated payment — all in one transaction.
- **Frontend:** ticket-type management inside an event, a public
  `/events/[slug]` page with a ticket selector, a `<dialog>` checkout, and an
  `/orders/[id]` confirmation.

Overselling is prevented three times over — a `FOR UPDATE` row lock, a
conditional decrement, and the Phase 1 `ticket_types_inventory_chk` constraint.
A concurrency test fires 30 simultaneous buyers at 10 tickets and asserts that
exactly 10 succeed. Details in [api/README.md](api/README.md).

Money never becomes a float: prices are decimal strings end to end, and every
total is computed by PostgreSQL in `numeric(14,2)`.


---

## Phase 5: Digital Delivery

Every ticket carries an admission token of the form `TKT_<uuid>`, rendered as a
QR code on a print-ready A4 PDF and previewed on the order page.

```bash
curl -O -J http://localhost:8080/api/v1/tickets/<ticket-id>/pdf
```

- **Backend:** `internal/ticketpdf` renders the A4 page with `go-pdf/fpdf` and
  `skip2/go-qrcode` — event, local date/time, venue, attendee, ticket id, seat
  when assigned, and a 60 mm high-error-correction QR. No payment data ever
  reaches a printed ticket (SRS 4.7).
- **Frontend:** the order confirmation embeds each QR and offers a
  "Download PDF ticket" button.

The QR is verified by **decoding it back with a real QR reader**, not by
trusting the encoder — including off the rendered PDF page, in grayscale, and
downscaled to 700 px. Details in [api/README.md](api/README.md).


---

## Phase 6: Mobile Ticket Scanner

An Expo / React Native app for Event Admins: sign in, pick an assigned event,
scan attendee QR codes, and see a full-screen green or red result.

```bash
cd mobile && npm install && npx expo start
```

- **Backend:** `POST /events/{id}/check-in` validates the token, records the
  admission and flips the ticket to `checked_in`; a repeat scan returns
  `409 already_checked_in`, a campaign QR `400 campaign_token`. Assignment
  endpoints let an organizer name their gate staff.
- **App:** login → event selector (only assigned events) → camera scanner, with
  a full-screen result carrying the attendee name and the running count.

Duplicate entry is prevented by the Phase 1
`check_in_one_active_per_ticket_uidx` index, not by application logic — 8
simultaneous scans of one ticket admit exactly one person. Details in
[api/README.md](api/README.md).


---

## Phase 7: Promo Codes & Campaigns

Organizers create promo codes with a percentage or fixed KZT discount, a
redemption limit and validity dates. Each campaign gets a QR code encoding a
trackable event link; scanning it opens the event with the discount already
applied.

- **Backend:** campaign CRUD, a promo preview that prices a basket, and a
  checkout that redeems the code inside the same transaction that takes the
  inventory.
- **Frontend:** a campaign workspace for organizers, and an attendee promo box
  that accepts a typed code or auto-applies one from `?c=CMP_…`.
- **Scanner:** `CMP_` codes are refused at the gate — as the bare token, as the
  campaign link, and as a link with tracking parameters appended.

The discount never comes from the client: the link carries only an opaque token,
and PostgreSQL computes the amount from the stored ticket prices. A single-use
code survives 12 simultaneous checkouts with exactly one redemption. Details in
[api/README.md](api/README.md).


---

## Phase 8: Support Chat & Moderation

Attendees open support cases from their order; organizers answer and track them
through Open → In Progress → Resolved. Platform admins can suspend a rogue event.

- **Support:** contextual cases (the order carries its event with it), a shared
  thread component for both sides, internal notes that never reach the attendee,
  and polling rather than sockets — SRS 4.13 explicitly does not require
  real-time delivery.
- **Moderation:** `POST /admin/events/{id}/suspend` behind the `platform_admin`
  role. Checkout returns **403 `event_suspended`** before touching inventory, the
  public page shows a banner instead of a ticket selector, and tickets already
  sold stay valid so paying attendees are not stranded.

Access follows SRS 4.13 exactly: a case is visible to its requester, the event's
organizer and platform admins — and anyone else gets a 404 rather than a 403,
because confirming a case exists is itself a leak. Details in
[api/README.md](api/README.md).

## Phase 9: Organizer Analytics & History

Every organizer question about a past event — how many sold, how much came in,
who actually turned up — answered from the rows themselves, plus one-click reuse
of an event that worked.

- **Analytics:** `GET /events/{id}/analytics` reports tickets sold, KZT revenue,
  discounts given, orders, check-ins and the two percentages, broken down by
  ticket type and by campaign, with a sales-over-time series. Everything is
  computed in SQL from `orders`, `tickets` and `check_in_records`; nothing is
  cached or counted twice. Date and ticket-type filters narrow it.
- **History:** `GET /events/{id}/timeline` reads the append-only `audit_logs`,
  so an event's record cannot be edited or quietly rewritten.
- **Duplication:** `POST /events/{id}/duplicate` copies the configuration —
  title, venue, timezone, capacity and every ticket type — and nothing else. The
  copy is a **draft** with zero orders, zero sold tickets and zero check-ins, and
  records where it came from in `duplicated_from_event_id`.

The dashboard groups events as Upcoming / Active / Completed / Cancelled /
Drafts. That stage is derived from the dates and status at read time rather than
stored, so an event cannot sit in a stale bucket. Details in
[api/README.md](api/README.md) and [web/README.md](web/README.md).

## Phase 10: Refunds, Paid-Sales Activation & Notifications

The last three MVP requirements from the SRS: money can be given back, an
organizer must be cleared before taking it, and people are told what happened.

- **Refunds (SRS 4.9):** `POST /orders/{id}/refund` reverses a paid order in one
  transaction — the order, its payment, a `refunds` record, every issued ticket,
  the inventory those tickets held, and an append-only audit entry. Refunded
  tickets become `refunded`, and the Phase 6 gate already refuses that status,
  so a refunded QR stops working the instant the transaction commits.
- **Paid-sales activation (SRS 4.5):** a four-step checklist — identity, payout
  destination, seller terms, and a simulated activation fee — that must be
  complete before a paid ticket can be sold. The gate lives *inside* the
  checkout transaction, so it cannot be raced. Free tickets are never gated:
  activation exists to clear an organizer to take money.
- **Notifications (SRS 4.10):** completing a checkout, or a refund, prints a
  formatted mock email to stdout and records it in the `notifications` outbox.
  Delivery is asynchronous — nobody's tickets wait on a mail server — and a
  notification failure can never fail a purchase that already took the money.

Nothing charges anybody: every payment and refund row carries
`is_simulated = true`, which SRS 4.6 requires. Details in
[api/README.md](api/README.md) and [web/README.md](web/README.md).

## Phase 12: Strict Specification Adherence

Five requirements the earlier phases had left on the page.

- **Administrative portal (SRS 2.1, 4.12):** `/admin`, restricted to the
  `platform_admin` role. One search box across users, events, orders **and**
  payments — an admin handed a name, an email or an order number rarely knows
  which kind of thing it is — plus a CSV export of the operational report.
- **Account recovery (SRS 4.1):** email verification and password reset, both
  single-use, both expiring, both **stored only as a SHA-256 hash** so a leaked
  backup contains no working links. The token arrives by the mock console
  emailer. "Forgot password" answers identically whether or not the address has
  an account, so the form cannot be used to enumerate who is registered.
- **Event images and refund policy (SRS 4.2, 4.9):** a multipart upload
  endpoint storing to local disk (standing in for object storage), with the
  file type sniffed from the content rather than trusted from the name, and a
  random stored filename. Both the banner and the policy appear on the public
  event page.
- **Manual attendee search (SRS 4.8):** `GET /events/{id}/attendees?q=` and a
  "Find attendee" screen in the scanner app, for when a QR will not scan at
  all. Admission goes by ticket id and runs the *same* transaction as a camera
  scan, so it is neither weaker nor a way around duplicate protection.
- **Notification triggers (SRS 4.10):** password reset, email verification,
  event cancellation and new support message now join purchase confirmation and
  refund completion on the console emailer.

Details in [api/README.md](api/README.md), [web/README.md](web/README.md) and
[mobile/README.md](mobile/README.md).

## Deferred technical debt (Phase 13)

Seven of the ten deferred items from the SRS. **Items 7 (i18n), 8 (interactive
seat map) and 10 (offline scanning) are not built** — see the end of this
section.

- **Cart holds (SRS 4.6, 4.3.1):** picking tickets now reserves them
  (`quantity_reserved`) instead of selling them. A basket lives for 15 minutes
  and then goes back on sale, released three ways: explicitly when the attendee
  cancels, opportunistically when anybody else touches that event's inventory,
  and by a background sweeper for everything else. `POST /events/{id}/holds`,
  `GET|DELETE /orders/{id}/hold`, `POST /orders/{id}/confirm`. The one-shot
  `POST /checkout` is now *composed from* hold and confirm inside a single
  transaction, so there is one implementation of buying a ticket, not two.
- **Processing fees (SRS 3.3):** `total = subtotal − discount + fee`, computed
  in PostgreSQL numeric. Default 3.5%, configurable with
  `PROCESSING_FEE_PERCENT` / `PROCESSING_FEE_FIXED_KZT`. Free baskets and
  fully-discounted baskets are charged nothing.
- **Cyrillic PDFs (SRS 7):** the transliteration hack is gone. DejaVu Sans
  Condensed is embedded in every ticket, so Kazakh and Russian render natively
  — including ә ғ қ ң ө ұ ү һ і and the ₸ sign.
- **Calendar export (SRS 4.11):** `GET /events/{id}/calendar.ics`, addressable
  by id or slug, in the event's own timezone with a VTIMEZONE that travels with
  the file. A stable UID means a re-download replaces the entry rather than
  duplicating it, and a cancelled event exports `STATUS:CANCELLED`.
- **Support attachments (bonus, SRS 4.13):** `support_messages` gained four
  attachment columns that move together under one CHECK. A message references a
  file already uploaded through `POST /uploads/images`, and the server refuses
  any URL that is not one of its own.
- **httpOnly session cookies (SRS 7):** the JWT no longer reaches the browser.
  Next.js route handlers hold it in an `HttpOnly; SameSite=Strict` cookie and
  proxy every API call. `document.cookie`, `localStorage` and `sessionStorage`
  are all empty on a signed-in page.
- **GA4 (bonus):** campaign-link visits, checkout starts and purchases, with
  UTM and referrer-host attribution. No PII, and deliberately **no campaign
  token** — attribution reports *that* a visit came through a QR, never which
  discount credential it carried. Inert unless `NEXT_PUBLIC_GA4_MEASUREMENT_ID`
  is set.

### Not built

- **i18n (kk/ru)** — the Cyrillic font subsets are wired into the layout, but
  no locale switcher or message catalogue exists.
- **Interactive seat map** — the backend half is done: `seat_holds` are written
  and released with the basket, `order_items.seat_id` and the ticket's
  section/row/number are populated, and one active hold per seat is enforced by
  a partial unique index. What is missing is the seat-layout API, seed data for
  a predefined venue, and the SVG picker itself.
- **Offline scanning** — the scanner still requires a connection.
