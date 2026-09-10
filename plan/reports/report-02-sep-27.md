# Biweekly Team Progress Report

CSCI 361 - Fall 2026

**Cover page**

## Report details

- Team name: <TEAM-NAME>
- Reporting period: September 14, 2026 - September 27, 2026
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

BiletFlow now has real users and real events. A person can register, verify
their address, sign in, reset a forgotten password, and - as an organizer -
create an event, add free and paid ticket tiers, and publish it to a public
page. Every case in the Phase 1 specification passes. The one thing that took
longer than planned was Cyrillic slugs, which is worth the time: Kazakh event
titles are the normal case here, not an edge case. On track.

## Progress this period

### Previous commitments

- Previous commitment: Registration, sign-in and sessions working end to end
  - Status: Completed
  - Result or reason: bcrypt (cost 12) hashing, JWT issuing and verification,
    and the register/login endpoints, wired to the browser. Sessions are held in
    an httpOnly cookie rather than `localStorage`, so a script on the page
    cannot read the token.
  - Evidence: <JIRA-URL>/browse/BF-17, <REPO-URL>/pull/11

- Previous commitment: Password reset and email verification
  - Status: Completed
  - Result or reason: Tokens are random, stored only as SHA-256 hashes, single
    use, and superseded when a new one is requested. Requesting a reset for an
    unknown address returns the same 202 as a known one, so the endpoint cannot
    be used to discover who has an account.
  - Evidence: <JIRA-URL>/browse/BF-23, <JIRA-URL>/browse/BF-24, <REPO-URL>/pull/14

- Previous commitment: Event creation, editing and publication
  - Status: Completed
  - Result or reason: Events are created as drafts, get a unique slug, and move
    through publish / unpublish / cancel from the dashboard. Cancelled is
    terminal.
  - Evidence: <JIRA-URL>/browse/BF-27, <REPO-URL>/pull/16

- Previous commitment: Ticket types with free and paid tiers
  - Status: Completed
  - Result or reason: Prices are `numeric(14,2)` in PostgreSQL and decimal
    strings over the wire - no float touches money anywhere. Tiers can be
    hidden without deletion, and sales windows are stored.
  - Evidence: <JIRA-URL>/browse/BF-26, <REPO-URL>/pull/17

- Previous commitment: Phase 2 test specification
  - Status: Completed
  - Result or reason: `2.md` covers the lifecycle state machine, ticket
    management and slug de-duplication.
  - Evidence: <REPO-URL>/blob/main/2.md

### Other progress

- Outcome: Cyrillic transliteration for slugs
  - Status: Done
  - Evidence or result: `api/internal/store/slug.go` maps Kazakh and Russian
    letters to ASCII, so "Алматы Джаз" becomes `almaty-dzhaz` and stays a usable
    URL. Duplicates get a numeric suffix. (BF-19)

- Outcome: Organizer profiles
  - Status: Done
  - Evidence or result: Contact details and a simulated payout account, per SRS
    4.1. (BF-25)

- Outcome: Phase 1 signed off
  - Status: Done
  - Evidence or result: All Phase 1 cases pass, including the account-enumeration
    and token-hashing checks. Recorded in `1.md`.

## Commitments for the next two weeks

- Commitment: Checkout that cannot oversell
  - Owner(s): Student B
  - Due date: October 4, 2026
  - Success criteria: An automated test fires 30 concurrent purchases at 10
    tickets; exactly 10 succeed, 20 receive 409, and `quantity_sold` is 10.

- Commitment: Free registration end to end
  - Owner(s): Student B (API), Student C (web)
  - Due date: October 4, 2026
  - Success criteria: A guest reserves a free ticket without payment details,
    gets an order and a QR ticket, and the organizer sees them in the attendee list.

- Commitment: Paid-sales activation gate
  - Owner(s): Student B (API), Student D (checklist UI)
  - Due date: October 11, 2026
  - Success criteria: Paid tiers cannot be bought before the four-step checklist
    is complete; free tiers on the same event stay purchasable.

- Commitment: Append-only audit log
  - Owner(s): Student A
  - Due date: October 11, 2026
  - Success criteria: A database trigger refuses UPDATE and DELETE on
    `audit_logs`; attempting either raises an exception.

- Commitment: Phase 3 test specification and results
  - Owner(s): Student E
  - Due date: October 11, 2026
  - Success criteria: `3.md` written and executed against a running system.

## Team contributions and coordination

- Team member: Student A
  - Contribution this period: Password hashing and JWTs, register/login,
    password reset, email verification, organizer profiles, and the console
    email sender.
  - Evidence: <JIRA-URL>/browse/BF-15, <JIRA-URL>/browse/BF-23, <REPO-URL>/pull/11
  - Next responsibility: Role permissions and the append-only audit trail.

- Team member: Student B
  - Contribution this period: Events CRUD, slug generation with transliteration,
    the lifecycle state machine, and ticket-type management.
  - Evidence: <JIRA-URL>/browse/BF-18, <JIRA-URL>/browse/BF-26, <REPO-URL>/pull/16
  - Next responsibility: The checkout transaction and inventory locking - the
    highest-risk work in the project.

- Team member: Student C
  - Contribution this period: Wired the auth pages to the API, added the API
    client and auth context, and built the public catalogue and event page.
  - Evidence: <JIRA-URL>/browse/BF-20, <JIRA-URL>/browse/BF-28, <REPO-URL>/pull/15
  - Next responsibility: The ticket selector and checkout dialog.

- Team member: Student D
  - Contribution this period: The event create and edit forms, and the organizer
    event dashboard with lifecycle actions.
  - Evidence: <JIRA-URL>/browse/BF-21, <JIRA-URL>/browse/BF-29, <REPO-URL>/pull/18
  - Next responsibility: The attendee and order lists, and the activation checklist.

- Team member: Student E
  - Contribution this period: Database CRUD and constraint tests, the Phase 2
    specification, and execution of Phase 1.
  - Evidence: <JIRA-URL>/browse/BF-22, <JIRA-URL>/browse/BF-30
  - Next responsibility: The Phase 3 specification and the concurrency test.

## Risks, blockers, and decisions needed

- Risk: Checkout concurrency is the hardest correctness problem we have, and it
  lands next sprint.
  - Impact: Getting it wrong means selling the same seat twice, which would
    undermine every figure the dashboard reports.
  - Next action: Student B writes the locking strategy (`SELECT … FOR UPDATE`
    inside the issuing transaction) as a short design note before coding, and
    Student E writes the concurrency test first, so the test exists before the
    implementation.
  - Owner: Student B

- Decision: Sessions in an httpOnly cookie rather than `localStorage`.
  - Impact: Requires a small server-side proxy in Next.js, which is slightly
    more work than reading a token in JavaScript.
  - Next action: Accepted and implemented. The tradeoff is deliberate - a token
    a script can read is a token an XSS bug can steal.
  - Owner: Student C

## Changes, reflection, and support

- Scope or schedule changes: None.
- Team reflection: Agreeing the JSON error envelope in Sprint 1 is paying off -
  Student C has not had to special-case a single endpoint's error shape.
  Repeat. Change: two pull requests sat unreviewed for three days; reviews are
  now claimed at the Wednesday sync rather than left to whoever notices.
- Instructor/TA help requested: None.
