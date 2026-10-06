# Biweekly Team Progress Report 5

CSCI 361 - Fall 2026

## Report details

- Team name: <TEAM-NAME>
- Reporting period: October 26 - November 8, 2026
- Submitted by: <STUDENT-A>
- Current stage: Build
- Overall status: On track

### Project links

- Source code repository: <REPO-URL>
- Task management tool: <JIRA-URL>

## Team members

| Full name | Student ID | Email |
| --- | --- | --- |
| <STUDENT-A> (S1) | <STUDENT-ID-A> | <EMAIL-A> |
| <STUDENT-B> (S2) | <STUDENT-ID-B> | <EMAIL-B> |
| <STUDENT-C> (S3) | <STUDENT-ID-C> | <EMAIL-C> |
| <STUDENT-D> (S4) | <STUDENT-ID-D> | <EMAIL-D> |
| <STUDENT-E> (S5) | <STUDENT-ID-E> | <EMAIL-E> |

{{< pagebreak >}}

## Progress snapshot

The team closed 2 phases, Ф7 and Ф8. In Ф7 the gate works: the Expo scanner admits a ticket once and refuses the second attempt; in Ф8 what happens when things go wrong: refunds, cancellations, the support desk and the admin portal. 22 issues were closed across the two phases: 77 files added, 66 modified and 5 deleted. The team is on track.

## Phases closed this period

A phase is the unit of work, not the week. Each phase opens with every issue created up front, and closes only when all of them are merged; the next phase starts after that.

### Ф7 - Check-in and the gate

**What this phase adds to the project:** the gate works: the Expo scanner admits a ticket once and refuses the second attempt.

- Issues: 8 stories, each closed by exactly one commit carrying the issue title.
- Files: 41 added, 23 modified, 1 deleted, 0 renamed.
- Lines: about +12709 / -891.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-52` | S1 | Update the money assertions for the fee |
| `BF-53` | S2 | Reserve stock with 15-minute holds |
| `BF-54` | S2 | Charge a processing fee |
| `BF-55` | S3 | Show the hold countdown at checkout |
| `BF-56` | S4 | Add the staff manager |
| `BF-57` | S5 | Scaffold the Expo scanner |
| `BF-58` | S5 | Scan and admit a ticket |
| `BF-59` | S5 | Document how to run the scanner |

### Ф8 - Refunds, support and administration

**What this phase adds to the project:** what happens when things go wrong: refunds, cancellations, the support desk and the admin portal.

- Issues: 11 stories and 3 defects, each closed by exactly one commit carrying the issue title.
- Files: 36 added, 43 modified, 4 deleted, 2 renamed.
- Lines: about +7722 / -947.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-60` | S1 | Add refund, support and report tables |
| `BF-61` | S1 | Add the report queue and settings |
| `BF-62` | S1 | Give every conflict its own error code |
| `BF-63` | S2 | Refund an order atomically |
| `BF-64` | S2 | Expose the rows the admin search needs |
| `BF-65` | S3 | Add the support widget to the order page |
| `BF-66` | S3 | Point the client at the proxy |
| `BF-67` | S4 | Open a support case with context |
| `BF-B2` | S4 | Move the session into an httpOnly cookie |
| `BF-B3` | S4 | Rename middleware to proxy for Next 16 |
| `BF-68` | S4 | Add the admin portal |
| `BF-69` | S5 | Refuse a refunded ticket at the gate |
| `BF-B1` | S5 | Keep the check-in record when a ticket is voided |
| `BF-70` | S5 | Add the Phase 6 specification |

Defects found and fixed inside Ф8: `BF-B2`, `BF-B3`, `BF-B1`. Each was filed the day it was found, linked to the story that introduced it, and fixed under its own key.

## Previous commitments

- Previous commitment: close Ф5 - Paid sales
  - Status: Completed
  - Result: Money enters the picture: activation gates paid sales, inventory holds under concurrent checkout, and payment is simulated.
  - Evidence: branch `phase-05`, issues BF-37-BF-44 on the board.

