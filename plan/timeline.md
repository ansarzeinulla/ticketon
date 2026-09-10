# Week-by-week timeline

Thirteen weeks, Mon 31 Aug – Sat 28 Nov 2026. Seven sprints; each sprint is two
weeks and ends on a report, except the last which is one week.

**How to read a week.** Each week lists, per person, the files that appear for
the first time and the commits that carry them. "Add" means the file is created
that week; "extend" means it already exists. Jira issues open on the Monday of
the week they are listed and close on the Friday unless stated otherwise.

**The weekly rhythm, every week without exception:**

| Day | What happens |
| --- | --- |
| Mon | 20-min check-in. Pull issues to *In Progress*. First commit of the week lands. |
| Tue–Thu | Work commits, at least one per person per day worked. PRs opened. |
| Thu | Peer review. Issues to *In Review*. |
| Fri | PRs merged, issues to *Done*, `main` green (`make api-check && make web-check`). |
| Sun | Report weeks only: Student A submits by 23:59. |

---

## Sprint 1 — Weeks 1–2 (Mon 31 Aug – Sun 13 Sep) → Report 1

**Goal:** the project runs on every machine, and the database is real.

**Jira at sprint start:** create project `BF`, board, and all ten epics
(`jira-backlog.md`). Open `BF-1` … `BF-14`. Sprint 1 is the only sprint where
the whole backlog is created at once; later sprints only refine it.

### Week 1 (31 Aug – 6 Sep)

- **A** — add `docker-compose.yml`, `db/init/01_extensions.sql`,
  `db/init/02_schema.sql` (users, events, ticket_types, orders first).
  Commits: `BF-1 Add Postgres service and extensions`,
  `BF-2 Add core schema: users, events, ticket types`.
- **B** — add `api/go.mod`, `api/cmd/api/main.go`, `api/internal/config/`,
  `api/internal/httpx/`. Commits: `BF-3 Scaffold the Go API and config`,
  `BF-4 Add JSON envelope and error codes`.
- **C** — add `web/` via `create-next-app`, `web/src/app/layout.tsx`, design
  tokens. Commits: `BF-5 Scaffold Next.js app`, `BF-6 Add design tokens and UI primitives`.
- **D** — add `web/src/app/(app)/layout.tsx`, `web/src/components/site-header.tsx`.
  Commits: `BF-7 Add signed-in shell and header`.
- **E** — add `Makefile`, `README.md`, `.github/workflows/ci.yml`.
  Commits: `BF-8 Add Makefile targets`, `BF-9 Add CI: build, vet, test`.

### Week 2 (7 Sep – 13 Sep)

- **A** — extend `02_schema.sql` (tickets, payments, attendees, audit_logs);
  add `db/tests/` suite and `db/seed/01_demo_data.sql`.
  Commits: `BF-10 Add ticketing and audit tables`, `BF-11 Add schema test suite`,
  `BF-12 Add demo seed data`.
- **B** — add `api/internal/store/store.go`, `api/internal/database/`.
  Commit: `BF-13 Add pgx pool and store base`.
- **C** — add `web/src/app/(auth)/login/page.tsx`, `register/page.tsx` (static).
- **D** — add `web/src/app/(app)/dashboard/page.tsx` (empty state).
- **E** — add `1.md` (Phase 1 test spec), `docs/biletflow-api.postman_collection.json`.
  Commit: `BF-14 Add Phase 1 test specification`.

**Close by Fri 11 Sep:** `BF-1` … `BF-14`. **Report 1 due Sun 13 Sep.**

---

## Sprint 2 — Weeks 3–4 (Mon 14 Sep – Sun 27 Sep) → Report 2

**Goal:** a person can register, sign in, and an organizer can create and
publish an event.

**Jira:** open `BF-15` … `BF-30`.

### Week 3 (14 Sep – 20 Sep)

- **A** — add `api/internal/auth/password.go`, `token.go`,
  `api/internal/store/users.go`, `tokens.go`, `api/internal/api/handlers_auth.go`,
  `middleware.go`. Commits: `BF-15 Hash passwords with bcrypt`,
  `BF-16 Issue and verify JWTs`, `BF-17 Add register and login endpoints`.
