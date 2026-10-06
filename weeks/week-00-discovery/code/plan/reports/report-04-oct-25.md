# Biweekly Team Progress Report 4

CSCI 361 - Fall 2026

## Report details

- Team name: <TEAM-NAME>
- Reporting period: October 12 - October 25, 2026
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

The team closed 2 phases, Ф5 and Ф6. In Ф5 money enters the picture: activation gates paid sales, inventory holds under concurrent checkout, and payment is simulated; in Ф6 a paid order produces a real ticket: QR token, printable A4 PDF, and a confirmation email. 15 issues were closed across the two phases: 40 files added, 39 modified and 6 deleted. The team is on track.

## Phases closed this period

A phase is the unit of work, not the week. Each phase opens with every issue created up front, and closes only when all of them are merged; the next phase starts after that.

### Ф5 - Paid sales

**What this phase adds to the project:** money enters the picture: activation gates paid sales, inventory holds under concurrent checkout, and payment is simulated.

- Issues: 8 stories, each closed by exactly one commit carrying the issue title.
- Files: 20 added, 25 modified, 4 deleted, 0 renamed.
- Lines: about +5417 / -674.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-37` | S1 | Add organizer profiles and money handling |
| `BF-38` | S1 | Enforce role permissions |
| `BF-39` | S2 | Add seat inventory |
| `BF-40` | S2 | Gate paid sales behind activation |
| `BF-41` | S3 | Show the activation notice |
| `BF-42` | S3 | Add the seat map and seated checkout |
| `BF-43` | S4 | Add the activation checklist |
| `BF-44` | S5 | Prove checkout cannot oversell |

### Ф6 - Digital tickets

**What this phase adds to the project:** a paid order produces a real ticket: QR token, printable A4 PDF, and a confirmation email.

- Issues: 7 stories, each closed by exactly one commit carrying the issue title.
- Files: 20 added, 14 modified, 2 deleted, 0 renamed.
- Lines: about +9105 / -726.

| Issue | Owner | Title (also the commit message) |
| --- | --- | --- |
| `BF-45` | S1 | Add the QR decoding dependency |
| `BF-46` | S1 | Send the order confirmation |
| `BF-47` | S2 | Generate admission QR tokens |
| `BF-48` | S2 | Embed a Unicode font for Cyrillic |
| `BF-49` | S3 | Link the printable PDF |
| `BF-50` | S4 | Show ticket status per order |
| `BF-51` | S5 | Add the Phase 4 specification |

## Previous commitments

- Previous commitment: close Ф3 - Events and ticket types
  - Status: Completed
  - Result: Organizers can create events and ticket types and move an event through its lifecycle.
  - Evidence: branch `phase-03`, issues BF-22-BF-28 on the board.

- Previous commitment: close Ф4 - Public catalogue and free registration
  - Status: Completed
  - Result: The first end-to-end path works: a visitor browses the public catalogue and registers for a free event.
  - Evidence: branch `phase-04`, issues BF-29-BF-36 on the board.

## Commitments for the next two weeks

- Commitment: close Ф7 - Check-in and the gate
  - Owners: all five (8 issues, 1-3 each)
  - Due date: end of Ф7
  - Success criteria: the gate works: the Expo scanner admits a ticket once and refuses the second attempt; all issues merged and CI green.

- Commitment: close Ф8 - Refunds, support and administration
  - Owners: all five (14 issues, 1-3 each)
  - Due date: end of Ф8
  - Success criteria: what happens when things go wrong: refunds, cancellations, the support desk and the admin portal; all issues merged and CI green.

## Team contributions and coordination

Zones do not overlap: every file in the repository has exactly one owner (`plan/ownership.map`), so no two people edit the same file and merge conflicts cannot occur. Review runs crosswise - each person reviews a fixed teammate's pull requests.

- Team member: <STUDENT-A> (S1)
  - Contribution this period: BF-37 Add organizer profiles and money handling; BF-38 Enforce role permissions; BF-45 Add the QR decoding dependency; BF-46 Send the order confirmation.
  - Evidence: 10 files added, 12 modified, 3 deleted, 0 renamed; about +2166/-357 lines.
  - Next responsibility: BF-52 in Ф7.

- Team member: <STUDENT-B> (S2)
  - Contribution this period: BF-39 Add seat inventory; BF-40 Gate paid sales behind activation; BF-47 Generate admission QR tokens; BF-48 Embed a Unicode font for Cyrillic.
  - Evidence: 21 files added, 10 modified, 2 deleted, 0 renamed; about +8238/-353 lines.
  - Next responsibility: BF-53, BF-54 in Ф7.

- Team member: <STUDENT-C> (S3)
  - Contribution this period: BF-41 Show the activation notice; BF-42 Add the seat map and seated checkout; BF-49 Link the printable PDF.
  - Evidence: 4 files added, 10 modified, 1 deleted, 0 renamed; about +2308/-598 lines.
  - Next responsibility: BF-55 in Ф7.

- Team member: <STUDENT-D> (S4)
  - Contribution this period: BF-43 Add the activation checklist; BF-50 Show ticket status per order.
  - Evidence: 3 files added, 2 modified, 0 deleted, 0 renamed; about +527/-31 lines.
  - Next responsibility: BF-56 in Ф7.

- Team member: <STUDENT-E> (S5)
  - Contribution this period: BF-44 Prove checkout cannot oversell; BF-51 Add the Phase 4 specification.
  - Evidence: 2 files added, 5 modified, 0 deleted, 0 renamed; about +1283/-61 lines.
  - Next responsibility: BF-57, BF-58, BF-59 in Ф7.

## Risks, blockers, and decisions needed

- Risk: concurrent checkout is the one place where a bug costs real money.
  - Impact: overselling a sold-out event would be visible to buyers.
  - Next action: a dedicated concurrency test runs in CI on every push.
  - Owner: <STUDENT-E>

## Changes, reflection, and support

- Scope or schedule changes: None. Both phases closed inside their planned window.
- Team reflection: Writing a file small and wrong on purpose, then rewriting it later, felt wasteful at first. It is the only way the history shows how the project actually grew.
- Instructor/TA help requested: None.
