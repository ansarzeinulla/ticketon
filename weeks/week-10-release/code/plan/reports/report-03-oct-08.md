# Biweekly Team Progress Report

CSCI 361 - Fall 2026

## Report details

- **Team name:** AAOAA
- **Reporting period:** September 25 - October 8, 2026
- **Report date:** October 8, 2026
- **Submitted by:** Ansar Zeinulla
- **Current stage:** Build (Week 2 and 3)
- **Overall status:** On track

### Project links

- **Source code repository:** [GitHub](https://github.com/ansarzeinulla/ticket-project)
- **Task management tool:** [Jira board](<JIRA-BOARD-URL>) — `oadiyatov@nu.edu.kz` invited as a user
- **File ownership map:** [`plan/ownership.map`](https://github.com/ansarzeinulla/ticket-project/blob/main/plan/ownership.map)
- **Delivery plan:** [`plan/README.md`](https://github.com/ansarzeinulla/ticket-project/blob/main/plan/README.md)

## Team members

| Full name | Student ID | Email |
| --- | --- | --- |
| Ansar Zeinulla (S1) | 202337329 | ansar.zeinulla@nu.edu.kz |
| Alibi Takhtanov (S2) | 202326483 | alibi.takhtanov@nu.edu.kz |
| Abylay Otaubay (S3) | 202336912 | abylay.otaubay@nu.edu.kz |
| Alinur Burlybayev (S4) | 202399031 | alinur.burlybayev@nu.edu.kz |
| Olzhas Nurseit (S5) | 202270951 | olzhas.nurseit@nu.edu.kz |

{{< pagebreak >}}

## Progress snapshot

The team closed week 2 and 3. In week 2, accounts exist: people register, sign in and reset a password, and the Expo app runs on a device and signs in against the same API; In week 3, the first end-to-end path: an organizer creates an event and a visitor registers for it from the public catalogue. 23 issues were closed, each by exactly one commit and one reviewed pull request. The team is on track.

## Progress this period

### Previous commitments

- **Previous commitment:** Close Week 1 (Running skeleton)
  - **Status:** Completed
  - **Result or reason:** An empty but running system: the database, API service, web app, CI pipeline and container images all start with one command.
  - **Evidence:** [branch `week-01`](https://github.com/ansarzeinulla/ticket-project/tree/week-01), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-01)

### Other progress

- **Outcome or deliverable:** Closed Week 2 (Identity and the mobile shell), 21–27 сентября
  - **Status:** Done
  - **Evidence or result:** 9 issues closed. [Commits on `week-02`](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-02).
  - **Details:** `S1` closed [`BF-14`](<JIRA-BOARD-URL>) Hash passwords with bcrypt; `S1` closed [`BF-15`](<JIRA-BOARD-URL>) Reject duplicate emails with 409; `S1` closed [`BF-16`](<JIRA-BOARD-URL>) Cover users, tokens and mail; `S2` closed [`BF-17`](<JIRA-BOARD-URL>) Document the authentication endpoints; and 5 more issues in the same week.

- **Outcome or deliverable:** Closed Week 3 (Events, catalogue and free registration), 28 сентября – 4 октября
  - **Status:** Done
  - **Evidence or result:** 14 issues closed. [Commits on `week-03`](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-03).
  - **Details:** `S1` closed [`BF-23`](<JIRA-BOARD-URL>) Seed a demo organizer and event; `S2` closed [`BF-25`](<JIRA-BOARD-URL>) Derive a slug from the title; `S2` closed [`BF-26`](<JIRA-BOARD-URL>) Manage ticket types; `S3` closed [`BF-28`](<JIRA-BOARD-URL>) Add the formatting dependencies; and 10 more issues in the same week.

## Commitments for the next two weeks

- **Commitment or milestone:** Close Week 4 (Paid sales and digital tickets)
  - **Owner(s):** All five team members (13 issues, 1-3 each)
  - **Due date:** 11 октября, 2026
  - **Success criteria:** money and tickets: activation gates paid sales, stock holds under concurrent checkout, and a paid order produces a QR ticket and a printable PDF; all issues merged and CI green.

- **Commitment or milestone:** Close Week 5 (Check-in and the gate)
  - **Owner(s):** All five team members (8 issues, 1-3 each)
  - **Due date:** 18 октября, 2026
  - **Success criteria:** the gate works: the scanner admits a ticket once and refuses the second attempt; all issues merged and CI green.

- **Commitment or milestone:** Mobile - the scanner admits a ticket and refuses a second scan
  - **Owner(s):** Olzhas Nurseit (S5)
  - **Due date:** 18 октября, 2026
  - **Success criteria:** a QR from a real order is accepted once; the repeat shows `already used`.

## Team contributions and coordination

*Note: zones do not overlap. Every file in the repository has exactly one owner ([`plan/ownership.map`](https://github.com/ansarzeinulla/ticket-project/blob/main/plan/ownership.map)), so two people never edit the same file. When a change needs a file owned by someone else, the requester raises it with the owner instead of editing it. All pull requests are reviewed and merged by S1 (Captain).*

- **Team member:** Ansar Zeinulla (S1)
  - **Contribution this period:** completed BF-14 Hash passwords with bcrypt, BF-15 Reject duplicate emails with 409, BF-16 Cover users, tokens and mail, BF-23 Seed a demo organizer and event, BF-24 Add order, ticket and attendee tables.
  - **Evidence:** [BF-14](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-15](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-16](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-23](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-24](https://github.com/ansarzeinulla/ticket-project/commits/week-03).
  - **Next responsibility:** BF-37, BF-38, BF-39, BF-50 in week 4.

- **Team member:** Alibi Takhtanov (S2)
  - **Contribution this period:** completed BF-17 Document the authentication endpoints, BF-25 Derive a slug from the title, BF-26 Manage ticket types, BF-27 Add the checkout transaction.
  - **Evidence:** [BF-17](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-25](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-26](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-27](https://github.com/ansarzeinulla/ticket-project/commits/week-03).
  - **Next responsibility:** BF-40, BF-41, BF-42, BF-51, BF-52 in week 4.

- **Team member:** Abylay Otaubay (S3)
  - **Contribution this period:** completed BF-18 Add the API client, BF-19 Add reset and verify pages, BF-28 Add the formatting dependencies, BF-29 Add the public catalogue, BF-30 Add the order page.
  - **Evidence:** [BF-18](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-19](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-28](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-29](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-30](https://github.com/ansarzeinulla/ticket-project/commits/week-03).
  - **Next responsibility:** BF-43, BF-44, BF-45, BF-53 in week 4.

- **Team member:** Alinur Burlybayev (S4)
  - **Contribution this period:** completed BF-20 Keep the session in the browser, BF-31 Move the create form to the client, BF-32 Add the organizer dashboard, BF-33 List orders and attendees.
  - **Evidence:** [BF-20](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-31](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-32](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-33](https://github.com/ansarzeinulla/ticket-project/commits/week-03).
  - **Next responsibility:** BF-46, BF-47, BF-54 in week 4.

- **Team member:** Olzhas Nurseit (S5)
  - **Contribution this period:** completed BF-21 Add the Phase 1 specification, BF-22 Scaffold the Expo app and sign in on the device, BF-34 Add the Phase 2 specification, BF-35 Add the Phase 3 specification, BF-36 List assigned events on the device.
  - **Evidence:** [BF-21](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-22](https://github.com/ansarzeinulla/ticket-project/commits/week-02), [BF-34](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-35](https://github.com/ansarzeinulla/ticket-project/commits/week-03), [BF-36](https://github.com/ansarzeinulla/ticket-project/commits/week-03).
  - **Next responsibility:** BF-48, BF-49, BF-55, BF-56, BF-57 in week 4.

## Risks, blockers, and decisions needed

- **Risk, blocker, or decision:** Concurrent checkout is the one place where a bug costs real money
  - **Impact:** Overselling a sold-out event would be visible to buyers.
  - **Next action or support needed:** A dedicated concurrency test runs in CI on every push.
  - **Owner:** Alibi Takhtanov (S2)

## Changes, reflection, and support

- **Scope or schedule changes:** None.
- **Team reflection:** Building the client and the API in parallel cost us a day. Fixing the merge order inside a week (schema, then handlers, then routes, then types, then UI) solved it.
- **Instructor/TA help requested:** None at this time.
