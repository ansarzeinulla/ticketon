# Biweekly Team Progress Report

CSCI 361 - Fall 2026

**Cover page**

## Report details

- Team name: <TEAM-NAME>
- Reporting period: November 9, 2026 - November 22, 2026
- Submitted by: Student A
- Current stage: Test
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

Every required MVP feature in the SRS is now built. Organizers can see accurate
analytics computed live from operational rows, run promotional campaigns with QR
codes, and moderate the platform from an admin portal; the interface speaks
Kazakh and Russian. Phases 7, 8 and 9 pass. With one week left we are moving
from building to hardening, and we finished the sprint with no required work
outstanding - so the final week is genuinely available for testing and the demo.

## Progress this period

### Previous commitments

- Previous commitment: Organizer analytics computed from operational rows
  - Status: Completed
  - Result or reason: Every figure is a query against orders, tickets, refunds
    and check-in records - no counters that could drift. We verified each stat
    card against a direct SQL query and they agree to the tiyn.
  - Evidence: <JIRA-URL>/browse/BF-84, <JIRA-URL>/browse/BF-85, <REPO-URL>/pull/56

- Previous commitment: Sales over time, by tier and by campaign
  - Status: Completed
  - Result or reason: A sales chart built from order timestamps, a per-tier
    breakdown and campaign attribution, with date-range and ticket-type filters.
    The chart is plain CSS, so no third-party charting bundle is shipped.
  - Evidence: <JIRA-URL>/browse/BF-86, <REPO-URL>/pull/58

- Previous commitment: Promo codes and campaign QR codes
  - Status: Completed
  - Result or reason: A campaign issues an opaque `CMP_` token and a downloadable
    QR PNG. The link carries no price or discount value - the server decides the
    discount - and redemption limits are enforced with a row lock, so a code with
    one use left cannot be redeemed twice concurrently. A fixed discount larger
    than the ticket is capped at the line price rather than going negative.
  - Evidence: <JIRA-URL>/browse/BF-80, <JIRA-URL>/browse/BF-82, <REPO-URL>/pull/54

- Previous commitment: Kazakh and Russian across the customer-facing interface
  - Status: Completed
  - Result or reason: A switcher offering Қазақша, Русский and English. The
    locale is resolved on the server and stored in a cookie, so the whole page
    re-renders in the chosen language rather than only the interactive parts.
  - Evidence: <JIRA-URL>/browse/BF-90, <JIRA-URL>/browse/BF-91, <REPO-URL>/pull/60

- Previous commitment: The platform admin portal and CSV export
  - Status: Completed
  - Result or reason: Administrators only, enforced by the API on every request
    rather than by hiding buttons. The CSV escapes commas and quotes per RFC 4180
    and is served `Cache-Control: no-store`.
  - Evidence: <JIRA-URL>/browse/BF-88, <JIRA-URL>/browse/BF-92, <REPO-URL>/pull/59

- Previous commitment: Phase 7, 8 and 9 specifications and results
  - Status: Completed
  - Result or reason: `7.md`, `8.md` and `9.md` written and executed; all cases pass.
  - Evidence: <REPO-URL>/blob/main/7.md, <REPO-URL>/blob/main/9.md

### Other progress

- Outcome: Event duplication for a new season
  - Status: Done
  - Evidence or result: An organizer duplicates a finished event into a fresh
    draft. Settings and ticket tiers carry over with sold counts reset to zero;
    orders, tickets, payments, check-ins, campaigns and support cases are
    deliberately excluded, so last season's attendees are not implied to hold
    tickets for the new date. (BF-89)

- Outcome: Timeline filtering
  - Status: Done
  - Evidence or result: The activity timeline filters by activity type and date
    range, with the end date treated as end-of-day. (BF-79)

- Outcome: All ten epics closed except release
  - Status: Done
  - Evidence or result: `BF-E1` … `BF-E9` closed. `BF-E10` remains open for the
    release work in the final week.

## Commitments for the next two weeks

The final period is one week - Monday 23 to Saturday 28 November - and the term
ends with the demo. No new features after Wednesday.

- Commitment: Validation hardening across every text field
  - Owner(s): Student A
  - Due date: November 24, 2026
  - Success criteria: Length limits are counted in characters rather than bytes,
    so Kazakh and Russian input is not silently given half the room; uploads are
    bounded by size and pixel dimensions.

