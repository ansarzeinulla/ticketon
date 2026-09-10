# Biweekly Team Progress Report

CSCI 361 - Fall 2026

**Cover page**

## Report details

- Team name: <TEAM-NAME>
- Reporting period: October 26, 2026 - November 8, 2026
- Submitted by: Student A
- Current stage: Build / Test
- Overall status: On track

### Project links

- Source code repository: <REPO-URL>
- Task management tool: <JIRA-URL>

## Team members

| Full name | Student ID | Email |
| --- | --- | --- |
| Student A | <STUDENT-ID-A> | <EMAIL-A> |
| Student B | <STUDENT-ID-B> | <EMAIL-B> |
| Student C | <STUDENT-ID-C> | <EMAIL-C> |
| Student D | <STUDENT-ID-D> | <EMAIL-D> |
| Student E | <STUDENT-ID-E> | <EMAIL-E> |

{{< pagebreak >}}

## Progress snapshot

Money can now go back as well as in, and attendees can ask for help. A refund
runs as one transaction that voids the tickets, restocks the tier and stops the
holder at the gate, while keeping the attendance history intact. The support
desk carries the order and event it was opened from, and staff notes stay
invisible to attendees. The platform admin portal can search across users,
events, orders and payments, and suspend a suspicious event without stranding
people who already paid. On track, with one defect found and fixed.

## Progress this period

### Previous commitments

- Previous commitment: Full refunds that restock and void tickets
  - Status: Completed
  - Result or reason: One transaction sets the order to refunded, writes the
    immutable refund row, voids every ticket, returns the stock and records the
    audit entry. A refunded ticket is refused at the gate with 409.
  - Evidence: <JIRA-URL>/browse/BF-65, <JIRA-URL>/browse/BF-67, <REPO-URL>/pull/46

- Previous commitment: Attendees can find their own orders
  - Status: Completed
  - Result or reason: A signed-in attendee sees every order from their account.
    A guest checkout is claimed automatically when somebody later registers with
    the same address, so tickets bought before signing up are not stranded.
  - Evidence: <JIRA-URL>/browse/BF-68, <REPO-URL>/pull/48

- Previous commitment: Support desk with contextual cases
  - Status: Completed
  - Result or reason: A case opened from an order carries the order and event
    automatically. The first public staff reply moves it to in progress and
    assigns it. Internal notes are filtered out in SQL, so they are absent from
    the attendee's API response, not merely hidden in the interface.
  - Evidence: <JIRA-URL>/browse/BF-69, <JIRA-URL>/browse/BF-76, <REPO-URL>/pull/50

- Previous commitment: Admin search and suspension
  - Status: Completed
  - Result or reason: One query across users, events, orders and payments.
    Suspending an event returns 403 at checkout immediately, and unsuspending
    returns it to unpublished so the organizer must consciously republish.
  - Evidence: <JIRA-URL>/browse/BF-63, <JIRA-URL>/browse/BF-64, <REPO-URL>/pull/44

- Previous commitment: Manual attendee search and check-in reversal
  - Status: Completed
  - Result or reason: Staff search by name and admit without a QR; the search
    response deliberately omits the admission token, so door staff cannot
    extract working ticket secrets. A reversal restores the ticket and keeps the
    original record for the audit trail.
  - Evidence: <JIRA-URL>/browse/BF-71, <JIRA-URL>/browse/BF-72, <REPO-URL>/pull/49

- Previous commitment: Phase 6 specification and Phase 4-6 results
  - Status: Completed
  - Result or reason: `6.md` written; Phases 4, 5 and 6 executed and recorded.
  - Evidence: <REPO-URL>/blob/main/6.md

### Other progress

- Outcome: Defect found and fixed - refunding a checked-in attendee
  - Status: Done
  - Evidence or result: Refunding an order whose holder had already been admitted
    violated a database consistency constraint, because the ticket kept its
    check-in timestamp while moving to refunded. The fix clears the timestamp in
    the same statement while deliberately keeping the historical check-in record -
    the person did attend, and erasing that would be dishonest. Found by the
    Phase 6 specification before it reached a demo. (BF-B1)

- Outcome: Free registration cancellation
  - Status: Done
  - Evidence or result: An organizer cancels a free registration; the tickets are
    voided and the stock returns, with no refund row because no money moved. (BF-75)

- Outcome: Event report queue and platform settings
  - Status: Done
  - Evidence or result: Signed-in users report an event; administrators review
    and resolve. The activation fee is configurable by administrators only. (BF-73, BF-74)

## Commitments for the next two weeks

