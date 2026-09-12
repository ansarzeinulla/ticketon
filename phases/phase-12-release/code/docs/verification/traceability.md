# Requirement traceability

Every Required-MVP requirement in `bilet.md` §8, plus all twelve §11 success
criteria, mapped to the code that implements it and the test that proves it.

A row is green only when a named, runnable test backs it. "Verified by hand"
appears exactly where a test cannot reach — a camera-less simulator, a printed
page — and says what was done instead.

Scope is §8 **Required MVP Features** and §11. The §8 "Bonus and Stretch"
items are listed at the end as deliberate exclusions.

Run everything with:

```bash
make up && make seed && make verify        # offline suites
make api-run && make api-smoke            # acceptance checks
make web-dev && make web-e2e               # browser suite
```

---

## §11 Success criteria

| # | Criterion | Proven by |
| --- | --- | --- |
| 1 | An organizer can publish a free event and distribute valid tickets | `e2e/free-event.spec.ts` "creates, publishes and unpublishes"; `TestPhase2SuccessCriteria`; smoke 04–13 |
| 2 | An organizer can complete simulated paid-sales activation and receive demonstration orders | `TestPhase10SuccessCriteria/criterion 2`; `activation_test.go` (7 tests); smoke 09–09g; `e2e/paid-checkout.spec.ts` "cannot be bought until the activation checklist is done" |
| 3 | An attendee can complete checkout and receive a QR-code ticket | `TestPhase4SuccessCriteria`; `TestPhase5SuccessCriteria`; `e2e/paid-checkout.spec.ts` "buys a paid ticket and receives a QR-code ticket" |
| 4 | An attendee can download and print a ticket that remains scannable at check-in | `ticketpdf` suite decodes the QR **off the rendered page**, in grayscale and downscaled to 700 px; `e2e/paid-checkout.spec.ts` "the downloaded ticket is a real A4 PDF"; smoke 20–21 |
| 5 | An Event Admin can use the mobile app to validate tickets and prevent duplicate entry | `TestPhase6SuccessCriteria`; `TestCheckInIsAtomicUnderConcurrency` (8 simultaneous scans admit exactly one); **verified by hand** in the iOS Simulator — `evidence/scanner-already-used.png` |
| 6 | An attendee can open a contextual support case and exchange asynchronous messages | `TestPhase8SuccessCriteria`; `e2e/support-and-moderation.spec.ts` "opens a support case … and the organizer replies" |
| 7 | An attendee can scan a Campaign QR Code, receive a discount, and complete an attributed order | `TestPhase7SuccessCriteria`; `TestPromoRedemptionLimitIsAtomic`; `e2e/paid-checkout.spec.ts` "a promo code is priced by the server" |
| 8 | An admission scanner rejects Campaign QR Codes while accepting valid ticket QRs | `TestGateRejectsEveryCampaignForm` (bare token, campaign link, link with tracking params); DB `05_business_rules.sql` token-namespace assertions; **verified by hand** — `evidence/scanner-campaign-qr-refused.png` |
| 9 | Organizers can view accurate ticket, attendance, payment and refund records | `TestPhase9SuccessCriteria`; `analytics_test.go` (11 tests); smoke 40–44 cross-check every figure against SQL |
| 10 | Organizers can view basic sales, capacity, campaign and attendance analytics without extra checkout fields | `e2e/analytics-and-gate.spec.ts` "analytics report figures computed from the rows"; "a refund is reflected in the analytics" |
| 11 | Organizers can review past and cancelled events, inspect a timeline, and duplicate without copying transactions | `e2e/analytics-and-gate.spec.ts` "timeline records what happened" and "duplicating … copies the configuration and none of the history"; smoke 45–48 |
| 12 | **Administrators can suspend a suspicious event and stop further sales** | `handlers_admin.go` + `e2e/support-and-moderation.spec.ts` "suspends an event and it stops selling"; smoke 33–35. **Closed in this work** — the API existed since Phase 8, the portal had no buttons |

---

## §8 Required MVP Features