- Commitment: Fees and estimated payout visible to organizers
  - Owner(s): Student B
  - Due date: November 24, 2026
  - Success criteria: The dashboard shows what the platform charged and what the
    organizer would be paid, alongside gross and refunds, per SRS 4.6.

- Commitment: Remaining localisation gaps closed
  - Owner(s): Student C
  - Due date: November 25, 2026
  - Success criteria: Every customer-facing page renders in Kazakh and Russian
    with no English strings left behind.

- Commitment: Full regression sweep, Phases 1 to 10
  - Owner(s): Student E
  - Due date: November 25, 2026
  - Success criteria: All ten phase specifications executed against a freshly
    seeded database; every failure either fixed or recorded with a reason.

- Commitment: Documentation, demo rehearsal and release tag
  - Owner(s): Student A (docs), all (rehearsal), Student E (tag)
  - Due date: November 27, 2026
  - Success criteria: `README.md` and the API collection current; the demo runs
    end to end from `make reset`; `v1.0.0` tagged.

## Team contributions and coordination

- Team member: Student A
  - Contribution this period: Timeline filtering by type and date range, and the
    operational CSV export with RFC 4180 escaping.
  - Evidence: <JIRA-URL>/browse/BF-79, <JIRA-URL>/browse/BF-88, <REPO-URL>/pull/59
  - Next responsibility: Validation hardening and the documentation pass.

- Team member: Student B
  - Contribution this period: Campaigns, promo codes, campaign QR tokens, atomic
    discount capping and redemption limits, and event duplication.
  - Evidence: <JIRA-URL>/browse/BF-80, <JIRA-URL>/browse/BF-89, <REPO-URL>/pull/54
  - Next responsibility: Fees and estimated payout in the analytics.

- Team member: Student C
  - Contribution this period: The promo box with debounced pricing, campaign link
    handling, and the Kazakh, Russian and English dictionaries with the switcher.
  - Evidence: <JIRA-URL>/browse/BF-83, <JIRA-URL>/browse/BF-90, <REPO-URL>/pull/60
  - Next responsibility: Closing the remaining localisation gaps.

- Team member: Student D
  - Contribution this period: The analytics queries and the dashboard - capacity,
    revenue, attendance, per-tier and per-campaign breakdowns - plus the admin
    portal interface.
  - Evidence: <JIRA-URL>/browse/BF-84, <JIRA-URL>/browse/BF-92, <REPO-URL>/pull/56
  - Next responsibility: Reserved-stock figures and supporting the regression sweep.

- Team member: Student E
  - Contribution this period: The Phase 7, 8 and 9 specifications and their
    execution, including the cross-organizer isolation and admin-only checks.
  - Evidence: <JIRA-URL>/browse/BF-93, <JIRA-URL>/browse/BF-94
  - Next responsibility: The Phase 10 specification and the full regression sweep.

## Risks, blockers, and decisions needed

- Risk: One week remains and it contains hardening, documentation, rehearsal and
  the demo.
  - Impact: A late defect could compete with rehearsal time.
  - Next action: A hard feature freeze at 18:00 on Wednesday 25 November.
    Thursday and Friday are fixes and rehearsal only. Agreed by the whole team.
  - Owner: Student A

- Risk: The bonus features were deferred all term and remain unbuilt.
  - Impact: None to the grade for the base MVP - SRS §8 lists them as bonus and
    §11 states bonus features are evaluated separately.
  - Next action: Leave them unbuilt. Attempting the offline scanner in the final
    week would put the required regression sweep at risk for no required credit.
  - Owner: Student E

## Changes, reflection, and support

- Scope or schedule changes: None to required scope. The five bonus features
  remain deliberately out of scope, as recorded since Report 4.
- Team reflection: Starting the analytics queries a week early, and writing
  `7.md` before the dashboard existed, meant the numbers had a target rather than
  the specification being written to match whatever the code produced. That
  ordering is the single practice most worth carrying into other projects.
  Change: nothing structural this late - the process is working.
- Instructor/TA help requested: Confirmation of the expected demo format and
  duration for the final session would help us rehearse to the right length.
