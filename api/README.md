# BiletFlow API — Phases 2 & 4-13

Go REST API for account registration, JWT login, event CRUD, ticket types, a
simulated KZT checkout, and printable QR tickets — backed by the Phase 1
PostgreSQL schema.

Requires Go 1.21+ (the module targets 1.25; Go's default `GOTOOLCHAIN=auto`
fetches the toolchain automatically) and the Phase 1 database running.

---

## Run it

```bash
make up        # PostgreSQL (from the repository root)
make api-run   # API on http://localhost:8080
```

Verify:

```bash
curl -s http://localhost:8080/health
```

| Command | What it does |
| --- | --- |
| `make api-run` | Run the API with `.env` loaded |
| `make api-build` | Compile to `api/bin/api` |
| `make api-test` | Full Go suite: unit + integration against PostgreSQL |
| `make api-smoke` | cURL acceptance checks against a running API |
| `make api-check` | gofmt + vet + test |

---

## Verifying the phase-2 success criteria

Everything below is automated twice — as Go integration tests
(`TestPhase2SuccessCriteria`) and as cURL checks (`make api-smoke`) — but here
are the requests to run by hand in Postman or a terminal.

**1. Register with an email and password → 201**

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/register -H 'Content-Type: application/json' -d '{"email":"dana@biletflow.test","password":"correct horse battery"}'
```

**2. Log in and receive a token → 200**

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"dana@biletflow.test","password":"correct horse battery"}'
```

Save the `access_token` from the response:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"dana@biletflow.test","password":"correct horse battery"}' | python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])')
```

**3. Create an event with that token → 201 Created**

```bash
curl -i -X POST http://localhost:8080/api/v1/events -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"title":"Almaty Winter Jazz Night","category":"music","venue_name":"Almaty Demo Hall","venue_address":"Abay Avenue 44, Almaty","starts_at":"2026-12-20T19:00:00+05:00","ends_at":"2026-12-20T22:00:00+05:00","timezone":"Asia/Almaty","capacity":250}'
```

And confirm the row is really in PostgreSQL:

```bash
docker compose exec -T db psql -U biletflow -d biletflow -c "SELECT id, title, status, organizer_id FROM events ORDER BY created_at DESC LIMIT 1;"
```

### Postman

Import [docs/biletflow-api.postman_collection.json](../docs/biletflow-api.postman_collection.json).
The Register and Login requests save the token into a `{{token}}` collection
variable automatically, so every later request is already authenticated;
Create Event saves `{{eventId}}`.

---

## Endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/health` | – | API and database status |
| `POST` | `/api/v1/auth/register` | – | Create an account, returns a token |
| `POST` | `/api/v1/auth/login` | – | Exchange credentials for a token |
| `GET` | `/api/v1/auth/me` | Bearer | The current account and its roles |
| `POST` | `/api/v1/events` | Bearer | Create an event (starts as a draft) |
| `GET` | `/api/v1/events` | – | Published public events, filtered and paged |
| `GET` | `/api/v1/events/mine` | Bearer | Every event this organizer owns |
| `GET` | `/api/v1/events/{id}` | optional | One event |
| `PATCH` | `/api/v1/events/{id}` | Bearer | Partial update |
| `DELETE` | `/api/v1/events/{id}` | Bearer | Delete a draft |
| `POST` | `/api/v1/events/{id}/publish` | Bearer | Draft/unpublished → published |
| `POST` | `/api/v1/events/{id}/unpublish` | Bearer | Published → unpublished |
| `POST` | `/api/v1/events/{id}/cancel` | Bearer | Cancel the event |
| `GET` | `/api/v1/events/{id}/timeline` | Bearer | Append-only activity log |
| `GET` | `/api/v1/events/{id}/ticket-types` | Bearer | Ticket types, hidden ones included |
| `POST` | `/api/v1/events/{id}/ticket-types` | Bearer | Create a ticket type |
| `PATCH` | `/api/v1/ticket-types/{id}` | Bearer | Update price, stock, sales window, visibility |
| `DELETE` | `/api/v1/ticket-types/{id}` | Bearer | Delete a ticket type with no sales |
| `GET` | `/api/v1/public/events/{slug}` | – | Attendee view: event + on-sale ticket types |
| `POST` | `/api/v1/events/{id}/checkout` | optional | Simulated purchase |
| `GET` | `/api/v1/orders/{id}` | optional | Order with its items and tickets |
| `GET` | `/api/v1/tickets/{id}` | – | One ticket with its delivery links |
| `GET` | `/api/v1/tickets/{id}/pdf` | – | Print-ready A4 PDF ticket |
| `GET` | `/api/v1/tickets/{id}/qr.png` | – | Admission QR as a PNG |
| `GET` | `/api/v1/events/scannable` | Bearer | Events this account may check in for |
| `POST` | `/api/v1/events/{id}/check-in` | Bearer | Admit a scanned QR token |
| `GET` | `/api/v1/events/{id}/check-in/stats` | Bearer | Live registered / checked-in counts |
| `POST` | `/api/v1/tickets/{id}/check-in/reverse` | Bearer | Undo an accidental check-in |
| `GET` | `/api/v1/events/{id}/staff` | Bearer | Event staff |
| `POST` | `/api/v1/events/{id}/staff` | Bearer | Assign an Event Admin by email |
| `DELETE` | `/api/v1/events/{id}/staff/{assignmentId}` | Bearer | Revoke an assignment |
| `GET` | `/api/v1/events/{id}/campaigns` | Bearer | Campaigns with their sales figures |
| `POST` | `/api/v1/events/{id}/campaigns` | Bearer | Create a promo code + campaign QR |
| `PATCH` | `/api/v1/campaigns/{id}` | Bearer | Enable or disable a campaign |
| `DELETE` | `/api/v1/campaigns/{id}` | Bearer | Delete an unredeemed campaign |
| `GET` | `/api/v1/campaigns/{id}/qr.png` | – | The campaign QR image |
| `POST` | `/api/v1/events/{id}/promo/preview` | optional | Price a promo code against a basket |
| `GET` | `/api/v1/support/categories` | – | Issue categories for the picker |
| `GET` | `/api/v1/support/cases` | Bearer | Cases this account opened |
| `POST` | `/api/v1/support/cases` | Bearer | Open a case on an order or ticket |
| `GET` | `/api/v1/support/cases/{id}` | Bearer | One thread |
| `POST` | `/api/v1/support/cases/{id}/messages` | Bearer | Reply |
| `PATCH` | `/api/v1/support/cases/{id}` | Bearer | Change status (organizer only) |
| `GET` | `/api/v1/events/{id}/support/cases` | Bearer | The organizer's inbox |
| `POST` | `/api/v1/admin/events/{id}/suspend` | **Admin** | Stop an event selling |
| `POST` | `/api/v1/admin/events/{id}/unsuspend` | **Admin** | Lift a suspension |
| `GET` | `/api/v1/events/{id}/analytics` | Bearer | Sales, revenue and attendance |
| `GET` | `/api/v1/events/{id}/timeline` | Bearer | The event's append-only history |
| `POST` | `/api/v1/events/{id}/duplicate` | Bearer | Copy the configuration into a new draft |
| `GET` | `/api/v1/events/{id}/orders` | Bearer | The organizer's attendee view |
| `POST` | `/api/v1/orders/{id}/refund` | Bearer | Refund an order in full |
| `GET` | `/api/v1/events/{id}/activation` | Bearer | Paid-sales checklist state |
| `POST` | `/api/v1/events/{id}/activation` | Bearer | Complete checklist steps |
| `POST` | `/api/v1/admin/events/{id}/paid-sales/suspend` | **Admin** | Stop paid sales only |
| `POST` | `/api/v1/admin/events/{id}/paid-sales/unsuspend` | **Admin** | Restore paid sales |
| `POST` | `/api/v1/auth/password-reset/request` | – | Email a reset token |
| `POST` | `/api/v1/auth/password-reset` | – | Consume it and set a password |
| `POST` | `/api/v1/auth/verify-email/request` | Bearer | Re-send the confirmation |
| `POST` | `/api/v1/auth/verify-email` | – | Confirm an address |
| `GET` | `/api/v1/events/{id}/attendees` | Bearer | Manual attendee search |
| `POST` | `/api/v1/events/{id}/check-in/manual` | Bearer | Admit without a QR |
| `POST` | `/api/v1/uploads/images` | Bearer | Upload an event banner |
| `GET` | `/uploads/{file}` | – | Serve an uploaded banner |
| `GET` | `/api/v1/admin/search` | **Admin** | Users, events, orders, payments |
| `GET` | `/api/v1/admin/reports/events.csv` | **Admin** | Operational report |
| `POST` | `/api/v1/events/{id}/holds` | optional | Reserve a basket for 15 minutes |
| `GET` | `/api/v1/orders/{id}/hold` | optional | Read a basket back |
| `DELETE` | `/api/v1/orders/{id}/hold` | optional | Cancel a basket now |
| `POST` | `/api/v1/orders/{id}/confirm` | optional | Pay for a held basket |
| `GET` | `/api/v1/events/{id}/calendar.ics` | – | iCalendar export (id or slug) |
| `GET` | `/api/v1/events/{id}/seats` | optional | Interactive seat map with live states |
| `GET` | `/api/v1/events/{id}/roster` | Bearer | Hashed guest list for offline scanning |
| `POST` | `/api/v1/events/{id}/check-in/sync` | Bearer | Reconcile admissions made offline |

