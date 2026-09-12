# Biweekly Team Progress Report 2

CSCI 361 - Fall 2026

## Report details

- Team name: <TEAM-NAME>
- Reporting period: September 14 - September 27, 2026
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

The team closed 2 phases, Ф1 and Ф2. In Ф1 an empty but running system: database, API service, web app, CI and container images all start with one command; in Ф2 accounts exist: people can register, sign in, and reset a password, and the API knows who is calling it. 16 issues were closed across the two phases: 95 files added, 21 modified and 3 deleted. The team is on track.

## Phases closed this period

A phase is the unit of work, not the week. Each phase opens with every issue created up front, and closes only when all of them are merged; the next phase starts after that.

### Ф1 - Running skeleton

**What this phase adds to the project:** an empty but running system: database, API service, web app, CI and container images all start with one command.

- Issues: 8 stories, each closed by exactly one commit carrying the issue title.
- Files: 54 added, 5 modified, 0 deleted, 0 renamed.
- Lines: about +5002 / -69.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-6` | S1 | Add extensions and the users table |
| `BF-7` | S1 | Add config and the error envelope |
| `BF-8` | S1 | Add the pool and a health route |
| `BF-9` | S2 | Document the API layout |
| `BF-10` | S3 | Scaffold Next.js |
| `BF-11` | S4 | Add the signed-in shell |
| `BF-12` | S5 | Add Postgres to compose |
| `BF-13` | S5 | Add CI |

### Ф2 - Identity and accounts

**What this phase adds to the project:** accounts exist: people can register, sign in, and reset a password, and the API knows who is calling it.

- Issues: 8 stories, each closed by exactly one commit carrying the issue title.
- Files: 41 added, 16 modified, 3 deleted, 1 renamed.
- Lines: about +6152 / -759.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-14` | S1 | Hash passwords with bcrypt |
| `BF-15` | S1 | Reject duplicate emails with 409 |
| `BF-16` | S1 | Cover users, tokens and mail |
| `BF-17` | S2 | Document the authentication endpoints |
| `BF-18` | S3 | Add the API client |
| `BF-19` | S3 | Add reset and verify pages |
| `BF-20` | S4 | Keep the session in the browser |
| `BF-21` | S5 | Add the Phase 1 specification |

## Previous commitments

- Previous commitment: close Ф0 - Discovery
  - Status: Completed
  - Result: The team chose the product, the stack and how it will be deployed, and wrote the requirements down before writing any code.
  - Evidence: branch `phase-00`, issues BF-1-BF-5 on the board.

## Commitments for the next two weeks

- Commitment: close Ф3 - Events and ticket types
  - Owners: all five (7 issues, 1-3 each)
  - Due date: end of Ф3
  - Success criteria: organizers can create events and ticket types and move an event through its lifecycle; all issues merged and CI green.

- Commitment: close Ф4 - Public catalogue and free registration
  - Owners: all five (8 issues, 1-3 each)
  - Due date: end of Ф4
  - Success criteria: the first end-to-end path works: a visitor browses the public catalogue and registers for a free event; all issues merged and CI green.

## Team contributions and coordination

Zones do not overlap: every file in the repository has exactly one owner (`plan/ownership.map`), so no two people edit the same file and merge conflicts cannot occur. Review runs crosswise - each person reviews a fixed teammate's pull requests.

- Team member: <STUDENT-A> (S1)
  - Contribution this period: BF-6 Add extensions and the users table; BF-7 Add config and the error envelope; BF-8 Add the pool and a health route; BF-14 Hash passwords with bcrypt; BF-15 Reject duplicate emails with 409; BF-16 Cover users, tokens and mail.
  - Evidence: 43 files added, 10 modified, 3 deleted, 1 renamed; about +4678/-287 lines.
  - Next responsibility: BF-22 in Ф3.

- Team member: <STUDENT-B> (S2)
  - Contribution this period: BF-9 Document the API layout; BF-17 Document the authentication endpoints.
  - Evidence: 0 files added, 2 modified, 0 deleted, 0 renamed; about +166/-50 lines.
  - Next responsibility: BF-23, BF-24 in Ф3.

- Team member: <STUDENT-C> (S3)
  - Contribution this period: BF-10 Scaffold Next.js; BF-18 Add the API client; BF-19 Add reset and verify pages.
  - Evidence: 35 files added, 5 modified, 0 deleted, 0 renamed; about +5451/-456 lines.
  - Next responsibility: BF-25 in Ф3.

- Team member: <STUDENT-D> (S4)
  - Contribution this period: BF-11 Add the signed-in shell; BF-20 Keep the session in the browser.
  - Evidence: 6 files added, 0 modified, 0 deleted, 0 renamed; about +272/-0 lines.
  - Next responsibility: BF-26, BF-27 in Ф3.

- Team member: <STUDENT-E> (S5)
  - Contribution this period: BF-12 Add Postgres to compose; BF-13 Add CI; BF-21 Add the Phase 1 specification.
  - Evidence: 11 files added, 4 modified, 0 deleted, 0 renamed; about +587/-35 lines.
  - Next responsibility: BF-28 in Ф3.

## Risks, blockers, and decisions needed

- None.

## Changes, reflection, and support

- Scope or schedule changes: None.
- Team reflection: Opening every issue before the phase starts felt bureaucratic on day one and paid for itself by day three: nobody had to ask what to work on.
- Instructor/TA help requested: None.