- **B** — add `api/internal/store/events.go`, `slug.go`,
  `api/internal/api/handlers_events.go`. Commits: `BF-18 Create and read events`,
  `BF-19 Derive unique slugs, transliterating Cyrillic`.
- **C** — wire `(auth)` pages to the API; add `web/src/lib/api.ts`,
  `web/src/lib/auth-context.tsx`. Commit: `BF-20 Wire sign-in and registration`.
- **D** — add `web/src/app/(app)/events/new/page.tsx`,
  `web/src/components/event-form-fields.tsx`. Commit: `BF-21 Add event create form`.
- **E** — add `db/tests/03_user_crud.sql`, `04_constraints.sql`; run Phase 1 spec.
  Commit: `BF-22 Add CRUD and constraint tests`.

### Week 4 (21 Sep – 27 Sep)

- **A** — add password reset and email verification: `api/internal/email/`,
  `handlers_account.go`, `store/profiles.go`.
  Commits: `BF-23 Add password reset with hashed tokens`,
  `BF-24 Add email verification`, `BF-25 Add organizer profiles`.
- **B** — add `store/ticket_types.go`, `handlers_ticket_types.go`; event
  lifecycle (publish/unpublish/cancel).
  Commits: `BF-26 Manage ticket types`, `BF-27 Add event lifecycle transitions`.
- **C** — add `web/src/app/events/page.tsx` (catalogue),
  `web/src/app/events/[slug]/page.tsx`. Commit: `BF-28 Add public catalogue and event page`.
- **D** — add `web/src/app/(app)/dashboard/events/[id]/page.tsx` and lifecycle
  buttons. Commit: `BF-29 Add event dashboard with lifecycle actions`.
- **E** — add `2.md`; run Phases 1–2. Commit: `BF-30 Add Phase 2 test specification`.

**Close by Fri 25 Sep:** `BF-15` … `BF-30`. **Report 2 due Sun 27 Sep.**

---

## Sprint 3 — Weeks 5–6 (Mon 28 Sep – Sun 11 Oct) → Report 3

**Goal:** money. Someone can buy a ticket, and two people cannot buy the same one.

**Jira:** open `BF-31` … `BF-46`. `BF-33` (inventory locking) is the sprint's
risk item — flag it in the report if it slips.

### Week 5 (28 Sep – 4 Oct)

- **A** — add `store/audit.go` and the append-only trigger in `02_schema.sql`;
  role checks in `middleware.go`.
  Commits: `BF-31 Make audit_logs append-only`, `BF-32 Enforce role permissions`.
- **B** — add `store/checkout.go`, `handlers_checkout.go`.
  Commits: `BF-33 Lock ticket_types rows during checkout`,
  `BF-34 Issue tickets in one transaction`, `BF-35 Add free registration path`.
- **C** — add `web/src/components/ticket-selector.tsx`, `checkout-dialog.tsx`.
  Commit: `BF-36 Add ticket selector and checkout dialog`.
- **D** — add `web/src/components/attendee-list.tsx`, `order-manager.tsx`.
  Commit: `BF-37 List orders and attendees for an organizer`.
- **E** — add `3.md`; add the concurrency test
  `TestCheckoutDoesNotOversellUnderConcurrency`.
  Commits: `BF-38 Add Phase 3 spec`, `BF-39 Prove checkout cannot oversell`.

### Week 6 (5 Oct – 11 Oct)

- **A** — add `store/notifications.go`, `api/internal/api/notify.go`.
  Commit: `BF-40 Record and send notifications`.
- **B** — add `store/activation.go`, `handlers_activation.go`; simulated payments.
  Commits: `BF-41 Gate paid sales behind activation`,
  `BF-42 Add simulated payment with decline path`.
- **C** — add `web/src/app/orders/[id]/page.tsx`. Commit: `BF-43 Add order confirmation page`.
- **D** — add `web/src/components/activation-checklist.tsx`.
  Commit: `BF-44 Add paid-sales activation checklist`.
- **E** — run Phase 3; add `db/tests/05_business_rules.sql`.
  Commits: `BF-45 Add business-rule tests`, `BF-46 Record Phase 3 results`.

**Close by Fri 9 Oct:** `BF-31` … `BF-46`. **Report 3 due Sun 11 Oct.**