Listing filters: `limit` (1–100, default 20), `offset`, `category`, `q`,
`starts_after`, `starts_before`. The organizer's own list, `GET /events/mine`,
also takes `status` and `lifecycle` (`upcoming`, `active`, `completed`,
`cancelled`, `draft`, `suspended`).

### Errors

Every non-2xx response uses the same envelope, so clients switch on `code`
rather than parsing prose:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "The request body failed validation.",
    "fields": { "ends_at": "End time must be after the start time." }
  }
}
```

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid_json` | Malformed body, wrong content type, unknown field |
| 400 | `validation_failed` | A path parameter is not a UUID |
| 401 | `unauthorized` / `invalid_credentials` | Missing, invalid or expired token; bad login |
| 403 | `forbidden` | Not the organizer, or the account is suspended |
| 404 | `not_found` | Unknown route, or an event you may not see |
| 405 | `method_not_allowed` | Wrong method on a real path (`Allow` header included) |
| 409 | `conflict` | Duplicate email or slug, illegal state transition, per-order limit |
| 409 | `insufficient_inventory` | Not enough stock; the body carries `remaining` |
| 409 | `not_on_sale` | The ticket type is hidden or outside its sales window |
| 409 | `sales_closed` | The event is unpublished, cancelled, or outside registration |
| 422 | `validation_failed` | Body failed validation, with `fields` |
| 500 | `internal_error` | Logged with the request id; no internals leak out |

---

## Behaviour worth knowing

- **Registration needs only email and password.** `full_name` is derived from
  the email local part when omitted (`dana.a@x.kz` → "Dana A"), because
  `users.full_name` is `NOT NULL` in the schema.
- **Registration returns a token**, so a client can register and immediately
  create an event without a second round trip.
- **Creating an event grants the organizer role.** Publishing an event and
  issuing free tickets is free and needs no approval (SRS 3.1), so a new
  account is not blocked from `POST /events`.
