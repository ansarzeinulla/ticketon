# Biweekly Team Progress Report 1

CSCI 361 - Fall 2026

## Report details

- Team name: <TEAM-NAME>
- Reporting period: September 1 - September 13, 2026
- Submitted by: <STUDENT-A>
- Current stage: Discovery
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

The team closed one phase, Ф0, in which the team chose the product, the stack and how it will be deployed, and wrote the requirements down before writing any code. 5 issues were opened and closed, producing 27 files. No application code was written on purpose: the point of this phase was to agree what to build before building it. The team is on track.

## Phases closed this period

A phase is the unit of work, not the week. Each phase opens with every issue created up front, and closes only when all of them are merged; the next phase starts after that.

### Ф0 - Discovery

**What this phase adds to the project:** the team chose the product, the stack and how it will be deployed, and wrote the requirements down before writing any code.

- Issues: 5 stories, each closed by exactly one commit carrying the issue title.
- Files: 27 added, 1 modified, 0 deleted, 0 renamed.
- Lines: about +3990 / -1.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-1` | S1 | Record the delivery plan |
| `BF-2` | S2 | Sketch the API contract |
| `BF-3` | S3 | Sketch the web routes |
| `BF-4` | S5 | Add the SRS and repository skeleton |
| `BF-5` | S5 | Describe the product and the stack |

## Previous commitments

Not applicable: first report.

## Commitments for the next two weeks

- Commitment: close Ф1 - Running skeleton
  - Owners: all five (8 issues, 1-3 each)
  - Due date: end of Ф1
  - Success criteria: an empty but running system: database, API service, web app, CI and container images all start with one command; all issues merged and CI green.

- Commitment: close Ф2 - Identity and accounts
  - Owners: all five (8 issues, 1-3 each)
  - Due date: end of Ф2
  - Success criteria: accounts exist: people can register, sign in, and reset a password, and the API knows who is calling it; all issues merged and CI green.

## Team contributions and coordination

Zones do not overlap: every file in the repository has exactly one owner (`plan/ownership.map`), so no two people edit the same file and merge conflicts cannot occur. Review runs crosswise - each person reviews a fixed teammate's pull requests.

- Team member: <STUDENT-A> (S1)
  - Contribution this period: BF-1 Record the delivery plan.
  - Evidence: 20 files added, 0 modified, 0 deleted, 0 renamed; about +2639/-0 lines.
  - Next responsibility: BF-6, BF-7, BF-8 in Ф1.

- Team member: <STUDENT-B> (S2)
  - Contribution this period: BF-2 Sketch the API contract.
  - Evidence: 1 files added, 0 modified, 0 deleted, 0 renamed; about +166/-0 lines.
  - Next responsibility: BF-9 in Ф1.

- Team member: <STUDENT-C> (S3)
  - Contribution this period: BF-3 Sketch the web routes.
  - Evidence: 1 files added, 0 modified, 0 deleted, 0 renamed; about +170/-0 lines.
  - Next responsibility: BF-10 in Ф1.

- Team member: <STUDENT-D> (S4)
  - Contribution this period: took part in the phase review and the design decisions; no issue of their own in this short phase.
  - Evidence: phase review notes on the board.
  - Next responsibility: BF-11 in Ф1.

- Team member: <STUDENT-E> (S5)
  - Contribution this period: BF-4 Add the SRS and repository skeleton; BF-5 Describe the product and the stack.
  - Evidence: 5 files added, 1 modified, 0 deleted, 0 renamed; about +1015/-1 lines.
  - Next responsibility: BF-12, BF-13 in Ф1.

## Risks, blockers, and decisions needed

- Risk: the scope in the requirements is larger than one semester.
  - Impact: the team could run out of time before the gate works end to end.
  - Next action: the four bonus items (offline sync, .ics export, the interactive seat map, GA4) are marked as the first things to cut.
  - Owner: <STUDENT-A>

## Changes, reflection, and support

- Scope or schedule changes: None. This is the first report.
- Team reflection: Deciding the file ownership map before writing code took a full evening and felt slow. It is what makes the next ten weeks conflict-free, so it was worth it.
- Instructor/TA help requested: None.
