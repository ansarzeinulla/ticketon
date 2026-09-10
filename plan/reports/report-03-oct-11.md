# Biweekly Team Progress Report

CSCI 361 - Fall 2026

**Cover page**

## Report details

- Team name: <TEAM-NAME>
- Reporting period: September 28, 2026 - October 11, 2026
- Submitted by: Student A
- Current stage: Build
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

BiletFlow takes money this period. Checkout runs as a single database
transaction that locks the tier, decrements inventory and issues the tickets
together, and we can now demonstrate that it does not oversell under load: 30
concurrent buyers against 10 tickets produce exactly 10 orders. Free
registration works without payment details, paid tiers are gated behind the
activation checklist, and the audit log physically refuses to be edited. This
was the sprint we were most worried about and it landed on time.

## Progress this period

### Previous commitments

- Previous commitment: Checkout that cannot oversell
  - Status: Completed
  - Result or reason: The tier rows are locked with `SELECT … FOR UPDATE` inside
    the same transaction that writes the order, the items and the tickets, so a
    losing buyer's whole attempt rolls back. `TestCheckoutDoesNotOversellUnderConcurrency`
    fires 30 goroutines at 10 tickets: 10 succeed, 20 get 409
    `insufficient_inventory`, and the database ends with `quantity_sold = 10`.
  - Evidence: <JIRA-URL>/browse/BF-33, <JIRA-URL>/browse/BF-39, <REPO-URL>/pull/24

- Previous commitment: Free registration end to end
  - Status: Completed
  - Result or reason: A guest reserves a free ticket with only a name and email,
    the order is created at a zero total, a QR ticket is issued and the confirmation
    email prints to the API console. No payment details are requested.
  - Evidence: <JIRA-URL>/browse/BF-35, <REPO-URL>/pull/26

- Previous commitment: Paid-sales activation gate
  - Status: Completed
  - Result or reason: Paid tiers return 403 until identity, payout, terms and the
    activation fee are all confirmed. Free tiers on the same event keep selling -
    activation gates money, not registration.
  - Evidence: <JIRA-URL>/browse/BF-41, <REPO-URL>/pull/29

- Previous commitment: Append-only audit log
  - Status: Completed
  - Result or reason: A `BEFORE UPDATE OR DELETE` trigger raises an exception, so
    history cannot be rewritten even with direct database access.
  - Evidence: <JIRA-URL>/browse/BF-31, <REPO-URL>/pull/22

- Previous commitment: Phase 3 test specification and results
  - Status: Completed
  - Result or reason: `3.md` written and executed; all cases pass.
  - Evidence: <REPO-URL>/blob/main/3.md

### Other progress

- Outcome: Simulated payments with a decline path
  - Status: Done
  - Evidence or result: Payments are clearly labelled simulations and never
    presented as real. A buyer email at the decline domain triggers a refusal,
    which returns 402 and rolls the transaction back, so a failed payment leaves
    no order, no ticket and no reserved stock. (BF-42)

- Outcome: Role-based permissions
  - Status: Done
  - Evidence or result: Attendee, organizer, event admin, support staff and
    platform admin, checked per request against the database rather than trusted
    from the token, so suspending an account takes effect immediately. (BF-32)

- Outcome: Notifications
  - Status: Done
  - Evidence or result: `notifications` rows plus formatted console emails for
    order confirmation and account actions. (BF-40)

## Commitments for the next two weeks

- Commitment: Printable A4 PDF tickets with scannable QR codes
  - Owner(s): Student B
  - Due date: October 18, 2026
  - Success criteria: One A4 page carrying event, venue, attendee, tier, seat,
    ticket identifier and QR; the QR decodes back to the exact stored token; no
    payment details appear.

- Commitment: Cyrillic renders natively on the ticket
  - Owner(s): Student B
  - Due date: October 18, 2026
  - Success criteria: A ticket for "Нұрлан Сағындық" prints the actual name, not
    a transliteration and not question marks.

- Commitment: The scanner app admits an attendee
  - Owner(s): Student E
  - Due date: October 25, 2026
  - Success criteria: Sign in on a device, pick an assigned event, scan a ticket,
    full-screen green with the attendee's name; a second scan is refused.

- Commitment: Gate staff delegation
  - Owner(s): Student A
  - Due date: October 25, 2026
  - Success criteria: An organizer assigns staff by email; that account sees only
    assigned events; revoking access cuts the gate off immediately.

- Commitment: Cart holds and the processing fee
  - Owner(s): Student B
  - Due date: October 25, 2026
  - Success criteria: Stock is reserved for 15 minutes during checkout and
    released automatically; the fee is shown before payment and stored on the order.

- Commitment: Phase 4 and 5 specifications
  - Owner(s): Student E
  - Due date: October 25, 2026
  - Success criteria: `4.md` and `5.md` written and used to sign the sprint off.

## Team contributions and coordination

- Team member: Student A
  - Contribution this period: Role permissions enforced per request, the
    append-only audit trigger, and the notification store and console mailer.
  - Evidence: <JIRA-URL>/browse/BF-31, <JIRA-URL>/browse/BF-32, <REPO-URL>/pull/22
  - Next responsibility: Gate staff assignment and revocation.

- Team member: Student B
  - Contribution this period: The checkout transaction with row locking, free
    registration, the activation gate and simulated payments with a decline path.
  - Evidence: <JIRA-URL>/browse/BF-33, <JIRA-URL>/browse/BF-42, <REPO-URL>/pull/24
  - Next responsibility: QR tokens and the printable PDF ticket.

- Team member: Student C
  - Contribution this period: The ticket selector with live totals in tiyn, the
    checkout dialog, and the order confirmation page.
  - Evidence: <JIRA-URL>/browse/BF-36, <JIRA-URL>/browse/BF-43, <REPO-URL>/pull/27
  - Next responsibility: Ticket cards with QR images and PDF download links.

- Team member: Student D
  - Contribution this period: The organizer's order and attendee lists, and the
    activation checklist UI.
  - Evidence: <JIRA-URL>/browse/BF-37, <JIRA-URL>/browse/BF-44, <REPO-URL>/pull/30
  - Next responsibility: The event activity timeline.

- Team member: Student E
  - Contribution this period: The Phase 3 specification, the concurrency test
    that proves checkout cannot oversell, and business-rule tests in SQL.
  - Evidence: <JIRA-URL>/browse/BF-39, <JIRA-URL>/browse/BF-45
  - Next responsibility: The Expo scanner app.

## Risks, blockers, and decisions needed

- Risk: The scanner app is the only mobile deliverable and only one person owns it.
  - Impact: If Student E is unavailable, SRS 4.8 is at risk with no second owner.
  - Next action: Student E pairs with Student C on the Expo setup in week 7 so a
    second person can run and debug the app. The check-in API stays on the Go
    side, where three of us can work on it.
  - Owner: Student E

- Risk: PDF generation with Cyrillic. The core PDF fonts have no Cyrillic at all.
  - Impact: Kazakh and Russian names would print as question marks - unacceptable
    for a Kazakhstan platform.
  - Next action: Embed a Unicode TrueType face in the binary rather than
    transliterating names. Investigated this period, scheduled as BF-50.
  - Owner: Student B

## Changes, reflection, and support

- Scope or schedule changes: None.
- Team reflection: Writing the concurrency test before the implementation was
  the single best decision of the sprint - it turned "we think it locks
  correctly" into a number we can show. We will use the same order for refunds.
  Change: our commits were bunched on Thursdays in week 5; week 6 was spread
  across four days and reviews were faster as a result.
- Instructor/TA help requested: None.
