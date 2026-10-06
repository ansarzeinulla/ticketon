# Biweekly Team Progress Report

CSCI 361 - Fall 2026

## Report details

- **Team name:** AAOAA
- **Reporting period:** October 23 - November 5, 2026
- **Report date:** November 5, 2026
- **Submitted by:** Ansar Zeinulla
- **Current stage:** Build (Week 6 and 7)
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

The team closed week 6 and 7. In week 6, what happens when things go wrong: refunds, cancellations, the support desk and the admin portal; In week 7, organizers see numbers and run campaigns, and the site speaks Kazakh and Russian. 23 issues were closed, each by exactly one commit and one reviewed pull request. The team is on track.

## Progress this period

### Previous commitments

- **Previous commitment:** Close Week 4 (Paid sales and digital tickets)
  - **Status:** Completed
  - **Result or reason:** Money and tickets: activation gates paid sales, stock holds under concurrent checkout, and a paid order produces a QR ticket and a printable PDF.
  - **Evidence:** [branch `week-04`](https://github.com/ansarzeinulla/ticket-project/tree/week-04), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-04)

- **Previous commitment:** Close Week 5 (Check-in and the gate)
  - **Status:** Completed
  - **Result or reason:** The gate works: the scanner admits a ticket once and refuses the second attempt.
  - **Evidence:** [branch `week-05`](https://github.com/ansarzeinulla/ticket-project/tree/week-05), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-05)

### Other progress

- **Outcome or deliverable:** Closed Week 6 (Refunds, support and administration), 19–25 октября
  - **Status:** Done
  - **Evidence or result:** 10 issues closed and 3 defects fixed. [Commits on `week-06`](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-06).
  - **Details:** `S1` closed [`BF-58`](<JIRA-BOARD-URL>) Add refund, support and report tables; `S1` closed [`BF-59`](<JIRA-BOARD-URL>) Add the report queue and settings; `S1` closed [`BF-60`](<JIRA-BOARD-URL>) Give every conflict its own error code; `S2` closed [`BF-61`](<JIRA-BOARD-URL>) Refund an order atomically; and 9 more issues in the same week.

- **Outcome or deliverable:** Closed Week 7 (Analytics, campaigns and localisation), 26 октября – 1 ноября
  - **Status:** Done
  - **Evidence or result:** 9 issues closed and 1 defects fixed. [Commits on `week-07`](https://github.com/ansarzeinulla/ticket-project/commits/week-07), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-07).
  - **Details:** `S1` closed [`BF-68`](<JIRA-BOARD-URL>) Add campaign tables; `S2` closed [`BF-69`](<JIRA-BOARD-URL>) Create campaigns and promo codes; `S2` closed [`BF-70`](<JIRA-BOARD-URL>) Apply and cap discounts atomically; `S3` closed [`BF-71`](<JIRA-BOARD-URL>) Add the English dictionary; and 6 more issues in the same week.

## Commitments for the next two weeks

- **Commitment or milestone:** Close Week 8 (Release shape)
  - **Owner(s):** All five team members (13 issues, 1-3 each)
  - **Due date:** 8 ноября, 2026
  - **Success criteria:** the product reaches the shape it ships in: character limits, payout figures, offline check-in sync and the remaining defects; all issues merged and CI green.

- **Commitment or milestone:** Mobile - check-ins collected without network sync when the device reconnects
  - **Owner(s):** Olzhas Nurseit (S5)
  - **Due date:** 8 ноября, 2026
  - **Success criteria:** scans queued offline appear in the attendee list after sync.

- **Commitment or milestone:** Close Week 9 (Verification)
  - **Owner(s):** All five team members (8 issues, 1-3 each)
  - **Due date:** 15 ноября, 2026
  - **Success criteria:** every phase specification is run against the running system and the failures found are fixed; all issues merged and CI green.

- **Commitment or milestone:** Mobile - the scanner is re-tested against the recorded runs
  - **Owner(s):** Olzhas Nurseit (S5)
  - **Due date:** 15 ноября, 2026
  - **Success criteria:** `5.md` passes on a physical Android device, screenshots attached.

## Team contributions and coordination

*Note: zones do not overlap. Every file in the repository has exactly one owner ([`plan/ownership.map`](https://github.com/ansarzeinulla/ticket-project/blob/main/plan/ownership.map)), so two people never edit the same file. When a change needs a file owned by someone else, the requester raises it with the owner instead of editing it. All pull requests are reviewed and merged by S1 (Captain).*

- **Team member:** Ansar Zeinulla (S1)
  - **Contribution this period:** completed BF-58 Add refund, support and report tables, BF-59 Add the report queue and settings, BF-60 Give every conflict its own error code, BF-68 Add campaign tables.
  - **Evidence:** [BF-58](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-59](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-60](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-68](https://github.com/ansarzeinulla/ticket-project/commits/week-07).
  - **Next responsibility:** BF-77, BF-B5, BF-B6, BF-84 in week 8.

- **Team member:** Alibi Takhtanov (S2)
  - **Contribution this period:** completed BF-61 Refund an order atomically, BF-62 Expose the rows the admin search needs, BF-69 Create campaigns and promo codes, BF-70 Apply and cap discounts atomically.
  - **Evidence:** [BF-61](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-62](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-69](https://github.com/ansarzeinulla/ticket-project/commits/week-07), [BF-70](https://github.com/ansarzeinulla/ticket-project/commits/week-07).
  - **Next responsibility:** BF-78, BF-B7, BF-B8, BF-85 in week 8.

- **Team member:** Abylay Otaubay (S3)
  - **Contribution this period:** completed BF-63 Add the support widget to the order page, BF-64 Point the client at the proxy, BF-71 Add the English dictionary, BF-72 Localise the attendee pages, BF-73 Add the language negotiation dependency.
  - **Evidence:** [BF-63](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-64](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-71](https://github.com/ansarzeinulla/ticket-project/commits/week-07), [BF-72](https://github.com/ansarzeinulla/ticket-project/commits/week-07), [BF-73](https://github.com/ansarzeinulla/ticket-project/commits/week-07).
  - **Next responsibility:** BF-79, BF-86, BF-87 in week 8.

- **Team member:** Alinur Burlybayev (S4)
  - **Contribution this period:** completed BF-65 Add the admin portal, BF-B2 Move the session into an httpOnly cookie, BF-B3 Rename middleware to proxy for Next 16, BF-74 Compute analytics from operational rows, BF-75 Localise the dashboard, BF-B4 Read the GA4 id from a server-safe module.
  - **Evidence:** [BF-65](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-B2](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-B3](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-74](https://github.com/ansarzeinulla/ticket-project/commits/week-07), [BF-75](https://github.com/ansarzeinulla/ticket-project/commits/week-07), [BF-B4](https://github.com/ansarzeinulla/ticket-project/commits/week-07).
  - **Next responsibility:** BF-80, BF-B9, BF-81, BF-88 in week 8.

- **Team member:** Olzhas Nurseit (S5)
  - **Contribution this period:** completed BF-66 Refuse a refunded ticket at the gate, BF-B1 Keep the check-in record when a ticket is voided, BF-67 Add the Phase 6 specification, BF-76 Add the Phase 7 specification.
  - **Evidence:** [BF-66](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-B1](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-67](https://github.com/ansarzeinulla/ticket-project/commits/week-06), [BF-76](https://github.com/ansarzeinulla/ticket-project/commits/week-07).
  - **Next responsibility:** BF-B10, BF-82, BF-83, BF-89, BF-90, BF-91 in week 8.

## Risks, blockers, and decisions needed

- **Risk, blocker, or decision:** Three locales multiply the number of screens to check
  - **Impact:** A missing key shows an English string to a Kazakh visitor.
  - **Next action or support needed:** `make i18n-check` fails the build on a missing key.
  - **Owner:** Abylay Otaubay (S3)

## Changes, reflection, and support

- **Scope or schedule changes:** None.
- **Team reflection:** Defects are filed the day they are found instead of at the end of the week. The board is uglier and much more honest.
- **Instructor/TA help requested:** None at this time.
