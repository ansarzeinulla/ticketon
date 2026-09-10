# Jira setup and backlog

## Project

- **Name:** BiletFlow · **Key:** `BF` · **Template:** Scrum
- **Sprints:** seven, matching the report periods (`Sprint 1` … `Sprint 7`)
- **Board columns:** `To Do` → `In Progress` → `In Review` → `Done`
- **Issue types:** Epic, Story, Task, Bug
- **Definition of done:** merged to `main`, CI green, the phase spec that covers
  it passes, and the Jira issue links the merging PR.

Connect the GitHub repo to Jira (*Apps → GitHub for Jira*) so a commit whose
message starts with `BF-nn` attaches itself to the issue. That single step is
what makes the weekly evidence in the reports real rather than hand-written.

## Epics — create all ten in week 1

| Epic | Title | Owner | SRS |
| --- | --- | --- | --- |
| `BF-E1` | Foundations, database and CI | A | §8, §9 |
| `BF-E2` | Identity, accounts and permissions | A | 4.1 |
| `BF-E3` | Events and ticket types | B | 4.2, 4.3 |
| `BF-E4` | Checkout, inventory and payments | B | 4.4–4.6 |
| `BF-E5` | Digital tickets and delivery | B / C | 4.7 |
| `BF-E6` | Check-in and the scanner app | E | 4.8 |
| `BF-E7` | Orders, cancellations and refunds | B | 4.9 |
| `BF-E8` | Support desk | D | 4.13 |
| `BF-E9` | Analytics, history and audit | D / A | 4.15, 4.16 |
| `BF-E10` | Administration, promo campaigns, release | A / B | 4.12, 4.14 |

## Cadence

- **Create** an issue the moment work is foreseen — at the latest, the Monday of
  the week it is scheduled. Never create an issue after doing the work.
- **Move** it the day the state changes. An issue that jumps `To Do` → `Done` on
  a Friday tells the grader nothing about how the fortnight actually went.
- **Close** an epic only when its last child closes; note the closure in the
  report for that period.
- **Bugs** get their own issue (`BF-B*`) linked to the story that introduced
  them. Do not silently fold a fix into an unrelated commit.

## Sprint 1 — Foundations (31 Aug – 13 Sep)

| Key | Summary | Epic | Owner |
| --- | --- | --- | --- |
| BF-1 | Add Postgres service and extensions | E1 | A |
| BF-2 | Add core schema: users, events, ticket types | E1 | A |
| BF-3 | Scaffold the Go API and configuration | E1 | B |
| BF-4 | Add the JSON error envelope and codes | E1 | B |
| BF-5 | Scaffold the Next.js application | E1 | C |
| BF-6 | Add design tokens and UI primitives | E1 | C |
| BF-7 | Add the signed-in shell and header | E1 | D |
| BF-8 | Add Makefile targets | E1 | E |
| BF-9 | Add CI: build, vet, test | E1 | E |
| BF-10 | Add ticketing and audit tables | E1 | A |
| BF-11 | Add the schema test suite | E1 | A |
| BF-12 | Add demo seed data | E1 | A |
| BF-13 | Add the pgx pool and store base | E1 | B |
| BF-14 | Add the Phase 1 test specification | E1 | E |

## Sprint 2 — Identity and events (14 – 27 Sep)

| Key | Summary | Epic | Owner |
| --- | --- | --- | --- |
| BF-15 | Hash passwords with bcrypt | E2 | A |
| BF-16 | Issue and verify JWTs | E2 | A |
| BF-17 | Add register and login endpoints | E2 | A |
| BF-18 | Create and read events | E3 | B |
| BF-19 | Derive unique slugs, transliterating Cyrillic | E3 | B |
| BF-20 | Wire sign-in and registration | E2 | C |
| BF-21 | Add the event create form | E3 | D |
| BF-22 | Add CRUD and constraint tests | E1 | E |
| BF-23 | Add password reset with hashed tokens | E2 | A |
| BF-24 | Add email verification | E2 | A |
| BF-25 | Add organizer profiles | E2 | A |
| BF-26 | Manage ticket types | E3 | B |
| BF-27 | Add event lifecycle transitions | E3 | B |
| BF-28 | Add the public catalogue and event page | E3 | C |
| BF-29 | Add the event dashboard with lifecycle actions | E3 | D |
| BF-30 | Add the Phase 2 test specification | E3 | E |

## Sprint 3 — Checkout (28 Sep – 11 Oct)

