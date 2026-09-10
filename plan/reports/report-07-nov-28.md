# Biweekly Team Progress Report

CSCI 361 - Fall 2026

**Cover page**

## Report details

- Team name: <TEAM-NAME>
- Reporting period: November 23, 2026 - November 28, 2026 (final report)
- Submitted by: Student A
- Current stage: Test / Complete
- Overall status: On track - delivered

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

BiletFlow is delivered. All eighteen required MVP features from SRS §8 are
built, all ten phase specifications pass against a freshly seeded database, and
`v1.0.0` is tagged. The final week went to hardening rather than features: text
limits now count characters instead of bytes so Kazakh and Russian input is not
quietly truncated, uploads are bounded, and organizers can see the platform fee
and their estimated payout. The team met all twelve success criteria in SRS §11.

## Progress this period

### Previous commitments

- Previous commitment: Validation hardening across every text field
  - Status: Completed
  - Result or reason: Every length limit is counted in characters, not bytes. The
    old byte limits gave Cyrillic half the room of Latin, since a Kazakh letter
    costs two bytes in UTF-8 - a bug that only shows up in the language most of
    our users write in. Passwords are pre-hashed before bcrypt so its 72-byte
    ceiling no longer leaks into the product. Uploads are bounded by size, type
    and pixel dimensions.
  - Evidence: <JIRA-URL>/browse/BF-95, <REPO-URL>/pull/64

- Previous commitment: Fees and estimated payout visible to organizers
  - Status: Completed
  - Result or reason: The dashboard now shows the processing fee charged and the
    estimated payout - the ticket money on unrefunded orders with that fee
    removed - alongside gross and refunds, which SRS 4.6 requires. Gross stays
    historical after a refund while the payout drops to zero.
  - Evidence: <JIRA-URL>/browse/BF-96, <REPO-URL>/pull/65

- Previous commitment: Remaining localisation gaps closed
  - Status: Completed
  - Result or reason: Every customer-facing page renders in Kazakh and Russian.
    SRS §7 makes this a requirement rather than a bonus, so we treated a missed
    English string as a defect.
  - Evidence: <JIRA-URL>/browse/BF-98, <REPO-URL>/pull/66

- Previous commitment: Full regression sweep, Phases 1 to 10
  - Status: Completed
  - Result or reason: All ten specifications executed against a database rebuilt
    from scratch. Three defects were found and fixed: an event that had already
    ended could still be published, a blank required field reported "too short"
    instead of "required", and the demo seed could not be re-run after a test
    purchase because of a foreign key it did not clear.
  - Evidence: <JIRA-URL>/browse/BF-100, <JIRA-URL>/browse/BF-101

- Previous commitment: Documentation, demo rehearsal and release tag
  - Status: Completed
  - Result or reason: `README.md` and the Postman collection are current; the
    demo runs end to end from `make reset && make seed`; `v1.0.0` tagged.
  - Evidence: <REPO-URL>/releases/tag/v1.0.0

### Other progress

- Outcome: SRS §11 success criteria verified one by one
  - Status: Done
  - Evidence or result: All twelve confirmed on a clean database: a free event
    published and tickets distributed; simulated activation and demonstration
    orders; checkout producing a QR ticket; a printed ticket scanning at the
    gate; the scanner preventing duplicate entry; a contextual support case
    answered; a campaign QR producing an attributed discounted order; the scanner
    refusing campaign codes while accepting tickets; accurate ticket, attendance,
    payment and refund records; analytics with no extra checkout fields; past and
    cancelled events reviewable and duplicable without their transactions; and an
    administrator suspending an event to stop further sales.

- Outcome: Final test position
  - Status: Done
  - Evidence or result: Go unit and integration suites green against PostgreSQL,
    the SQL schema suite green, the web typecheck, lint and unit tests green, and
    all ten phase specifications passing.

## Commitments for the next two weeks

The project ends with this report. No further commitments; the items below are
what a following team would pick up.

- Commitment: The five deferred bonus features
  - Owner(s): Unassigned - future work
  - Due date: Not scheduled
  - Success criteria: Offline scanner synchronisation, `.ics` calendar export,
    the interactive seat map, GA4 funnel analytics and support attachments.
    Deferred deliberately since Report 4 and never started, because SRS §8 lists
    them as bonus and §11 evaluates them separately.

- Commitment: Production readiness
  - Owner(s): Unassigned - future work
  - Due date: Not scheduled
  - Success criteria: Real payment and payout integration, encryption at rest,
    and a documented backup and recovery procedure. All are outside the academic
    MVP scope; the payment layer is a clearly labelled simulation throughout.

## Team contributions and coordination

- Team member: Student A
  - Contribution this period: Character-based validation limits, upload
    constraints, the documentation pass, and this report. Across the term: the
    database, identity, permissions, the audit trail and the admin backend.
  - Evidence: <JIRA-URL>/browse/BF-95, <JIRA-URL>/browse/BF-102
  - Next responsibility: Project handover.

- Team member: Student B
  - Contribution this period: Fees and estimated payout reporting, and fixes from
    the sweep. Across the term: events, ticketing, the checkout transaction,
    PDFs, refunds and campaigns.
  - Evidence: <JIRA-URL>/browse/BF-96, <REPO-URL>/pull/65
  - Next responsibility: Handover notes on the checkout and refund transactions.

- Team member: Student C
  - Contribution this period: Closing the localisation gaps. Across the term: the
    whole attendee-facing web experience and the Kazakh and Russian dictionaries.
  - Evidence: <JIRA-URL>/browse/BF-98, <REPO-URL>/pull/66
  - Next responsibility: Handover notes on the i18n structure.

- Team member: Student D
  - Contribution this period: Reserved-stock figures in the dashboard and demo
    support. Across the term: the organizer dashboard, analytics, the support
    desk and the admin portal.
  - Evidence: <JIRA-URL>/browse/BF-99
  - Next responsibility: Handover notes on the analytics queries.

- Team member: Student E
  - Contribution this period: The Phase 10 specification, the full regression
    sweep that found the three final defects, and the release tag. Across the
    term: the scanner app, Docker and CI, and all ten specifications.
  - Evidence: <JIRA-URL>/browse/BF-97, <JIRA-URL>/browse/BF-100
  - Next responsibility: Handover of the test suite and release process.

## Risks, blockers, and decisions needed

None outstanding. The project is delivered and tagged.

For a following team, two items are worth stating plainly rather than leaving
implied:

- The payment layer is a simulation, labelled as such in the interface and the
  database. It must not be mistaken for a real integration.
- Encryption at rest and a backup procedure are specified in SRS §7 but are
  deployment concerns with no artefact in this repository. They remain open.

## Changes, reflection, and support

- Scope or schedule changes: None this period. Across the term the only scope
  change was the deliberate deferral of the five bonus features in Report 4,
  which was never reversed.
- Team reflection: Three things we would repeat. Writing the whole database
  schema in week 1, so constraints shaped the code instead of being retrofitted.
  Writing each phase specification before the feature it tests, so the code had a
  target rather than the test being written to match whatever the code did.
  Reviewing across lanes, so no part of the system had only one person who
  understood it. One thing we would change: our first sprint's commits bunched
  at the end of the week, and we only fixed that after Report 1 - the weekly
  minimum should have been agreed on day one.
- Instructor/TA help requested: None. Thank you for the term.