| Feature | SRS | Code | Proven by |
| --- | --- | --- | --- |
| User registration and authentication | 4.1 | `handlers_auth.go`, `handlers_account.go` | `auth_test.go` (18), `account_test.go` (9), `e2e/auth.spec.ts` (7) |
| Organizer profiles | 4.1 | `store/profiles.go`, `handlers_profile.go`, `dashboard/profile` | `profile_test.go` (12) — **closed in this work**, the table was write-only |
| Event creation and publication | 4.2 | `handlers_events.go`, `events/new`, `events/[id]/edit` | `events_test.go` (25), `e2e/free-event.spec.ts`. **Edit and unpublish UI closed in this work** |
| Free and paid ticket types | 4.3 | `handlers_ticket_types.go`, `ticket-type-manager.tsx` | `ticket_types_test.go` (8); all five counters — **checked-in added in this work** |
| Simulated paid-sales activation | 4.5 | `store/activation.go`, `activation-checklist.tsx` | `activation_test.go` (7); the payout account it registers — **added in this work** |
| Simulated KZT checkout | 4.6 | `store/checkout.go`, `checkout-dialog.tsx` | `checkout_test.go` (14) incl. `TestCheckoutDoesNotOversellUnderConcurrency` (30 buyers, 10 tickets, exactly 10 succeed) |
| QR-code ticket generation | 4.7 | `internal/ticketpdf/qr.go` | `TestQRRoundTripsTheExactToken`, `TestQRScansAfterDownscaling` |
| Downloadable, printable PDF tickets | 4.7 | `internal/ticketpdf/render.go`, `ticket-card.tsx` | `ticketpdf_test.go` (15), incl. `TestRenderOmitsPaymentDetails` |
| Email confirmations | 4.10 | `internal/email`, `internal/api/notify.go` | `email_test.go` (10), `notifications_test.go` (14) — **5 of 9 triggers were missing, closed in this work** |
| Attendee and order management | 4.4, 4.9 | `order-manager.tsx`, `attendee-list.tsx` | `refunds_test.go` (8), `cancellations_test.go` (9), `e2e/analytics-and-gate.spec.ts`. **Attendee list closed in this work** |
| iOS/Android ticket-verification app | 4.8 | `mobile/` | `checkin_test.go` (11); simulator walkthrough, `evidence/scanner-*.png`. **Undo check-in closed in this work** |
| Basic cancellations and refunds | 4.9 | `store/refunds.go`, `store/cancellations.go` | `refunds_test.go`, `cancellations_test.go`. **Cancelling a free registration returned HTTP 500 — fixed in this work** |
| Administrative moderation | 4.12 | `handlers_admin.go`, `handlers_admin_users.go`, `handlers_moderation.go`, `/admin` | `portal_test.go` (17), `admin_users_test.go` (7), `moderation_test.go` (11). **User suspension, the report queue and configurable settings all closed in this work** |
| Basic organizer analytics | 4.15 | `store/analytics.go`, `event-analytics.tsx` | `analytics_test.go` (11), `e2e/analytics-and-gate.spec.ts` |
| Organizer event history and duplication | 4.16 | `store/duplicate.go`, `event-timeline.tsx` | `analytics_test.go` duplication tests; append-only audit enforced by trigger, `05_business_rules.sql` |
| Asynchronous support chat | 4.13 | `handlers_support.go`, `support-thread.tsx` | `support_test.go` (7), `e2e/support-and-moderation.spec.ts` (2). **Explicit case assignment closed in this work** |
| Promo codes and Campaign QR codes | 4.14 | `handlers_campaigns.go`, `promo-box.tsx` | `campaigns_test.go` (12) |
| Docker-based demonstration deployment | §8 | `api/Dockerfile`, `web/Dockerfile`, `docker-compose.yml` | `docker compose --profile app up` brings all three up healthy — **closed in this work**, compose previously started PostgreSQL only |

---

## Non-functional requirements (§7) that are testable

| Requirement | Proven by |
| --- | --- |
| Passwords stored using secure hashing | `TestHashIsSaltedPerCall`, `TestPasswordChangeStoresOnlyAHash`, smoke 51c |
| Payment-card data never stored | `TestPayoutDestinationIsVisibleAndMasked`; only an opaque reference and a masked value exist |
| Organizer and administrator actions auditable | append-only trigger on `audit_logs`; `TestUserSuspensionIsAudited`, `TestCancellationIsAudited`, `TestReviewIsAudited` |
| Seat availability atomic; no double-selling | `tickets_one_live_ticket_per_seat_uidx`; `04_constraints.sql`, `05_business_rules.sql` |
| No double admission | `check_in_one_active_per_ticket_uidx`; `TestCheckInIsAtomicUnderConcurrency` |
| Promo redemption limits enforced atomically | `TestPromoRedemptionLimitIsAtomic` (12 concurrent checkouts, one redemption) |
| Campaign QR links carry no trusted price | `TestGateRejectsEveryCampaignForm`; the link carries an opaque token only |
| Printable tickets legible on A4 in grayscale | `ticketpdf` decodes the QR off the rendered page in grayscale |
| Calendar exports preserve the event's timezone | *n/a — calendar export is out of scope (bonus)*; event timezone handling itself is covered by `datetime.test.ts` round-trip tests |
| WCAG 2.1 AA on event pages | `field.test.tsx` — the `aria-errormessage` association was **broken and is fixed in this work**; colour is never the only signal in the scanner |
| Money in KZT, never a float | `money.test.ts` (13); `numeric(14,2)` end to end; `04_constraints.sql` arithmetic checks |

---

## Deliberately out of scope

These are §8 "Bonus and Stretch Features", excluded by decision. They are
listed so their absence is a recorded choice rather than an oversight.

| Item | SRS | Why excluded |
| --- | --- | --- |
| Assigned seating and the interactive seat map | 4.3.1, UC8 | §8 Bonus. The schema carries `venues`, `seats` and `seat_holds` with the concurrency guarantees already tested at the database level, so the foundation is there; no API or UI was built. |
| Calendar export (.ics and calendar links) | 4.11, UC6 | §8 Bonus. |
| GA4 traffic/funnel analytics | 4.15 | §8 explicitly "not required for the base MVP". |
| Offline scanner synchronisation | 4.8 | §8 Bonus; 4.8 says "The academic MVP requires online verification only." |
| Support-message attachments | 4.13 | §8 Bonus. |
| Kazakh/Russian localisation | §7 | Not in the §8 Required list; §13.4 names localisation polish as the first thing to cut. The database `locale` column and Cyrillic slug transliteration are in place for it. |
| Real money movement, production KYC, app-store publication | §8 Excluded | Explicitly excluded by the SRS. |