- Previous commitment: close Ф6 - Digital tickets
  - Status: Completed
  - Result: A paid order produces a real ticket: QR token, printable A4 PDF, and a confirmation email.
  - Evidence: branch `phase-06`, issues BF-45-BF-51 on the board.

## Commitments for the next two weeks

- Commitment: close Ф9 - Analytics, campaigns, localisation
  - Owners: all five (10 issues, 1-3 each)
  - Due date: end of Ф9
  - Success criteria: organizers see numbers and run campaigns, and the site speaks Kazakh and Russian; all issues merged and CI green.

- Commitment: close Ф10 - Shipping shape
  - Owners: all five (15 issues, 1-3 each)
  - Due date: end of Ф10
  - Success criteria: the product reaches the shape it is meant to ship in; character limits, payout figures and the remaining defects are settled; all issues merged and CI green.

## Team contributions and coordination

Zones do not overlap: every file in the repository has exactly one owner (`plan/ownership.map`), so no two people edit the same file and merge conflicts cannot occur. Review runs crosswise - each person reviews a fixed teammate's pull requests.

- Team member: <STUDENT-A> (S1)
  - Contribution this period: BF-52 Update the money assertions for the fee; BF-60 Add refund, support and report tables; BF-61 Add the report queue and settings; BF-62 Give every conflict its own error code.
  - Evidence: 11 files added, 21 modified, 1 deleted, 0 renamed; about +2982/-371 lines.
  - Next responsibility: BF-71 in Ф9.

- Team member: <STUDENT-B> (S2)
  - Contribution this period: BF-53 Reserve stock with 15-minute holds; BF-54 Charge a processing fee; BF-63 Refund an order atomically; BF-64 Expose the rows the admin search needs.
  - Evidence: 9 files added, 18 modified, 0 deleted, 0 renamed; about +3723/-670 lines.
  - Next responsibility: BF-72, BF-73 in Ф9.

- Team member: <STUDENT-C> (S3)
  - Contribution this period: BF-55 Show the hold countdown at checkout; BF-65 Add the support widget to the order page; BF-66 Point the client at the proxy.
  - Evidence: 2 files added, 6 modified, 2 deleted, 0 renamed; about +571/-192 lines.
  - Next responsibility: BF-74, BF-75, BF-76 in Ф9.

- Team member: <STUDENT-D> (S4)
  - Contribution this period: BF-56 Add the staff manager; BF-67 Open a support case with context; BF-B2 Move the session into an httpOnly cookie; BF-B3 Rename middleware to proxy for Next 16; BF-68 Add the admin portal.
  - Evidence: 17 files added, 5 modified, 1 deleted, 2 renamed; about +2281/-109 lines.
  - Next responsibility: BF-77, BF-78, BF-B4 in Ф9.

- Team member: <STUDENT-E> (S5)
  - Contribution this period: BF-57 Scaffold the Expo scanner; BF-58 Scan and admit a ticket; BF-59 Document how to run the scanner; BF-69 Refuse a refunded ticket at the gate; BF-B1 Keep the check-in record when a ticket is voided; BF-70 Add the Phase 6 specification.
  - Evidence: 38 files added, 16 modified, 1 deleted, 0 renamed; about +10874/-496 lines.
  - Next responsibility: BF-79 in Ф9.

## Risks, blockers, and decisions needed

- Risk: the scanner is tested on one Android device only.
  - Impact: camera permissions behave differently across devices.
  - Next action: record a device check in the phase specification and attach screenshots as evidence.
  - Owner: <STUDENT-E>

## Changes, reflection, and support

- Scope or schedule changes: None.
- Team reflection: Defects are now filed the day they are found instead of at the end of the phase. The board is uglier and much more honest.
- Instructor/TA help requested: None.
