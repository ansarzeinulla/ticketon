# Biweekly Team Progress Report 6

CSCI 361 - Fall 2026

## Report details

- Team name: <TEAM-NAME>
- Reporting period: November 9 - November 22, 2026
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

The team closed 2 phases, Ф9 and Ф10. In Ф9 organizers see numbers and run campaigns, and the site speaks Kazakh and Russian; in Ф10 the product reaches the shape it is meant to ship in; character limits, payout figures and the remaining defects are settled. 25 issues were closed across the two phases: 41 files added, 180 modified and 6 deleted. The team is on track.

## Phases closed this period

A phase is the unit of work, not the week. Each phase opens with every issue created up front, and closes only when all of them are merged; the next phase starts after that.

### Ф9 - Analytics, campaigns, localisation

**What this phase adds to the project:** organizers see numbers and run campaigns, and the site speaks Kazakh and Russian.

- Issues: 9 stories and 1 defects, each closed by exactly one commit carrying the issue title.
- Files: 25 added, 82 modified, 1 deleted, 0 renamed.
- Lines: about +8096 / -1493.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-71` | S1 | Add campaign tables |
| `BF-72` | S2 | Create campaigns and promo codes |
| `BF-73` | S2 | Apply and cap discounts atomically |
| `BF-74` | S3 | Add the English dictionary |
| `BF-75` | S3 | Localise the attendee pages |
| `BF-76` | S3 | Add the language negotiation dependency |
| `BF-77` | S4 | Compute analytics from operational rows |
| `BF-78` | S4 | Localise the dashboard |
| `BF-B4` | S4 | Read the GA4 id from a server-safe module |
| `BF-79` | S5 | Add the Phase 7 specification |

Defects found and fixed inside Ф9: `BF-B4`. Each was filed the day it was found, linked to the story that introduced it, and fixed under its own key.

### Ф10 - Shipping shape

**What this phase adds to the project:** the product reaches the shape it is meant to ship in; character limits, payout figures and the remaining defects are settled.

- Issues: 9 stories and 6 defects, each closed by exactly one commit carrying the issue title.
- Files: 16 added, 98 modified, 5 deleted, 2 renamed.
- Lines: about +10751 / -2781.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-80` | S1 | Count text limits in characters, not bytes |
| `BF-B5` | S1 | Stop sharing notification state on the server |
| `BF-B6` | S1 | Make the seed re-runnable after a purchase |
| `BF-81` | S1 | Count text limits in the remaining handlers |
| `BF-82` | S2 | Count text limits in commerce handlers |
| `BF-B7` | S2 | Give a suspended event its own code |
| `BF-B8` | S2 | Name the refund conflict |
| `BF-83` | S2 | Cut Cyrillic text by characters on the ticket |
| `BF-84` | S3 | Localise the remaining attendee screens |
| `BF-85` | S4 | Show reserved stock |
| `BF-B9` | S4 | Name the reversal conflict |
| `BF-86` | S4 | Localise the remaining dashboard screens |
| `BF-B10` | S5 | Name the check-in reversal conflict |
| `BF-87` | S5 | Add the Phase 8, 9 and 10 specifications |
| `BF-88` | S5 | Point the scanner at the queue |

Defects found and fixed inside Ф10: `BF-B5`, `BF-B6`, `BF-B7`, `BF-B8`, `BF-B9`, `BF-B10`. Each was filed the day it was found, linked to the story that introduced it, and fixed under its own key.

## Previous commitments

- Previous commitment: close Ф7 - Check-in and the gate
  - Status: Completed
  - Result: The gate works: the Expo scanner admits a ticket once and refuses the second attempt.
  - Evidence: branch `phase-07`, issues BF-52-BF-59 on the board.

- Previous commitment: close Ф8 - Refunds, support and administration
  - Status: Completed
  - Result: What happens when things go wrong: refunds, cancellations, the support desk and the admin portal.
  - Evidence: branch `phase-08`, issues BF-60-BF-70 on the board.

## Commitments for the next two weeks

- Commitment: close Ф11 - Verification
  - Owners: all five (8 issues, 1-3 each)
  - Due date: end of Ф11
  - Success criteria: verification: every phase specification is run against the running system and the failures found are fixed; all issues merged and CI green.

- Commitment: close Ф12 - Documentation and release
  - Owners: all five (5 issues, 1-3 each)
  - Due date: end of Ф12
  - Success criteria: documentation, a demo rehearsal and the release tag; all issues merged and CI green.

## Team contributions and coordination

Zones do not overlap: every file in the repository has exactly one owner (`plan/ownership.map`), so no two people edit the same file and merge conflicts cannot occur. Review runs crosswise - each person reviews a fixed teammate's pull requests.

- Team member: <STUDENT-A> (S1)
  - Contribution this period: BF-71 Add campaign tables; BF-80 Count text limits in characters, not bytes; BF-B5 Stop sharing notification state on the server; BF-B6 Make the seed re-runnable after a purchase; BF-81 Count text limits in the remaining handlers.
  - Evidence: 0 files added, 28 modified, 1 deleted, 0 renamed; about +1856/-611 lines.
  - Next responsibility: BF-89 in Ф11.

- Team member: <STUDENT-B> (S2)
  - Contribution this period: BF-72 Create campaigns and promo codes; BF-73 Apply and cap discounts atomically; BF-82 Count text limits in commerce handlers; BF-B7 Give a suspended event its own code; BF-B8 Name the refund conflict; BF-83 Cut Cyrillic text by characters on the ticket.
  - Evidence: 8 files added, 29 modified, 0 deleted, 0 renamed; about +3921/-708 lines.
  - Next responsibility: BF-90 in Ф11.

- Team member: <STUDENT-C> (S3)
  - Contribution this period: BF-74 Add the English dictionary; BF-75 Localise the attendee pages; BF-76 Add the language negotiation dependency; BF-84 Localise the remaining attendee screens.
  - Evidence: 8 files added, 51 modified, 0 deleted, 0 renamed; about +3586/-931 lines.
  - Next responsibility: BF-91, BF-92 in Ф11.

- Team member: <STUDENT-D> (S4)
  - Contribution this period: BF-77 Compute analytics from operational rows; BF-78 Localise the dashboard; BF-B4 Read the GA4 id from a server-safe module; BF-85 Show reserved stock; BF-B9 Name the reversal conflict; BF-86 Localise the remaining dashboard screens.
  - Evidence: 14 files added, 43 modified, 1 deleted, 1 renamed; about +3915/-756 lines.
  - Next responsibility: BF-93 in Ф11.

- Team member: <STUDENT-E> (S5)
  - Contribution this period: BF-79 Add the Phase 7 specification; BF-B10 Name the check-in reversal conflict; BF-87 Add the Phase 8, 9 and 10 specifications; BF-88 Point the scanner at the queue.
  - Evidence: 11 files added, 29 modified, 4 deleted, 1 renamed; about +5569/-1268 lines.
  - Next responsibility: BF-94, BF-95, BF-96 in Ф11.

## Risks, blockers, and decisions needed

- Risk: three locales multiply the number of screens to check.
  - Impact: a missing key shows an English string to a Kazakh visitor.
  - Next action: `make i18n-check` fails the build on a missing key.
  - Owner: <STUDENT-C>

## Changes, reflection, and support

- Scope or schedule changes: None. The bonus items were kept, so nothing was cut.
- Team reflection: Localisation touched almost every screen at once. Doing it in one sweep per owner was right; splitting it per screen would have produced dozens of near-identical commits.
- Instructor/TA help requested: None.
