# Biweekly Team Progress Report

CSCI 361 - Fall 2026

**Cover page**

## Report details

- Team name: <TEAM-NAME>
- Reporting period: October 12, 2026 - October 25, 2026
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

The ticket became a real object this period and the door works. Attendees get an
A4 PDF with a QR code that survives printing in grayscale, Kazakh names render
natively rather than being transliterated, and the Expo scanner admits an
attendee with a full-screen green result in well under two seconds - refusing a
second scan of the same ticket. With checkout, delivery and check-in all
working, the core attendee journey is complete end to end. On track.

## Progress this period

### Previous commitments

- Previous commitment: Printable A4 PDF tickets with scannable QR codes
  - Status: Completed
  - Result or reason: One A4 page (595.28 x 841.89 pt) carrying event title,
    local date and time, venue, attendee, tier, seat where applicable, the ticket
    identifier and the QR. The raw token is printed underneath as a manual
    fallback for door staff. No payment details appear anywhere on it.
  - Evidence: <JIRA-URL>/browse/BF-49, <REPO-URL>/pull/34

- Previous commitment: Cyrillic renders natively on the ticket
  - Status: Completed
  - Result or reason: We embed DejaVu Sans Condensed in the binary, covering
    Cyrillic including the Kazakh letters ә қ ң ө ұ ү һ і and the ₸ sign. An
    attendee named "Нұрлан Сағындық" sees their own name on the ticket. The
    earlier transliteration approach was removed.
  - Evidence: <JIRA-URL>/browse/BF-50, <REPO-URL>/pull/35

- Previous commitment: The scanner app admits an attendee
  - Status: Completed
  - Result or reason: Sign in, pick an assigned event, scan with the camera:
    full-screen green with the attendee's name, tier, seat and a live gate count.
    A second scan of the same ticket gives full-screen red with the time of the
    first entry, enforced by a partial unique index rather than by application
    logic alone.
  - Evidence: <JIRA-URL>/browse/BF-59, <JIRA-URL>/browse/BF-60, <REPO-URL>/pull/40

- Previous commitment: Gate staff delegation
  - Status: Completed
  - Result or reason: Staff are assigned by email and see only their assigned
    events. Revoking an assignment cuts off the gate on the next request - 403.
  - Evidence: <JIRA-URL>/browse/BF-54, <REPO-URL>/pull/37

- Previous commitment: Cart holds and the processing fee
  - Status: Completed
  - Result or reason: Stock is reserved for 15 minutes while a buyer is in
    checkout and released automatically when the hold expires. The fee is shown
    before payment and stored on the order, with a database constraint asserting
    `total = subtotal - discount + fee`.
  - Evidence: <JIRA-URL>/browse/BF-55, <JIRA-URL>/browse/BF-56, <REPO-URL>/pull/38

- Previous commitment: Phase 4 and 5 specifications
  - Status: Completed
  - Result or reason: `4.md` and `5.md` written and executed.
  - Evidence: <REPO-URL>/blob/main/4.md, <REPO-URL>/blob/main/5.md

### Other progress

- Outcome: Campaign codes are refused at the gate
  - Status: Done
  - Evidence or result: A promotional `CMP_` token - bare or embedded in an event
    URL - is rejected with 400 `campaign_token` before any ticket lookup, and no
    check-in row is written. SRS 4.14 requires this explicitly. (BF-61)

- Outcome: QR round-trip proof
  - Status: Done
  - Evidence or result: An automated test generates the QR, decodes it with an
    independent library and asserts the string equals the stored `qr_token`
    byte for byte. The admission token is separate from the ticket's database id.

- Outcome: Event activity timeline
  - Status: Done
  - Evidence or result: A chronological feed of publication, ticket changes,
    orders and check-ins, newest first. (BF-58)

## Commitments for the next two weeks

- Commitment: Full refunds that restock and void tickets
  - Owner(s): Student B
  - Due date: November 1, 2026
  - Success criteria: One transaction sets the order to refunded, writes the
    refund ledger row, voids the tickets, clears any check-in timestamp and
    returns the stock; a refunded ticket is refused at the gate.