| Key | Summary | Epic | Owner |
| --- | --- | --- | --- |
| BF-31 | Make audit_logs append-only | E9 | A |
| BF-32 | Enforce role permissions | E2 | A |
| BF-33 | Lock ticket_types rows during checkout | E4 | B |
| BF-34 | Issue tickets in one transaction | E4 | B |
| BF-35 | Add the free registration path | E4 | B |
| BF-36 | Add the ticket selector and checkout dialog | E4 | C |
| BF-37 | List orders and attendees for an organizer | E7 | D |
| BF-38 | Add the Phase 3 test specification | E4 | E |
| BF-39 | Prove checkout cannot oversell | E4 | E |
| BF-40 | Record and send notifications | E2 | A |
| BF-41 | Gate paid sales behind activation | E4 | B |
| BF-42 | Add simulated payment with a decline path | E4 | B |
| BF-43 | Add the order confirmation page | E5 | C |
| BF-44 | Add the paid-sales activation checklist | E4 | D |
| BF-45 | Add business-rule tests | E1 | E |
| BF-46 | Record Phase 3 results | E4 | E |

## Sprint 4 — Tickets and the gate (12 – 25 Oct)

| Key | Summary | Epic | Owner |
| --- | --- | --- | --- |
| BF-47 | Send the order confirmation with tickets | E5 | A |
| BF-48 | Generate admission QR tokens | E5 | B |
| BF-49 | Render an A4 PDF ticket | E5 | B |
| BF-50 | Embed a Unicode font for Cyrillic | E5 | B |
| BF-51 | Show QR tickets and the PDF download | E5 | C |
| BF-52 | Scaffold the Expo scanner app | E6 | E |
| BF-53 | Sign in and list assigned events | E6 | E |
| BF-54 | Assign and revoke gate staff | E6 | A |
| BF-55 | Reserve stock with 15-minute holds | E4 | B |
| BF-56 | Charge a processing fee | E4 | B |
| BF-57 | Add the interactive seat map *(bonus)* | E4 | C |
| BF-58 | Show the event activity timeline | E9 | D |
| BF-59 | Scan and admit a ticket | E6 | E |
| BF-60 | Refuse a second admission | E6 | E |
| BF-61 | Reject campaign codes at the gate | E6 | E |
| BF-62 | Add the Phase 4 and 5 specifications | E6 | E |

## Sprint 5 — Refunds and support (26 Oct – 8 Nov)

| Key | Summary | Epic | Owner |
| --- | --- | --- | --- |
| BF-63 | Add unified admin search | E10 | A |
| BF-64 | Suspend users and events | E10 | A |
| BF-65 | Refund an order atomically | E7 | B |
| BF-66 | Restock refunded inventory | E7 | B |
| BF-67 | Void refunded tickets at the gate | E7 | B |
| BF-68 | Let an attendee find their own orders | E7 | C |
| BF-69 | Open a support case with context | E8 | D |
| BF-70 | Add the organizer inbox | E8 | D |
| BF-71 | Search attendees by name | E6 | E |
| BF-72 | Undo an accidental check-in | E6 | E |
| BF-73 | Add the report queue | E10 | A |
| BF-74 | Make the activation fee configurable | E10 | A |
| BF-75 | Cancel a free registration | E7 | B |
| BF-76 | Add staff-only notes and case status | E8 | D |
| BF-77 | Add the Phase 6 specification | E7 | E |
| BF-78 | Record Phase 4–6 results | E6 | E |

## Sprint 6 — Analytics and campaigns (9 – 22 Nov)

| Key | Summary | Epic | Owner |
| --- | --- | --- | --- |
| BF-79 | Filter the activity timeline | E9 | A |
| BF-80 | Create campaigns and promo codes | E10 | B |
| BF-81 | Issue campaign QR codes | E10 | B |
| BF-82 | Apply and cap discounts atomically | E10 | B |
| BF-83 | Apply a promo code at checkout | E10 | C |
| BF-84 | Compute analytics from operational rows | E9 | D |
| BF-85 | Show capacity, revenue and attendance | E9 | D |
| BF-86 | Break sales down by tier and campaign | E9 | D |
| BF-87 | Add the Phase 7 specification | E9 | E |
| BF-88 | Export an operational CSV report | E10 | A |
| BF-89 | Duplicate an event without its transactions | E9 | B |
| BF-90 | Add Kazakh and Russian dictionaries | E5 | C |
| BF-91 | Add the language switcher | E5 | C |
| BF-92 | Add the platform admin portal | E10 | D |
| BF-93 | Add the Phase 8 and 9 specifications | E8 | E |
| BF-94 | Record Phase 7–9 results | E9 | E |

## Sprint 7 — Hardening and release (23 – 28 Nov)

| Key | Summary | Epic | Owner |
| --- | --- | --- | --- |
| BF-95 | Count text limits in characters, not bytes | E10 | A |
| BF-96 | Report fees and estimated payout | E9 | B |
| BF-97 | Add the Phase 10 specification | E10 | E |
| BF-98 | Close the i18n gaps on customer-facing pages | E5 | C |
| BF-99 | Show reserved stock and my-tickets | E9 | D |
| BF-100 | Full regression sweep, Phases 1–10 | E10 | E |
| BF-101 | Fix defects found in the sweep | E10 | all |
| BF-102 | Update the README and API docs | E10 | A |
| BF-103 | Demo rehearsal on a fresh database | E10 | all |
| BF-104 | Tag v1.0.0 | E10 | E |