- **Events are created as drafts.** Publishing is a separate, explicit call.
- **Slugs are generated and de-duplicated.** Two events titled "Concert" become
  `concert` and `concert-2`. A slug the *client* chose is never silently
  renamed — a clash returns 409. Cyrillic titles are transliterated, so
  "Концерт в Алматы" becomes `kontsert-v-almaty`.
- **PATCH tells absent from null.** An omitted field is untouched; an explicit
  `null` clears a nullable column.
- **Only drafts can be deleted.** A published event, or one with orders, must
  be cancelled so ticket history survives.
- **Authorisation re-reads the account.** A token issued before a suspension
  stops working immediately, and role changes take effect without re-login.
- **Lifecycle changes are audited** into `audit_logs`, which is the same
  append-only table the organizer timeline (SRS 4.16) reads from.

### Authentication design

Access tokens are stateless HS256 JWTs valid for 24 hours, carrying `sub`,
`email`, `roles`, `iss`, `iat`, `nbf`, `exp` and a `jti`. The parser pins the
algorithm, so a token re-signed with `alg: none` is rejected.

There is no refresh token and no revocation list: both need a storage table the
Phase 1 schema does not have, so they are deliberately deferred rather than
bolted on. Suspension is effective immediately because every authorised request
re-reads the account.

Passwords are bcrypt hashes at cost 12. Passwords over 72 bytes are rejected
rather than silently truncated by bcrypt, and login spends a comparable amount
of time on unknown addresses so timing does not reveal which are registered.

---

## Layout

```
cmd/api/main.go          startup, graceful shutdown
internal/config          environment settings
internal/database        pgx connection pool
internal/httpx           JSON decode, response and error envelope
internal/auth            bcrypt hashing, JWT issue/parse
internal/store           users, events, audit log, PATCH Optional type
internal/api             router, middleware, handlers, tests
internal/testutil        test database harness
scripts/smoke_test.sh    cURL acceptance checks
```

---

## Tests

```bash
make api-test              # everything
make api-test-v            # verbose
cd api && go test ./internal/api/ -run TestPhase2SuccessCriteria -v
```

Unit tests cover password hashing, JWT issuing and parsing, slug generation,
the PATCH `Optional` type and configuration loading.

Integration tests run the **real** HTTP stack — router, middleware, handlers —
against a **real** PostgreSQL database (`biletflow_test`, created and migrated
from `db/init` on first run), then assert against the tables with direct SQL.
Nothing is mocked, so a passing test means the request path genuinely works.

The suite fails rather than skips when the database is unreachable, and each
test truncates the test database before and after it runs. Override the target
with `TEST_DATABASE_URL` / `TEST_ADMIN_DATABASE_URL`.


---

## Ticket types and checkout (Phase 4)

### Money never becomes a float

Prices cross the wire as decimal **strings** (`"5000.00"`), and all arithmetic —
line totals, order totals — is done by PostgreSQL in `numeric(14,2)`. No amount
is ever parsed into a JavaScript or Go float, so nothing can be rounded away.

```json
{ "name": "General Admission", "price_kzt": "5000", "quantity_total": 5 }
```

### How overselling is prevented

Three independent layers, each of which would be sufficient on its own:

1. **A row lock.** The checkout transaction opens with
   `SELECT … FROM ticket_types WHERE id = ANY($1) … ORDER BY id FOR UPDATE`.
   Concurrent buyers queue behind each other instead of all reading the same
   "remaining" and all deciding there is room. Locking in `id` order means two
   orders touching the same pair of ticket types cannot deadlock.
2. **A conditional decrement.** The update itself repeats the test:
   `UPDATE ticket_types SET quantity_sold = quantity_sold + $2 WHERE id = $1
   AND quantity_sold + quantity_reserved + $2 <= quantity_total`. Zero rows
   affected means the stock went, and the transaction rolls back.
3. **A database constraint.** `ticket_types_inventory_chk` from Phase 1 refuses
   to store `quantity_sold + quantity_reserved > quantity_total` at all.

`TestCheckoutDoesNotOversellUnderConcurrency` fires 30 simultaneous buyers at 10
tickets and asserts that exactly 10 succeed, 20 get a 409, and `quantity_sold`
lands on exactly 10. Removing layers 1 and 2 together makes that test fail, so
it is genuinely testing the guard rather than passing by luck.

### What one checkout writes

A single transaction, all or nothing:

| Table | Row |
| --- | --- |
| `ticket_types` | `quantity_sold` increased |
| `orders` | one order, `status = paid`, `currency = KZT` |
| `order_items` | one line per ticket type |
| `attendees` | one attendee, from the checkout form |
| `tickets` | one per ticket bought, `status = valid`, `qr_token` prefixed `TKT_` |
| `payments` | one payment, `is_simulated = true`, `status = succeeded` |
| `audit_logs` | an `order.created` timeline entry |

### Deliberately deferred

- **Paid-sales activation is not enforced.** SRS 4.5 requires an organizer to
  complete activation before selling paid tickets. The schema models it
  (`paid_sales_activations`), but the checkout does not yet check it — gating on
  it now would make it impossible to demonstrate a paid purchase. The check
  belongs in `handleCheckout`, next to the event-status checks.
- **No reservation step.** `quantity_reserved` exists in the schema and is
  respected by every inventory calculation, but the simulated checkout is
  instant, so it goes straight to `quantity_sold`. A real payment flow would
  reserve first, then convert on confirmation.
- **No processing fee.** `processing_fee_kzt` is always `0`, so
  `total = subtotal - discount`.
- **Event capacity is not a second limit.** The sum of `quantity_total` across
  ticket types is what bounds sales; `events.capacity` is informational.


---

## Digital ticket delivery (Phase 5)

### The admission token

Every ticket carries a `qr_token` of the form `TKT_<uuid>`:

```
TKT_4eaea347-8a87-495f-9c92-385c807202e7
```

