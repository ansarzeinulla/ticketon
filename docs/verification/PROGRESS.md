# Gap closure — status

Scope: bilet.md §8 "Required MVP Features" + §11 success criteria.
Bonus/stretch items (seat map, .ics calendar, GA4, offline sync, i18n) are out
of scope by decision, and listed at the end.

## Phase 3 — backend (done)

| # | SRS | What was missing | Closed by |
| --- | --- | --- | --- |
| G1 | 4.9 | Cancelling a free registration **500'd** — `refunds_amount_chk` rejects a zero refund, and the order list said `refundable: true` | `POST /orders/{id}/cancel`, `cancellable` flag, own email template, audit entry |
| G2 | 4.12 | No way to suspend a *user* | `POST /admin/users/{id}/suspend` + `/unsuspend`; checkout now 403s `organizer_suspended` |
| G3 | 4.12 | "Review reported events" — no report entity at all | `event_reports` table, `POST /events/{id}/report`, admin queue + decision |
| G4 | 4.12 | Activation fee was a Go constant | `platform_settings` table, `GET`/`PATCH /admin/settings` |
| G5 | 4.1 | Organizer profile unreadable and uneditable; no password change | `GET`/`PATCH /auth/profile`, `POST /auth/password` |
| G6 | 4.10 | 5 of 9 notifications missing | payment failure, event update, payout status, case assignment, case status |
| G7 | 4.13 | Case assignment was implicit only | `POST /support/cases/{id}/assign` |

Found while closing them, not in the original plan:
- The activation checklist never wrote a `payout_accounts` row, so SRS 4.5's
  "connects or registers a valid payout account" had no destination behind it.
- The simulated gateway could never decline, so SRS 4.10's payment-failure
  notification and SRS 4.6's "failed transactions issue no tickets" were not
  demonstrable. Added a reserved decline domain.
- `notifications.support_case_id` existed since Phase 1 and was never written.

## Phase 4 — UI (done)

| # | SRS | What was missing | Closed by |
| --- | --- | --- | --- |
| G8 | 4.12, **§11** | Admin portal could display suspension but not cause it | suspend/unsuspend + paid-sales controls, with typed confirmations |
| G9 | 4.2 | No edit form, no unpublish; cancel fired on one click | `/dashboard/events/[id]/edit`, Unpublish button, cancel confirmation |
| G10 | 1.2, 4.4 | `/events` 404'd; no attendee list | public catalogue page, attendee search + manual check-in panel |
| G11 | 4.3 | Only 3 of 5 counters shown; checked-in not even in the API | `quantity_checked_in` added; all five surfaced |
| G12 | 4.8 | Staff endpoints unreachable — scanner unusable without cURL | gate-staff panel on the event page |
| G13 | 4.8 | "Undo an accidental check-in" existed in the API, unused | undo on the scan overlay and in attendee search |
| G15 | — | `mobile/.env.local` pinned localhost, breaking device demos | blanked, with the reason documented |
| G17 | WCAG | `aria-errormessage` pointed at an id that was never rendered | id added; also added to select and textarea, which had none |

## Remaining

- Phase 5: Playwright E2E + component unit tests
- Phase 6: driven walkthrough with screenshot evidence
- Phase 7: seed credentials (G14), Docker compose for api/web (G16), CI (G18),
  traceability matrix