---

## Sprint 4 — Weeks 7–8 (Mon 12 Oct – Sun 25 Oct) → Report 4

**Goal:** the ticket exists as a printable artefact, and the door scanner works.

**Jira:** open `BF-47` … `BF-62`.

### Week 7 (12 Oct – 18 Oct)

- **A** — order confirmation email with ticket links.
  Commit: `BF-47 Send order confirmation with tickets`.
- **B** — add `api/internal/ticketpdf/` (`render.go`, `qr.go`, `fonts.go`),
  `store/tickets.go`, `handlers_tickets.go`.
  Commits: `BF-48 Generate admission QR tokens`,
  `BF-49 Render an A4 PDF ticket`, `BF-50 Embed a Unicode font for Cyrillic`.
- **C** — add `web/src/components/ticket-card.tsx`.
  Commit: `BF-51 Show QR tickets and PDF download`.
- **D** — extend the dashboard with ticket status columns.
- **E** — add `mobile/` (Expo scaffold), `mobile/app/login.tsx`, `events.tsx`.
  Commits: `BF-52 Scaffold the Expo scanner app`, `BF-53 Sign in and list assigned events`.

### Week 8 (19 Oct – 25 Oct)

- **A** — add `store/staff.go`, `handlers_staff.go` (gate staff delegation).
  Commit: `BF-54 Assign and revoke gate staff`.
- **B** — seat holds: `store/holds.go`, `handlers_holds.go`, processing fee.
  Commits: `BF-55 Reserve stock with 15-minute holds`, `BF-56 Charge a processing fee`.
- **C** — add `web/src/components/seat-map.tsx`, `seat-checkout.tsx`.
  Commit: `BF-57 Add interactive seat map`.
- **D** — add `web/src/components/event-timeline.tsx`.
  Commit: `BF-58 Show the event activity timeline`.
- **E** — add `mobile/app/scan/[eventId].tsx`, `api/.../handlers_checkin.go`,
  `store/checkin.go`; add `4.md`, `5.md`.
  Commits: `BF-59 Scan and admit a ticket`,
  `BF-60 Refuse a second admission`, `BF-61 Reject campaign codes at the gate`,
  `BF-62 Add Phase 4 and 5 specs`.

**Close by Fri 23 Oct:** `BF-47` … `BF-62`. **Report 4 due Sun 25 Oct.**

---

## Sprint 5 — Weeks 9–10 (Mon 26 Oct – Sun 8 Nov) → Report 5

**Goal:** money can go back, and an attendee can ask for help.

**Jira:** open `BF-63` … `BF-78`.

### Week 9 (26 Oct – 1 Nov)

- **A** — add `store/admin.go`, `handlers_admin_portal.go`, `handlers_admin_users.go`.
  Commits: `BF-63 Add unified admin search`, `BF-64 Suspend users and events`.
- **B** — add `store/refunds.go`, `handlers_refunds.go`.
  Commits: `BF-65 Refund an order atomically`, `BF-66 Restock refunded inventory`,
  `BF-67 Void refunded tickets at the gate`.
- **C** — add `web/src/app/(app)/orders/page.tsx` (my tickets).
  Commit: `BF-68 Let an attendee find their own orders`.
- **D** — add `store/support.go`, `handlers_support.go`,
  `web/src/components/order-support.tsx`.
  Commits: `BF-69 Open a support case with context`, `BF-70 Add the organizer inbox`.
- **E** — add `mobile/app/attendees/[eventId].tsx`; check-in reversal.
  Commits: `BF-71 Search attendees by name`, `BF-72 Undo an accidental check-in`.

### Week 10 (2 Nov – 8 Nov)

- **A** — add `store/moderation.go`, `handlers_moderation.go`, platform settings.
  Commits: `BF-73 Add the report queue`, `BF-74 Make the activation fee configurable`.
- **B** — add `store/cancellations.go`, `handlers_cancellations.go`.
  Commit: `BF-75 Cancel a free registration`.
- **C** — order page refund/cancel states.
- **D** — internal notes, case status, polling.
  Commit: `BF-76 Add staff-only notes and case status`.
