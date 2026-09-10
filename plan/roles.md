# Roles and ownership

Five people, five lanes. The split follows the codebase, so two people rarely
need the same file in the same week, and every SRS area has exactly one owner
who answers for it.

| | Role | Owns | SRS |
| --- | --- | --- | --- |
| **Student A** | Team lead & backend core | Database, identity, permissions, audit, admin backend, notifications | 4.1, 4.12, 4.16, 4.10 |
| **Student B** | Backend — commerce | Events, ticket types, checkout, inventory, tickets/PDF, refunds | 4.2–4.7, 4.9, 4.14 |
| **Student C** | Frontend — attendee web | Public catalogue, event page, checkout UI, order & ticket pages, i18n | 4.3, 4.6, 4.7, NFR §7 |
| **Student D** | Frontend — organizer & admin, analytics, support | Dashboard, analytics, support desk, admin portal | 4.13, 4.15, 4.16, 4.12 |
| **Student E** | Mobile & QA/DevOps | Expo scanner, check-in, Docker, CI, test specs, release | 4.8, §8 deployment |

## Student A — Team lead & backend core

Owns the parts everything else is built on, and runs the reports.

- `db/init/`, `db/seed/`, `db/tests/` — schema, constraints, triggers, demo data
- `api/internal/auth/`, `api/internal/config/`, `api/internal/httpx/`
- `api/internal/store/`: `users.go`, `tokens.go`, `profiles.go`, `audit.go`,
  `admin.go`, `moderation.go`, `notifications.go`, `store.go`
- `api/internal/api/`: `handlers_auth.go`, `handlers_account.go`,
  `handlers_profile.go`, `handlers_admin*.go`, `handlers_moderation.go`,
  `middleware.go`, `server.go` (route table)
- `api/internal/email/` — templates and the console sender

Also: writes and submits every biweekly report, chairs Monday check-ins.

## Student B — Backend, commerce

The money path. The hardest correctness work in the project lives here.

- `api/internal/store/`: `events.go`, `ticket_types.go`, `checkout.go`,
  `holds.go`, `seating.go`, `activation.go`, `campaigns.go`, `refunds.go`,
  `cancellations.go`, `tickets.go`, `duplicate.go`, `slug.go`
- `api/internal/api/`: `handlers_events.go`, `handlers_ticket_types.go`,
  `handlers_checkout.go`, `handlers_holds.go`, `handlers_activation.go`,
  `handlers_campaigns.go`, `handlers_promo.go`, `handlers_refunds.go`,
  `handlers_cancellations.go`, `handlers_tickets.go`, `handlers_public.go`
- `api/internal/ticketpdf/` — A4 PDF, QR, embedded Unicode font

Non-negotiables for this lane: money never touches a float, and inventory is
decided inside the transaction that issues the ticket.

## Student C — Frontend, attendee web

Everything a ticket buyer sees.

- `web/src/app/events/` — catalogue and event page
- `web/src/app/orders/` — order confirmation and ticket cards
- `web/src/app/(auth)/` — login, register, password reset, verify email
- `web/src/components/`: `checkout-dialog.tsx`, `ticket-selector.tsx`,
  `seat-map.tsx`, `seat-checkout.tsx`, `ticket-card.tsx`, `promo-box.tsx`,
  `language-switcher.tsx`
- `web/src/lib/i18n/` — Kazakh, Russian and English dictionaries

Non-negotiable: the customer-facing interface ships in Kazakh and Russian. SRS
§7 makes that a requirement, not a nice-to-have.

## Student D — Frontend organizer/admin, analytics, support

Owns three features end to end — backend and UI — so nothing falls between
the API and the screen.

- `web/src/app/(app)/dashboard/`, `web/src/app/(app)/admin/`
- `web/src/components/`: `event-analytics.tsx`, `event-timeline.tsx`,
  `order-manager.tsx`, `attendee-list.tsx`, `campaign-manager.tsx`,
  `activation-checklist.tsx`, `admin-portal.tsx`, support thread views
- `api/internal/store/analytics.go`, `api/internal/store/support.go`
- `api/internal/api/handlers_analytics.go`, `handlers_support.go`,
  `handlers_attendees.go`

## Student E — Mobile & QA/DevOps

The scanner app, and the quality bar for everyone else.

- `mobile/` — the whole Expo app: login, event picker, camera scan, manual
  search, offline screen
- `api/internal/api/handlers_checkin.go`, `handlers_offline.go`,
  `api/internal/store/checkin.go`, `staff.go`, `offline.go`, `attendees.go`
- `docker-compose.yml`, `Makefile`, `.github/workflows/`
- `1.md` … `10.md` — the phase test specifications
- `docs/` — Postman collection, verification notes; `README.md`

Also: owns the definition of done. A feature is not done until the phase spec
for it passes.

## Working agreements

- **Review across lanes.** A reviews C, B reviews D, C reviews E, D reviews A,
  E reviews B. Nobody reviews their own area, so knowledge spreads.
- **Backend before frontend, by days not weeks.** B ships an endpoint Monday,
  C consumes it Wednesday. Agree the JSON shape in the Jira issue first.
- **The spec is the handover.** E writes the phase spec while the feature is
  being built; the owner is done when it passes.
- **Blocked for more than one day is a report item.** Say so on Monday, not on
  submission Sunday.
