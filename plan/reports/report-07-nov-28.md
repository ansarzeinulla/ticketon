# Biweekly Team Progress Report 7

CSCI 361 - Fall 2026

## Report details

- Team name: <TEAM-NAME>
- Reporting period: November 23 - November 28, 2026
- Submitted by: <STUDENT-A>
- Current stage: Test
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

The team closed 2 phases, Ф11 and Ф12. In Ф11 verification: every phase specification is run against the running system and the failures found are fixed; in Ф12 documentation, a demo rehearsal and the release tag. 13 issues were closed across the two phases: 31 files added, 40 modified and 0 deleted. The team is on track.

## Phases closed this period

A phase is the unit of work, not the week. Each phase opens with every issue created up front, and closes only when all of them are merged; the next phase starts after that.

### Ф11 - Verification

**What this phase adds to the project:** verification: every phase specification is run against the running system and the failures found are fixed.

- Issues: 8 stories, each closed by exactly one commit carrying the issue title.
- Files: 31 added, 24 modified, 0 deleted, 0 renamed.
- Lines: about +15194 / -834.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-89` | S1 | Say "required" instead of "too short" |
| `BF-90` | S2 | Refuse to publish an event that has ended |
| `BF-91` | S3 | Close the localisation gaps |
| `BF-92` | S3 | Add the test runner dependencies |
| `BF-93` | S4 | Make the money card answer the tier filter |
| `BF-94` | S5 | Run the phase specifications and record results |
| `BF-95` | S5 | Correct the specifications after the run |
| `BF-96` | S5 | Add the Postman collection |

### Ф12 - Documentation and release

**What this phase adds to the project:** documentation, a demo rehearsal and the release tag.

- Issues: 5 stories, each closed by exactly one commit carrying the issue title.
- Files: 0 added, 16 modified, 0 deleted, 0 renamed.
- Lines: about +934 / -282.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-97` | S1 | Record the final plan state |
| `BF-98` | S2 | Update the API documentation |
| `BF-99` | S3 | Update the web documentation |
| `BF-100` | S5 | Update the README and run instructions |
| `BF-101` | S5 | Tag v1.0.0 |

## Previous commitments

- Previous commitment: close Ф9 - Analytics, campaigns, localisation
  - Status: Completed
  - Result: Organizers see numbers and run campaigns, and the site speaks Kazakh and Russian.
  - Evidence: branch `phase-09`, issues BF-71-BF-79 on the board.

- Previous commitment: close Ф10 - Shipping shape
  - Status: Completed
  - Result: The product reaches the shape it is meant to ship in; character limits, payout figures and the remaining defects are settled.
  - Evidence: branch `phase-10`, issues BF-80-BF-88 on the board.

## Commitments for the next two weeks

- Commitment: deliver the demo and hand in the final report
  - Owners: all five
  - Due date: November 28, 2026
  - Success criteria: every phase specification passes on a clean checkout, and the demo runs end to end without a manual fix.

## Team contributions and coordination

Zones do not overlap: every file in the repository has exactly one owner (`plan/ownership.map`), so no two people edit the same file and merge conflicts cannot occur. Review runs crosswise - each person reviews a fixed teammate's pull requests.

- Team member: <STUDENT-A> (S1)
  - Contribution this period: BF-89 Say "required" instead of "too short"; BF-97 Record the final plan state.
  - Evidence: 1 files added, 14 modified, 0 deleted, 0 renamed; about +1067/-274 lines.
  - Next responsibility: the demo.

- Team member: <STUDENT-B> (S2)
  - Contribution this period: BF-90 Refuse to publish an event that has ended; BF-98 Update the API documentation.
  - Evidence: 0 files added, 3 modified, 0 deleted, 0 renamed; about +477/-143 lines.
  - Next responsibility: the demo.

- Team member: <STUDENT-C> (S3)
  - Contribution this period: BF-91 Close the localisation gaps; BF-92 Add the test runner dependencies; BF-99 Update the web documentation.
  - Evidence: 3 files added, 6 modified, 0 deleted, 0 renamed; about +1590/-436 lines.
  - Next responsibility: the demo.

- Team member: <STUDENT-D> (S4)
  - Contribution this period: BF-93 Make the money card answer the tier filter.
  - Evidence: 0 files added, 2 modified, 0 deleted, 0 renamed; about +316/-95 lines.
  - Next responsibility: the demo.

- Team member: <STUDENT-E> (S5)
  - Contribution this period: BF-94 Run the phase specifications and record results; BF-95 Correct the specifications after the run; BF-96 Add the Postman collection; BF-100 Update the README and run instructions; BF-101 Tag v1.0.0.
  - Evidence: 27 files added, 15 modified, 0 deleted, 0 renamed; about +12678/-168 lines.
  - Next responsibility: the demo.

## Risks, blockers, and decisions needed

- None.

## Changes, reflection, and support

- Scope or schedule changes: None.
- Team reflection: The thing that saved the semester was the ownership map: in thirteen phases the team never resolved a merge conflict.
- Instructor/TA help requested: None.