The `TKT_` prefix is enforced by the Phase 1 `tickets_qr_token_prefix_chk`
constraint and is what keeps an admission code disjoint from a campaign QR
(`CMP_`), so a promotional code can never be presented at the gate (SRS 4.14).

The body is a **fresh random UUID, not the ticket's own id**. A ticket id
travels in URLs and logs; it must not double as a working admission credential.

### The PDF

`internal/ticketpdf` renders a single A4 page (595.28 × 841.89 pt) with
[`go-pdf/fpdf`](https://github.com/go-pdf/fpdf) and
[`skip2/go-qrcode`](https://github.com/skip2/go-qrcode). It carries exactly what
SRS 4.7 requires — event title, start and end in the **event's own timezone**,
venue name and address, attendee name, ticket type, ticket id, order number, an
assigned seat when there is one, and the QR — and deliberately carries no
payment information at all.

Design choices that matter for something meant to be printed and scanned:

- **60 mm QR at 1024 px** ≈ 430 DPI, well beyond what any scanner needs.
- **High error correction** (~30% recoverable), because a ticket gets folded,
  creased and photographed at an angle in a queue.
- **Pure black on white, no colour-coded information**, so a grayscale print
  loses nothing. Verified: rendering the PDF to grayscale at 700 px wide and
  scanning it still returns the exact token.
- **The token is printed as text under the code**, so staff can key it in when a
  scanner fails.

### How the QR is tested

Not by trusting the encoder. `TestQRRoundTripsTheExactToken` encodes a token and
decodes it back with a real QR reader ([gozxing](https://github.com/makiuchi-d/gozxing)),
asserting the scanned string is byte-for-byte the token. The API test does the
same over HTTP against `/qr.png` and cross-checks the result against the value
stored in PostgreSQL.

The PDF's own text is asserted by inflating its content streams and reading the
text-showing operators — the same bytes a viewer renders — rather than trusting
the inputs.

### Access model

`/tickets/{id}/pdf` and `/qr.png` are reachable by ticket UUID with no token,
the same capability model as the order confirmation: a guest checkout has no
account to authenticate against, so the unguessable id in the link is what
grants access. Anyone holding the link can print the ticket — which is exactly
what a ticket is — and duplicate admission is prevented at scan time by the
`check_in_one_active_per_ticket_uidx` constraint, not by hiding the PDF.

Responses are `Cache-Control: no-store` because a ticket can be cancelled or
refunded after it was issued.

### Known limitation

The PDF uses the built-in Helvetica font, whose cp1252 encoding has no Cyrillic.
Latin-1 accents (`Café`) render correctly; Cyrillic is **transliterated**
(`Концерт в Алматы` → `Kontsert v Almaty`) rather than emitted as mojibake. The
proper fix is embedding a Unicode TTF such as DejaVu Sans in
`internal/ticketpdf`, which SRS section 7 will require before a Kazakh or
Russian launch.


---

## Check-in (Phase 6)

`POST /api/v1/events/{id}/check-in` takes a scanned QR token and admits its
holder:

```json
{ "qr_token": "TKT_4eaea347-8a87-495f-9c92-385c807202e7", "device_label": "ios scanner" }
```

### How duplicate entry is prevented

`check_in_one_active_per_ticket_uidx` — the Phase 1 unique partial index over
`check_in_records` where `reversed_at IS NULL` — is the guarantee. Two scanners
hitting the same ticket at the same instant cannot both succeed: one insert
wins, the other comes back a unique violation and is reported as already used.
The ticket row is also locked with `FOR UPDATE` first, so both paths agree on
what happened.

That insert runs inside a **savepoint**. A unique violation aborts the enclosing
PostgreSQL transaction, and the handler still has to query who used the ticket
and when in order to answer usefully — so the failure is contained rather than
allowed to poison the transaction.

`TestCheckInIsAtomicUnderConcurrency` fires 8 simultaneous scans at one ticket
and asserts exactly one is admitted and exactly one active record exists.

### What the scanner is told

Every refusal has its own code, because "denied" gives venue staff nothing to
act on:

| Status | Code | Meaning |
| --- | --- | --- |
| 200 | – | Admitted; returns attendee, ticket type, seat and live counts |
| 409 | `already_checked_in` | Used before; carries the attendee name and first-entry time |
| 409 | `wrong_event` | A valid ticket, for a different event |
| 409 | `ticket_not_valid` | Cancelled or refunded |
| 400 | `campaign_token` | A `CMP_` promotional code (SRS 4.14 — never admission) |
| 404 | `unknown_ticket` | Matches no ticket |
| 403 | `forbidden` | Not assigned to this event's gate |

### Who may scan

The organizer, a platform admin, or anyone holding an **unrevoked**
`event_admin` / `manager` assignment on that event. `GET /events/scannable`
returns exactly that set, which is what fills the app's event selector — an
Event Admin never sees an event they were not assigned to, and a revoked
assignment removes both the listing and the gate immediately.

Assignment is by email, because an organizer knows their colleague's address,
not their user id. Being assigned also grants the `event_admin` role.

### Reversal

`POST /tickets/{id}/check-in/reverse` marks the record reversed rather than
deleting it, returns the ticket to `valid`, and lets it be scanned again
(SRS 4.8). The reversed row is retained for the audit trail.


---

## Promo codes and campaigns (Phase 7)

### The two namespaces never meet

| Prefix | What it is | At the gate |
| --- | --- | --- |
| `TKT_<uuid>` | An admission ticket | Admits one person, once |
| `CMP_<uuid>` | A promotional campaign | **Always refused** (SRS 4.14) |

Both prefixes are enforced by Phase 1 check constraints, so the separation
holds in the database and not merely in application code.

A campaign QR does **not** encode the bare token. SRS 4.14 requires a trackable
event link, so what a scanner reads is:

```
http://localhost:3000/events/spring-festival-2027?c=CMP_3493d2c0-9200-42ea-a799-1313f9268a30
```

That is why `ClassifyScannedCode` looks for the token *inside* a URL as well as
at the start of a bare string. Matching only a leading prefix would classify a
scanned poster as "unknown code" and leave venue staff guessing, when the honest
answer is "that is the advert, not their ticket".

`TestGateRejectsEveryCampaignForm` scans the bare token, the campaign link, an
https variant and a link with extra tracking parameters — all four are refused
with `campaign_token`, and no `check_in_records` row is written.

### The discount is never taken from the client

The link carries an opaque token and nothing else. The server looks up what that
token is worth, prices the basket from the **stored ticket-type prices**, and
computes the discount in PostgreSQL:

```sql
CASE WHEN $type = 'percentage' THEN round(base * $value / 100, 2)
     ELSE least($value, base) END
```

`base` covers only the lines the campaign applies to, so a campaign restricted
to VIP tickets discounts the VIP line and leaves the rest alone. A fixed
discount is capped at the basket, so a 99 999 KZT code on a 1 000 KZT ticket
produces a 0.00 total rather than a negative one.

The preview endpoint mirrors this with `math/big.Rat` — never `float64` — and
rounds half away from zero exactly as `round(numeric, 2)` does, so the figure on
screen and the figure in the order agree to the tiyn.

### Redemption limits are atomic

Validation and payment are separate moments, and the last redemption can be
taken in between. So the campaign row is locked `FOR UPDATE` **inside the
checkout transaction** and its limit re-checked there, not just when the
attendee typed the code. `campaigns_redemption_limit_chk` is the backstop.

`TestPromoRedemptionLimitIsAtomic` fires 12 simultaneous checkouts at a
single-use code: exactly one is discounted, eleven get `promo_exhausted`, and
`redemption_count` lands on 1.

The winner also marks the campaign `exhausted`, which is why exhaustion is
checked *before* the general status — everyone behind the winner must be told
"fully redeemed" rather than the vaguer "not active".

### Why every failure has its own code

"Invalid code" tells an attendee nothing about what to do next:

| Code | Meaning |
| --- | --- |
| `promo_not_found` | No campaign uses that code |
| `promo_wrong_event` | A real code, for a different event |
| `promo_not_active` | Disabled by the organizer |
| `promo_not_started` | Valid, but not yet |
| `promo_expired` | The window has closed |
| `promo_exhausted` | Every redemption is gone |
| `promo_not_applicable` | Does not cover the selected ticket types |

The checkout path returns these too, not only the preview, because a code can
run out between seeing the discount and paying for it.

### Campaign reporting

`GET /events/{id}/campaigns` returns orders, tickets sold, gross revenue and
discount given per campaign (SRS 4.14), computed from the authoritative order
records rather than a counter that could drift.


---

## Support chat and moderation (Phase 8)

### Who can see a case

SRS 4.13 requires that people cannot read cases they are not party to. A case is
visible to the person who opened it, to the organizer of the event it concerns,
and to platform admins. Everyone else gets **404, not 403** — a 403 would
confirm that a case exists, which is itself a leak about someone else's order.

Internal notes live on the same thread and are filtered out in SQL for the
requester, so a staff note cannot reach an attendee through any code path.
An attendee who sets `internal_note: true` is simply ignored.

Only staff can change a status. A requester can reply but cannot mark their own
case answered.

### Opening a case needs an account

`support_cases.requester_user_id` is `NOT NULL`, and a conversation needs an
identity to come back to. A guest checkout stores no user id, so
`OrderBelongsTo` matches on the account id **or the buyer email** — someone who
bought as a guest and registered later with the same address keeps access to
their own order. The order page tells them exactly which address to use.

Context is captured automatically: pass an `order_id` and the case picks up the
event through it (SRS 4.13).

### Status moves on its own where it should

An organizer's first reply moves an `open` case to `in_progress` and assigns it
to them, so nobody has to remember to set it. An internal note does not — it is
not a reply to the customer. Resolving stamps `resolved_at` and reopening clears
it, which `support_cases_resolved_chk` requires stay consistent.

### Polling, not sockets

SRS 4.13 explicitly does not require real-time delivery for the MVP, and the
"academic MVP does not require ... AI chatbots, or real-time WebSocket
delivery". The web client polls an open thread every 10 s and the organizer's
inbox every 15 s; a resolved thread stops polling entirely.

### Suspension

`POST /admin/events/{id}/suspend` needs the `platform_admin` role, which has no
self-service route. Grant it deliberately:

```sql
INSERT INTO user_roles (user_id, role)
SELECT id, 'platform_admin' FROM users WHERE email = 'admin@biletflow.kz';
```

Suspension is a distinct `event_status` value, not a flag — cancelling is the
organizer calling their event off, suspending is the platform stepping in, and
only the platform can undo it. Lifting it returns the event to `unpublished`, so
the organizer must consciously publish again.

What it does and does not do:

| | Behaviour |
| --- | --- |
| Checkout | **403 `event_suspended`**, checked before any inventory is touched |
| Public page | Still resolves, with `suspended: true` so the banner shows |
| Existing tickets | Stay valid; holders can still be checked in at the gate |
| Organizer | Sees a banner on their own event page; cannot lift it |

Leaving existing tickets valid is deliberate: stranding people who already paid
is not the remedy for an organizer's misconduct.


## Analytics and duplication (Phase 9)

### The figures are computed, never accumulated

There is no `tickets_sold_total` column to drift. Every number in
`GET /events/{id}/analytics` is a `count` or `sum` over the rows that caused it,
evaluated when you ask:

| Reported | Comes from |
| --- | --- |
| `tickets_sold` | `tickets` in `valid` or `checked_in` |
| `gross_revenue_kzt` | `sum(total_kzt)` over counted orders |
| `discounts_kzt` | `sum(discount_kzt)` over the same orders |
| `checked_in` | `tickets` in `checked_in` |
| `percentage_sold` | sold ÷ `sum(quantity_total)` of the ticket types |
| `check_in_percentage` | checked in ÷ sold |

An order counts once it has been paid for and keeps counting if it is later
refunded:

```go
const countedOrderStatuses = `('paid', 'completed', 'refunded', 'partially_refunded')`
```

A refund reverses money, not history — the sale happened, and an organizer
looking at last month's event should see what actually occurred. Refunded
*tickets*, however, leave `valid`, so they stop counting as sold. Pending and
abandoned carts never count at all.

Money crosses the wire as a decimal string, cast in SQL rather than in Go:

```sql
COALESCE(sum(o.total_kzt), 0)::numeric(14,2)::text
```

Without the `numeric(14,2)` step an empty event reports `"0"` while a busy one
reports `"20000.00"`, and the client has to guess. Both are now `"0.00"` shaped.

Percentages are rounded to one decimal place server-side so every client shows
the same number.

### Filters

`from` and `to` accept either `2006-01-02` or a full RFC 3339 timestamp; a plain
`to` date is treated as the end of that day rather than its first instant, which
is what someone typing "to 31 March" means. `ticket_type_id` narrows to one tier.

### The history cannot be rewritten

`GET /events/{id}/timeline` reads `audit_logs`, which carries a trigger that
rejects `UPDATE` and `DELETE`. An entry is therefore evidence, not a note.

### Duplication copies configuration, not consequences

`POST /events/{id}/duplicate` runs exactly two inserts: the event, then its
ticket types. Orders, tickets, payments, check-in records and support cases are
never read, so there is no path by which last year's sales could appear against
this year's event.

| Copied | Deliberately not copied |
| --- | --- |
| Title (suffixed `(copy)`), description, category | Orders, tickets, payments |
| Venue, address, timezone, capacity | Check-in records, staff assignments |
| Every ticket type with its price and quantity | Support cases |
| — | Campaigns and promo codes |

The copy is always a `draft` regardless of the original's status, its ticket
types start at `quantity_sold = 0`, and `duplicated_from_event_id` records the
origin.

Campaigns are left behind because a promo code is globally unique — copying
`SPRING20` into a second event would either collide or silently rename itself,
and both are worse than the organizer creating the code they meant. Overrides
(`title`, `starts_at`, `ends_at`, …) may be passed in the body to skip the edit
that usually follows.

A slug collision on the copy is retried inside a savepoint rather than guessed
in advance, so two organizers duplicating at once cannot deadlock on the name.

### Lifecycle is derived

`Lifecycle(event, now)` returns `cancelled`, `suspended`, `draft`, `completed`,
`active` or `upcoming`, checked in that order, and is recomputed on every read.
Nothing stores it, so an event cannot be found sitting in "Upcoming" a week
after it finished. `GET /events/mine?lifecycle=…` filters on the derived value
after the query rather than in SQL — an organizer's own list is small, and doing
it honestly beats duplicating the date logic in two places.


## Refunds, activation and notifications (Phase 10)

### A refund moves everything or nothing

`POST /orders/{id}/refund` is one transaction. A refund that voided the tickets
but left the money marked as taken - or the reverse - is precisely the
inconsistency a transaction exists to prevent:

| Written | What it does |
| --- | --- |
| `orders` | `status = 'refunded'`, `refunded_kzt = total_kzt` |
| `payments` | the succeeded payment becomes `refunded` |
| `refunds` | a `succeeded`, `is_simulated` record of the reversal |
| `tickets` | every live ticket becomes `refunded` |
| `ticket_types` | `quantity_sold` down, `quantity_refunded` up |
| `audit_logs` | an `order.refunded` entry that cannot be edited |

The order row is locked with `FOR UPDATE` first, so two organizers clicking
Refund at the same moment do not both succeed: the second waits, sees
`refunded`, and is told so.

`refunded_kzt` is set from `total_kzt` **in SQL** rather than from a value this
process computed, so `orders_refund_not_above_total_chk` cannot be tripped by a
rounding difference between Go and PostgreSQL.

### Why a refunded ticket stops scanning

Nothing was added to the gate. Phase 6 already refuses a ticket whose status is
`cancelled` or `refunded`, so the single `UPDATE tickets SET status =
'refunded'` is what closes the door. The scanner shows its red screen with
*"This ticket is refunded and cannot be used for entry"*.

A **checked-in** ticket is voided too - somebody who attended and was then
refunded should not be readmitted on a second scan - and `checked_in_at` is
cleared alongside the status, because `tickets_checked_in_consistency_chk`
holds that the column is set if and only if the ticket is currently checked in.
The attendance itself is not erased: the `check_in_records` row stays exactly
as it was. A refund voids a ticket; it does not rewrite what happened on the
night.

### The seat goes back on sale

`quantity_sold` tracks live tickets, not tickets ever issued, so a refund
decrements it and increments `quantity_refunded`. Without that, an event would
sell out against tickets that no longer admit anybody. The invariant
`quantity_sold = count(tickets in 'valid' or 'checked_in')` is pinned by a test.

Only **full** refunds are supported, which is what SRS 4.9 asks for
("initiate full refunds"). Partial refunds need a per-ticket selection and a
proration policy; the schema already carries `refunded_kzt` and a
`partially_refunded` status, so that stays open without a migration.

### Activation is checked inside the checkout transaction

SRS 4.5 says paid tickets shall not be purchasable before activation. The check
sits in `CheckoutStore.Checkout`, next to the prices it depends on, rather than
in the handler:

```go
if isPositiveAmount(locked[item.TicketTypeID].priceKZT) { paid = true }
```

Reading the activation in the handler would leave a window where an admin
suspends paid sales between the check and the sale. A basket with even one paid
line is refused as a whole - a partial order is a surprise nobody asked for.

Free tickets are never gated. A mixed event keeps selling its free tier while
its paid tier waits, because activation exists to gate money, not registration.

### The checklist

Four steps, each stamped with the moment it was completed:

| Step | Column | Stands in for |
| --- | --- | --- |
| `identity` | `identity_verified_at` | a document or company-registry check |
| `payout` | `payout_verified_at` | verifying a payout destination |
| `terms` | `terms_accepted_at` | accepting the seller terms |
| `fee` | `activation_payment_id` | the activation fee payment |

They are four flags on one endpoint rather than four endpoints: an organizer
ticking the last box expects sales to open in that same click, and splitting the
final transition out would leave a window where the checklist reads complete
while sales are still shut. The transition itself is decided **in SQL** against
the row as it stands, and `activations_checklist_chk` enforces the same rule as
a constraint - an `active` row with a gap in its checklist cannot exist.

Re-submitting is safe: `COALESCE` keeps the first completion time, and the fee
payment is written only when one does not already exist.

The fee is `5000.00 KZT`, recorded as a `payments` row with
`purpose = 'paid_sales_activation'` and `is_simulated = true`. Nothing is
charged to anybody.

### Suspending paid sales

`POST /admin/events/{id}/paid-sales/suspend` is narrower than the Phase 8 event
suspension: it stops the money while leaving free registration working, which is
the proportionate response when the concern is about payments rather than about
the event. An organizer cannot re-activate their way out of it; only the
platform can lift it, and lifting re-checks the whole checklist rather than
trusting the old status.

### Notifications are an outbox, not a print statement

`internal/email` has a `Sender` interface, a `ConsoleSender` that prints, and a
`Mailer` that dispatches without making the caller wait. Swapping in SMTP later
changes that package and nothing else.

The order of operations matters:

1. the checkout or refund commits;
2. the response is written;
3. a `notifications` row is recorded as `pending`;
4. the message is dispatched on a goroutine;
5. the row is marked `sent` or `failed`.

The row is written **before** delivery, so a notification that was attempted is
visible even if delivery then failed - that ordering is what makes the table an
outbox rather than a log of successes. A notification failure never fails the
purchase: the attendee has paid and the tickets exist.

The dispatch context is deliberately *not* the request's. The request is
finished by then, and a cancelled context would abort the very delivery it was
supposed to carry. `Mailer.Wait()` drains in-flight sends on shutdown, and is
what lets a test assert on an asynchronous send instead of sleeping and hoping.

What the console prints:

```
==========================================================================
  MOCK EMAIL - simulated delivery, nothing left this machine
--------------------------------------------------------------------------
  To:      aigerim@biletflow.test
  Subject: Your Tickets to Astana Winter Gala 2026
  Type:    order.confirmation
--------------------------------------------------------------------------
  Hi Aigerim,

  Your order BF-SHK3LHIGMO is confirmed. Your 2 tickets are attached below.
  ...
    BF-TKT-GUIFHX44W4  Gala Seat
      http://localhost:8080/api/v1/tickets/63db.../pdf
==========================================================================
```

The whole block is assembled before it is written, under a mutex, so two
concurrent sends cannot interleave into something unreadable exactly when you
need to read it. `API_BASE_URL` decides the download links, because a relative
path is no use in an inbox.


## Account recovery, the portal and uploads (Phase 12)

### Tokens are stored as hashes, never as themselves

`user_tokens` holds a SHA-256 of what was emailed, not the token. A leaked
database backup therefore contains no working password-reset links - only the
inbox does. Verification and reset share one table because they share every
property that matters: single use, expiring, delivered out of band. Only the
purpose and the lifetime differ.

| | Lifetime | Why |
| --- | --- | --- |
| `password_reset` | 1 hour | It is a password in disguise: whoever holds it can take the account. |
| `email_verification` | 72 hours | It only claims an address, so it can survive a weekend in an inbox. |

Issuing supersedes: clicking "forgot password" three times leaves one working
key in the inbox, not three. Consuming a reset also invalidates every other
outstanding token for that account - somebody recovering an account they lost
control of should not leave the attacker holding a live verification link.

The whole check happens in one `UPDATE ... WHERE consumed_at IS NULL AND
expires_at > now() RETURNING user_id`, so two concurrent requests cannot both
redeem the same token.

### Why "no account with that email" is never said

`POST /auth/password-reset/request` returns the same 202 and the same body
whether or not the address exists. Saying otherwise would turn the form into a
way of testing which addresses are registered - a privacy leak dressed as
helpfulness. The store still reports `ErrNotFound` honestly; the handler is
what declines to pass it on.

Unknown, expired and already-used tokens share one error code for the same
reason: distinguishing them would confirm that a guessed token was once real.

Resetting returns **no session**. Proving control of an inbox is not the same
as intending to sign in on this device, and issuing a token here would let a
stolen reset link skip the login screen entirely.

### Uploads trust the bytes, not the client

`POST /uploads/images` takes a multipart file and:

1. caps the body with `http.MaxBytesReader`, so an oversized upload fails while
   reading rather than after being buffered whole;
2. sniffs the type from the file's own first 512 bytes with
   `http.DetectContentType` - the `Content-Type` header and the filename are
   both supplied by the client and neither is evidence of anything;
3. stores under a **random** name, never the uploaded one. A client-supplied
   filename is a path-traversal vector and a way to overwrite somebody else's
   banner, and neither is worth keeping the original name for.

Files land in `UPLOAD_DIR` (default `./data/uploads`) and are served from
`/uploads/{file}`. Swapping local disk for S3 changes this handler and the
static route, and nothing else. The serving route refuses anything containing a
path separator and anything that is not a plain file, so it cannot be walked
into an index of every organizer's banner.

Reading is public and uploading is not: a banner is shown to attendees who are
not signed in.

### The attendee search is not a skeleton key

`GET /events/{id}/attendees` deliberately does **not** return QR tokens. Staff
searching by name are standing next to the person; handing every door device a
working admission credential for every attendee would make the QR pointless.
Admission goes by ticket id instead, which only an already-authorised device
could have obtained from the search.

`POST /events/{id}/check-in/manual` runs the identical transaction as a scan -
the same row lock, the same `check_in_one_active_per_ticket_uidx`, the same
duplicate handling. Only the lookup differs:

```go
func (s *CheckInStore) CheckIn(...)           { return s.admit(ctx, eventID, "t.qr_token = $1", token, ...) }
func (s *CheckInStore) CheckInByTicketID(...) { return s.admit(ctx, eventID, "t.id = $1", ticketID, ...) }
```

The predicate is one of two compile-time literals and the value is always a
bound parameter. The device label records that the admission was typed rather
than scanned, which is worth being able to audit later.

Both endpoints go through the same authorisation as the scanner: the organizer,
or somebody assigned to work that event's door.

### The portal is one search, not four

SRS 4.12 asks admins to search users, events, orders **and** payments.
`GET /admin/search?q=` searches all four with one term, because an admin handed
"aigerim@…" rarely knows which kind of record it belongs to, and making them
pick the right tab first is a worse tool. An empty query is legitimate and
returns the most recent rows of each kind, so the portal opens on something
useful rather than four empty tables.

The report is CSV because the person asking for an "operational report" is
going to open it in a spreadsheet. It is written with `encoding/csv`, so an
event called `Jazz, Blues & "More"` stays one column, and served `no-store` -
the next download is meant to show what has changed since this one.

### Notification triggers

| Trigger | Type | Sent to |
| --- | --- | --- |
| Checkout completes | `order.confirmation` | buyer |
| Refund completes | `refund.completed` | buyer |
| Password reset requested | `account.password_reset` | account |
| Verification requested | `account.email_verification` | account |
| Event cancelled | `event.cancelled` | every ticket holder |
| Support message posted | `support.new_message` | the other side |

A cancellation sends **one message per order**, not per ticket: somebody who
bought four seats wants one email. An internal support note notifies nobody -
that is the entire point of the checkbox.


## Reservations, fees and calendars (Phase 13)

### Buying a ticket has one implementation, not two

`POST /checkout` used to hold the whole purchase. It is now composed:

```go
held, err := s.hold(ctx, tx, ...)     // reserve
result, err := s.confirm(ctx, tx, ...) // sell
```

Both halves run in the caller's transaction, so a one-shot purchase is a hold
that is paid for immediately - not a second copy of the inventory arithmetic
that can drift from the first.

### Reserved, then sold

SRS 4.6 asks that "ticket inventory shall be temporarily reserved during
checkout". A basket increments `quantity_reserved`; confirming moves the count
to `quantity_sold` in a single statement:

```sql
UPDATE ticket_types
   SET quantity_reserved = GREATEST(quantity_reserved - $2::int, 0),
       quantity_sold     = quantity_sold + $2::int
 WHERE id = $1
```

Both counters move together, so `quantity_sold + quantity_reserved` never
changes mid-conversion and `ticket_types_inventory_chk` cannot be tripped
halfway through.

### Three ways a basket is released

| When | How |
| --- | --- |
| The attendee cancels | `DELETE /orders/{id}/hold` |
| Somebody else shops that event | swept inside the next `hold` transaction |
| Nobody comes back | a background sweeper, once a minute |

The second is what makes correctness independent of the timer: a stale basket
can never be the reason a real sale is refused, even with the sweeper wedged.

Reading an expired basket releases it too - and does so in **its own committed
transaction**. Every path out of that branch returns an error, and an error
rolls the transaction back, so a release written inline would be quietly
undone. That was not hypothetical: it showed up as a smoke check that passed on
the second run and failed on the first.

### The processing charge

```sql
CASE WHEN subtotal_kzt - discount_kzt > 0
     THEN round((subtotal_kzt - discount_kzt) * $2::numeric / 100 + $3::numeric, 2)
     ELSE 0 END
```

Evaluated by PostgreSQL, never in Go - a percentage of a price is exactly the
arithmetic float64 gets wrong. Charged on the **discounted** subtotal, because
a fee is taken on what actually moves. Zero when nothing does, which is SRS
3.3's "free events: no platform fee".

The fee is added to the attendee's total and never reaches the organizer: the
organizer's proceeds are the ticket price, and the charge for moving the money
is not theirs to keep. **Every historical total in this repo moved by 3.5% when
this landed** - Phase 4's "10 000 KZT" order is now 10 350. Set
`PROCESSING_FEE_PERCENT=0` to restore the old figures.

### Unicode tickets

`AddUTF8FontFromBytes` with an embedded DejaVu Sans Condensed, so the font
travels inside the binary and inside the PDF. Two consequences worth knowing:

- fpdf writes UTF-8 text as **UTF-16BE literal strings**, so the test
  extractor decodes that rather than grepping ASCII;
- `truncate` counts runes. A byte-based cut would slice a two-byte Cyrillic
  letter in half and put a replacement character on somebody's ticket.

### Calendar files

RFC 5545 by hand: one VEVENT, CRLF throughout (Outlook rejects bare newlines),
commas and semicolons escaped (an unescaped comma in a venue name silently
splits the property and the address vanishes), and lines folded at 75 octets
**without splitting a multi-byte character**.

`DTSTART;TZID=Asia/Almaty` with a VTIMEZONE block, not UTC: an attendee who
travels should still see the event at the hour it happens at the venue. The UID
is the event id, so re-downloading replaces the entry; a cancelled event that
was once published stays downloadable, because refusing it would leave a stale
entry in somebody's calendar with no way to correct it.