- **E** — add `6.md`; run Phases 4–6.
  Commits: `BF-77 Add Phase 6 spec`, `BF-78 Record Phase 4–6 results`.

**Close by Fri 6 Nov:** `BF-63` … `BF-78`. **Report 5 due Sun 8 Nov.**

---

## Sprint 6 — Weeks 11–12 (Mon 9 Nov – Sun 22 Nov) → Report 6

**Goal:** the organizer can see what happened, and campaigns can be run.

**Jira:** open `BF-79` … `BF-94`.

### Week 11 (9 Nov – 15 Nov)

- **A** — timeline endpoint with date and type filters.
  Commit: `BF-79 Filter the activity timeline`.
- **B** — add `store/campaigns.go`, `handlers_campaigns.go`, `handlers_promo.go`.
  Commits: `BF-80 Create campaigns and promo codes`,
  `BF-81 Issue campaign QR codes`, `BF-82 Apply and cap discounts atomically`.
- **C** — add `web/src/components/promo-box.tsx`; campaign links.
  Commit: `BF-83 Apply a promo code at checkout`.
- **D** — add `store/analytics.go`, `handlers_analytics.go`,
  `web/src/components/event-analytics.tsx`.
  Commits: `BF-84 Compute analytics from operational rows`,
  `BF-85 Show capacity, revenue and attendance`, `BF-86 Break sales down by tier and campaign`.
- **E** — add `7.md`; add `mobile/lib/offline*.ts` (bonus, only if ahead).
  Commit: `BF-87 Add Phase 7 spec`.

### Week 12 (16 Nov – 22 Nov)

- **A** — CSV export, `handlers_admin.go`.
  Commit: `BF-88 Export an operational CSV report`.
- **B** — event duplication: `store/duplicate.go`.
  Commit: `BF-89 Duplicate an event without its transactions`.
- **C** — add `web/src/lib/i18n/` (kk, ru, en) and `language-switcher.tsx`.
  Commits: `BF-90 Add Kazakh and Russian dictionaries`, `BF-91 Add the language switcher`.
- **D** — add `web/src/app/(app)/admin/page.tsx`, `admin-portal.tsx`.
  Commit: `BF-92 Add the platform admin portal`.
- **E** — add `8.md`, `9.md`; run Phases 7–9.
  Commits: `BF-93 Add Phase 8 and 9 specs`, `BF-94 Record Phase 7–9 results`.

**Close by Fri 20 Nov:** `BF-79` … `BF-94`. **Report 6 due Sun 22 Nov.**

---

## Sprint 7 — Week 13 (Mon 23 Nov – Sat 28 Nov) → Report 7 (final)

**Goal:** harden, document, tag, demo. No new features after Wednesday.

**Jira:** open `BF-95` … `BF-104`. Everything closes by Fri 27 Nov.

- **Mon 23** — **A** rune-based validation limits and file/image constraints
  (`BF-95`); **B** fees and estimated payout in analytics (`BF-96`);
  **E** `10.md` written (`BF-97`).
- **Tue 24** — **C** i18n gaps closed on every customer-facing page (`BF-98`);
  **D** attendee "my tickets" and reserved-stock figures wired (`BF-99`).
- **Wed 25** — **feature freeze at 18:00.** Full sweep: `make api-check`,
  `make web-check`, `make test`, Phases 1–10 re-run (`BF-100`).
- **Thu 26** — bug fixes only (`BF-101`). **A** updates `README.md` and
  `docs/` (`BF-102`).
- **Fri 27** — demo rehearsal against a fresh `make reset && make seed`
  (`BF-103`). Tag `v1.0.0` (`BF-104`). All issues *Done*.
- **Sat 28** — **Report 7 submitted.** Demo.

---

## If you fall behind

Cut in this order. The first three are explicitly bonus in SRS §8 and cost you
nothing in the base MVP:

1. Offline scanner sync (`mobile/lib/offline*.ts`)
2. Calendar `.ics` export (`api/internal/calendar/`)
3. Interactive seat map (`seat-map.tsx`) — fall back to the quantity stepper
4. GA4 analytics scripts

Never cut: checkout correctness, refunds, the scanner's duplicate-entry guard,
or Kazakh/Russian on the customer-facing pages. Those are required.
