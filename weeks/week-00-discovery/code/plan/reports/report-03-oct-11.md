# Biweekly Team Progress Report 3

CSCI 361 - Fall 2026

## Report details

- Team name: <TEAM-NAME>
- Reporting period: September 28 - October 11, 2026
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

The team closed 2 phases, Ф3 and Ф4. In Ф3 organizers can create events and ticket types and move an event through its lifecycle; in Ф4 the first end-to-end path works: a visitor browses the public catalogue and registers for a free event. 15 issues were closed across the two phases: 47 files added, 25 modified and 5 deleted. The team is on track.

## Phases closed this period

A phase is the unit of work, not the week. Each phase opens with every issue created up front, and closes only when all of them are merged; the next phase starts after that.

### Ф3 - Events and ticket types

**What this phase adds to the project:** organizers can create events and ticket types and move an event through its lifecycle.

- Issues: 7 stories, each closed by exactly one commit carrying the issue title.
- Files: 23 added, 11 modified, 2 deleted, 0 renamed.
- Lines: about +4065 / -291.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-22` | S1 | Seed a demo organizer and event |
| `BF-23` | S2 | Derive a slug from the title |
| `BF-24` | S2 | Manage ticket types |
| `BF-25` | S3 | Add event types and client methods |
| `BF-26` | S4 | Move the create form to the client |
| `BF-27` | S4 | Add the organizer dashboard |
| `BF-28` | S5 | Add the Phase 2 specification |

### Ф4 - Public catalogue and free registration

**What this phase adds to the project:** the first end-to-end path works: a visitor browses the public catalogue and registers for a free event.

- Issues: 8 stories, each closed by exactly one commit carrying the issue title.
- Files: 24 added, 14 modified, 3 deleted, 0 renamed.
- Lines: about +4497 / -788.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-29` | S1 | Add order, ticket and attendee tables |
| `BF-30` | S2 | Serve the public event page |
| `BF-31` | S2 | Add the checkout transaction |
| `BF-32` | S3 | Add the public catalogue |
| `BF-33` | S3 | Add the order page |
| `BF-34` | S3 | Add the formatting dependencies |
| `BF-35` | S4 | List orders and attendees |
| `BF-36` | S5 | Add the Phase 3 specification |

## Previous commitments

- Previous commitment: close Ф1 - Running skeleton
  - Status: Completed
  - Result: An empty but running system: database, API service, web app, CI and container images all start with one command.
  - Evidence: branch `phase-01`, issues BF-6-BF-13 on the board.

- Previous commitment: close Ф2 - Identity and accounts
  - Status: Completed
  - Result: Accounts exist: people can register, sign in, and reset a password, and the API knows who is calling it.
  - Evidence: branch `phase-02`, issues BF-14-BF-21 on the board.

## Commitments for the next two weeks

- Commitment: close Ф5 - Paid sales
  - Owners: all five (8 issues, 1-3 each)
  - Due date: end of Ф5
  - Success criteria: money enters the picture: activation gates paid sales, inventory holds under concurrent checkout, and payment is simulated; all issues merged and CI green.

- Commitment: close Ф6 - Digital tickets
  - Owners: all five (7 issues, 1-3 each)
  - Due date: end of Ф6
  - Success criteria: a paid order produces a real ticket: QR token, printable A4 PDF, and a confirmation email; all issues merged and CI green.

## Team contributions and coordination

Zones do not overlap: every file in the repository has exactly one owner (`plan/ownership.map`), so no two people edit the same file and merge conflicts cannot occur. Review runs crosswise - each person reviews a fixed teammate's pull requests.

- Team member: <STUDENT-A> (S1)
  - Contribution this period: BF-22 Seed a demo organizer and event; BF-29 Add order, ticket and attendee tables.
  - Evidence: 2 files added, 4 modified, 1 deleted, 0 renamed; about +642/-111 lines.
  - Next responsibility: BF-37, BF-38 in Ф5.

- Team member: <STUDENT-B> (S2)
  - Contribution this period: BF-23 Derive a slug from the title; BF-24 Manage ticket types; BF-30 Serve the public event page; BF-31 Add the checkout transaction.
  - Evidence: 15 files added, 5 modified, 0 deleted, 0 renamed; about +3205/-165 lines.
  - Next responsibility: BF-39, BF-40 in Ф5.

- Team member: <STUDENT-C> (S3)
  - Contribution this period: BF-25 Add event types and client methods; BF-32 Add the public catalogue; BF-33 Add the order page; BF-34 Add the formatting dependencies.
  - Evidence: 14 files added, 11 modified, 3 deleted, 0 renamed; about +2931/-720 lines.
  - Next responsibility: BF-41, BF-42 in Ф5.

- Team member: <STUDENT-D> (S4)
  - Contribution this period: BF-26 Move the create form to the client; BF-27 Add the organizer dashboard; BF-35 List orders and attendees.
  - Evidence: 14 files added, 2 modified, 0 deleted, 0 renamed; about +1458/-2 lines.
  - Next responsibility: BF-43 in Ф5.

- Team member: <STUDENT-E> (S5)
  - Contribution this period: BF-28 Add the Phase 2 specification; BF-36 Add the Phase 3 specification.
  - Evidence: 2 files added, 3 modified, 1 deleted, 0 renamed; about +326/-81 lines.
  - Next responsibility: BF-44 in Ф5.

## Risks, blockers, and decisions needed

- None.

## Changes, reflection, and support

- Scope or schedule changes: None.
- Team reflection: The first end-to-end path took longer than planned because the client and the API were built in parallel. Fixing the merge order inside a phase (schema, then handlers, then routes, then types, then UI) solved it.
- Instructor/TA help requested: None.