- Commitment: Attendees can find their own orders
  - Owner(s): Student C
  - Due date: November 1, 2026
  - Success criteria: A signed-in attendee sees every order placed with their
    account, including guest orders later claimed by the same email address.

- Commitment: Support desk with contextual cases
  - Owner(s): Student D
  - Due date: November 8, 2026
  - Success criteria: An attendee opens a case from an order; the organizer
    answers from an inbox; the first staff reply moves it to in progress and
    assigns it; internal notes are invisible to the attendee.

- Commitment: Admin search and suspension
  - Owner(s): Student A
  - Due date: November 8, 2026
  - Success criteria: One search box across users, events, orders and payments;
    suspending an event stops sales immediately while already-sold tickets still scan.

- Commitment: Manual attendee search and check-in reversal
  - Owner(s): Student E
  - Due date: November 8, 2026
  - Success criteria: Staff find an attendee by name and admit them without a QR;
    an accidental check-in can be undone and the ticket used again.

- Commitment: Phase 6 specification and Phase 4-6 results
  - Owner(s): Student E
  - Due date: November 8, 2026
  - Success criteria: `6.md` written; Phases 4, 5 and 6 executed and recorded.

## Team contributions and coordination

- Team member: Student A
  - Contribution this period: Gate staff assignment and revocation, and the
    order confirmation email carrying ticket links.
  - Evidence: <JIRA-URL>/browse/BF-54, <JIRA-URL>/browse/BF-47, <REPO-URL>/pull/37
  - Next responsibility: The admin portal - unified search, suspension and the
    moderation queue.

- Team member: Student B
  - Contribution this period: QR token generation, the A4 PDF renderer with an
    embedded Unicode font, 15-minute cart holds and the processing fee.
  - Evidence: <JIRA-URL>/browse/BF-49, <JIRA-URL>/browse/BF-55, <REPO-URL>/pull/34
  - Next responsibility: Refunds - atomic, restocking and gate-invalidating.

- Team member: Student C
  - Contribution this period: Ticket cards with QR images served directly from
    the API, and PDF download links; paired with Student E on the Expo setup.
  - Evidence: <JIRA-URL>/browse/BF-51, <REPO-URL>/pull/36
  - Next responsibility: The attendee's own orders page.

- Team member: Student D
  - Contribution this period: The event activity timeline and ticket status
    columns in the dashboard.
  - Evidence: <JIRA-URL>/browse/BF-58, <REPO-URL>/pull/39
  - Next responsibility: The support desk, backend and UI.

- Team member: Student E
  - Contribution this period: The Expo scanner - sign-in, event picker, camera
    scanning, duplicate refusal and campaign-code rejection - plus the Phase 4
    and 5 specifications.
  - Evidence: <JIRA-URL>/browse/BF-59, <JIRA-URL>/browse/BF-62, <REPO-URL>/pull/40
  - Next responsibility: Manual attendee search and check-in reversal.

## Risks, blockers, and decisions needed

- Risk: Half the required backlog remains and six weeks are left, including
  refunds, support, analytics, campaigns and the admin portal.
  - Impact: A slip in Sprint 5 would push analytics into the final week, which is
    reserved for hardening.
  - Next action: The bonus items stay unstarted. Offline scanner sync and the
    `.ics` export are formally deferred to "only if Sprint 6 finishes early".
  - Owner: Student A

- Decision: The mobile app targets the Expo Go runtime, not a store build.
  - Impact: We cannot demonstrate an App Store or Play submission.
  - Next action: Accepted - SRS §8 explicitly excludes store publication from the
    MVP. We demonstrate on a physical device through Expo Go.
  - Owner: Student E

## Changes, reflection, and support

- Scope or schedule changes: None to required scope. Bonus features formally
  deferred, as recorded above.
- Team reflection: Pairing Student C with Student E on the Expo setup removed
  our single point of failure on mobile in about two hours - cheap insurance,
  worth repeating for any lane only one person can run. Change: the PDF work
  needed a spike before an estimate; we will spike unknowns explicitly rather
  than estimating them blind.
- Instructor/TA help requested: None.