- Commitment: Organizer analytics computed from operational rows
  - Owner(s): Student D
  - Due date: November 15, 2026
  - Success criteria: Capacity, sold, remaining, percentage, gross, discounts,
    refunds, net, checked-in and absent - every figure matching a direct SQL
    query to the tiyn, with no pre-computed counters anywhere.

- Commitment: Sales over time, by tier and by campaign
  - Owner(s): Student D
  - Due date: November 22, 2026
  - Success criteria: A sales chart from order timestamps, a per-tier breakdown
    and campaign attribution; filters for date range and ticket type.

- Commitment: Promo codes and campaign QR codes
  - Owner(s): Student B (API), Student C (checkout)
  - Due date: November 22, 2026
  - Success criteria: A campaign issues an opaque `CMP_` token and a downloadable
    QR; discounts are validated and capped server-side and never exceed the line
    price; redemption limits hold under concurrency.

- Commitment: Kazakh and Russian across the customer-facing interface
  - Owner(s): Student C
  - Due date: November 22, 2026
  - Success criteria: A switcher offering Қазақша, Русский and English; the
    catalogue, event page, checkout dialog and order pages all translate.

- Commitment: The platform admin portal and CSV export
  - Owner(s): Student A (export), Student D (portal UI)
  - Due date: November 22, 2026
  - Success criteria: Administrators only; the CSV escapes commas and quotes per
    RFC 4180 and is served `Cache-Control: no-store`.

- Commitment: Phase 7, 8 and 9 specifications and results
  - Owner(s): Student E
  - Due date: November 22, 2026
  - Success criteria: `7.md`, `8.md` and `9.md` written and executed.

## Team contributions and coordination

- Team member: Student A
  - Contribution this period: The admin portal backend - unified search, user and
    event suspension, the report queue and configurable platform settings.
  - Evidence: <JIRA-URL>/browse/BF-63, <JIRA-URL>/browse/BF-73, <REPO-URL>/pull/44
  - Next responsibility: Timeline filtering and the operational CSV export.

- Team member: Student B
  - Contribution this period: The atomic refund path with restocking and gate
    invalidation, free registration cancellation, and the check-in consistency fix.
  - Evidence: <JIRA-URL>/browse/BF-65, <JIRA-URL>/browse/BF-75, <REPO-URL>/pull/46
  - Next responsibility: Campaigns, promo codes and campaign QR tokens.

- Team member: Student C
  - Contribution this period: The attendee's own orders page and the refunded and
    cancelled states on the order page.
  - Evidence: <JIRA-URL>/browse/BF-68, <REPO-URL>/pull/48
  - Next responsibility: The promo box, and Kazakh and Russian localisation.

- Team member: Student D
  - Contribution this period: The support desk end to end - case creation with
    context, the organizer inbox, staff-only notes, case status and polling.
  - Evidence: <JIRA-URL>/browse/BF-69, <JIRA-URL>/browse/BF-76, <REPO-URL>/pull/50
  - Next responsibility: Analytics, backend and dashboard.

- Team member: Student E
  - Contribution this period: Manual attendee search and check-in reversal in the
    scanner, the Phase 6 specification, and execution of Phases 4 to 6 - which is
    how the refund defect was caught.
  - Evidence: <JIRA-URL>/browse/BF-71, <JIRA-URL>/browse/BF-78
  - Next responsibility: The Phase 7, 8 and 9 specifications.

## Risks, blockers, and decisions needed

- Risk: Analytics, campaigns, i18n and the admin portal all land in Sprint 6,
  which is the last full sprint before the freeze.
  - Impact: Anything that slips has only the six-day final week to recover in,
    and that week is reserved for hardening.
  - Next action: Student D starts the analytics queries in week 11 rather than
    week 12, and Student E writes `7.md` before the dashboard exists so the
    numbers have a target to hit.
  - Owner: Student A

- Decision: Unauthorised access to a support case returns 404 rather than 403.
  - Impact: Slightly unusual, and worth stating so it is not read as a bug.
  - Next action: Accepted. A 403 would confirm that a case with that identifier
    exists, which leaks the existence of other people's support requests.
  - Owner: Student D

## Changes, reflection, and support

- Scope or schedule changes: None to required scope. Bonus features remain
  deferred and unstarted.
- Team reflection: The phase specifications are earning their keep - the refund
  defect was a database constraint violation that would have surfaced in the
  demo. Writing the spec before the feature, then running it, is now our default.
  Change: refunds needed a schema question answered mid-implementation; database
  questions go to Student A at the Monday check-in rather than mid-sprint.
- Instructor/TA help requested: None.
