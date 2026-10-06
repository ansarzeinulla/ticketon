# Biweekly Team Progress Report

CSCI 361 - Fall 2026

## Report details

- **Team name:** AAOAA
- **Reporting period:** October 9 - October 22, 2026
- **Report date:** October 22, 2026
- **Submitted by:** Ansar Zeinulla
- **Current stage:** Build (Week 4 and 5)
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

The team closed week 4 and 5. In week 4, money and tickets: activation gates paid sales, stock holds under concurrent checkout, and a paid order produces a QR ticket and a printable PDF; In week 5, the gate works: the scanner admits a ticket once and refuses the second attempt. 21 issues were closed, each by exactly one commit and one reviewed pull request. The team is on track.

## Progress this period

### Previous commitments

- **Previous commitment:** Close Week 2 (Identity and the mobile shell)
  - **Status:** Completed
  - **Result or reason:** Accounts exist: people register, sign in and reset a password, and the Expo app runs on a device and signs in against the same API.
  - **Evidence:** [branch `week-02`](https://github.com/ansarzeinulla/ticket-project/tree/week-02), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-02)

- **Previous commitment:** Close Week 3 (Events, catalogue and free registration)
  - **Status:** Completed
  - **Result or reason:** The first end-to-end path: an organizer creates an event and a visitor registers for it from the public catalogue.
  - **Evidence:** [branch `week-03`](https://github.com/ansarzeinulla/ticket-project/tree/week-03), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-03)

### Other progress

- **Outcome or deliverable:** Closed Week 4 (Paid sales and digital tickets), 5–11 октября
  - **Status:** Done
  - **Evidence or result:** 13 issues closed. [Commits on `week-04`](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-04).
  - **Details:** `S1` closed [`BF-37`](<JIRA-BOARD-URL>) Add organizer profiles and money handling; `S1` closed [`BF-38`](<JIRA-BOARD-URL>) Enforce role permissions; `S2` closed [`BF-40`](<JIRA-BOARD-URL>) Add seat inventory; `S2` closed [`BF-41`](<JIRA-BOARD-URL>) Gate paid sales behind activation; and 9 more issues in the same week.

- **Outcome or deliverable:** Closed Week 5 (Check-in and the gate), 12–18 октября
  - **Status:** Done
  - **Evidence or result:** 8 issues closed. [Commits on `week-05`](https://github.com/ansarzeinulla/ticket-project/commits/week-05), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-05).
  - **Details:** `S1` closed [`BF-50`](<JIRA-BOARD-URL>) Update the money assertions for the fee; `S2` closed [`BF-51`](<JIRA-BOARD-URL>) Reserve stock with 15-minute holds; `S2` closed [`BF-52`](<JIRA-BOARD-URL>) Charge a processing fee; `S3` closed [`BF-53`](<JIRA-BOARD-URL>) Show the hold countdown at checkout; and 4 more issues in the same week.

## Commitments for the next two weeks

- **Commitment or milestone:** Close Week 6 (Refunds, support and administration)
  - **Owner(s):** All five team members (13 issues, 1-3 each)
  - **Due date:** 25 октября, 2026
  - **Success criteria:** what happens when things go wrong: refunds, cancellations, the support desk and the admin portal; all issues merged and CI green.

- **Commitment or milestone:** Mobile - the gate refuses a refunded ticket
  - **Owner(s):** Olzhas Nurseit (S5)
  - **Due date:** 25 октября, 2026
  - **Success criteria:** scanning a refunded ticket shows the refusal reason.

- **Commitment or milestone:** Close Week 7 (Analytics, campaigns and localisation)
  - **Owner(s):** All five team members (10 issues, 1-3 each)
  - **Due date:** 1 ноября, 2026
  - **Success criteria:** organizers see numbers and run campaigns, and the site speaks Kazakh and Russian; all issues merged and CI green.

## Team contributions and coordination

*Note: zones do not overlap. Every file in the repository has exactly one owner ([`plan/ownership.map`](https://github.com/ansarzeinulla/ticket-project/blob/main/plan/ownership.map)), so two people never edit the same file. When a change needs a file owned by someone else, the requester raises it with the owner instead of editing it. All pull requests are reviewed and merged by S1 (Captain).*

- **Team member:** Ansar Zeinulla (S1)
  - **Contribution this period:** completed BF-37 Add organizer profiles and money handling, BF-38 Enforce role permissions, BF-39 Send the order confirmation, BF-50 Update the money assertions for the fee.
  - **Evidence:** [BF-37](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-38](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-39](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-50](https://github.com/ansarzeinulla/ticket-project/commits/week-05).
  - **Next responsibility:** BF-58, BF-59, BF-60, BF-68 in week 6.

- **Team member:** Alibi Takhtanov (S2)
  - **Contribution this period:** completed BF-40 Add seat inventory, BF-41 Gate paid sales behind activation, BF-42 Embed a Unicode font for Cyrillic, BF-51 Reserve stock with 15-minute holds, BF-52 Charge a processing fee.
  - **Evidence:** [BF-40](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-41](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-42](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-51](https://github.com/ansarzeinulla/ticket-project/commits/week-05), [BF-52](https://github.com/ansarzeinulla/ticket-project/commits/week-05).
  - **Next responsibility:** BF-61, BF-62, BF-69, BF-70 in week 6.

- **Team member:** Abylay Otaubay (S3)
  - **Contribution this period:** completed BF-43 Show the activation notice, BF-44 Add the seat map and seated checkout, BF-45 Link the printable PDF, BF-53 Show the hold countdown at checkout.
  - **Evidence:** [BF-43](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-44](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-45](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-53](https://github.com/ansarzeinulla/ticket-project/commits/week-05).
  - **Next responsibility:** BF-63, BF-64, BF-71, BF-72, BF-73 in week 6.

- **Team member:** Alinur Burlybayev (S4)
  - **Contribution this period:** completed BF-46 Add the activation checklist, BF-47 Show ticket status per order, BF-54 Add the staff manager.
  - **Evidence:** [BF-46](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-47](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-54](https://github.com/ansarzeinulla/ticket-project/commits/week-05).
  - **Next responsibility:** BF-65, BF-B2, BF-B3, BF-74, BF-75, BF-B4 in week 6.

- **Team member:** Olzhas Nurseit (S5)
  - **Contribution this period:** completed BF-48 Prove checkout cannot oversell, BF-49 Add the Phase 4 specification, BF-55 Prepare the gate backend and a fake scan payload, BF-56 Scan and admit a ticket, BF-57 Document how to run the scanner.
  - **Evidence:** [BF-48](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-49](https://github.com/ansarzeinulla/ticket-project/commits/week-04), [BF-55](https://github.com/ansarzeinulla/ticket-project/commits/week-05), [BF-56](https://github.com/ansarzeinulla/ticket-project/commits/week-05), [BF-57](https://github.com/ansarzeinulla/ticket-project/commits/week-05).
  - **Next responsibility:** BF-66, BF-B1, BF-67, BF-76 in week 6.

## Risks, blockers, and decisions needed

- **Risk, blocker, or decision:** The scanner is tested on one Android device only
  - **Impact:** Camera permissions behave differently across devices.
  - **Next action or support needed:** Device checks are recorded in `5.md` with screenshots attached as evidence.
  - **Owner:** Olzhas Nurseit (S5)

## Changes, reflection, and support

- **Scope or schedule changes:** None.
- **Team reflection:** Writing a file small and wrong on purpose, then rewriting it later, felt wasteful at first. It is how the history shows the project actually grew.
- **Instructor/TA help requested:** None at this time.
