# Biweekly Team Progress Report

CSCI 361 - Fall 2026

## Report details

- **Team name:** AAOAA
- **Reporting period:** November 6 - November 19, 2026
- **Report date:** November 19, 2026
- **Submitted by:** Ansar Zeinulla
- **Current stage:** Test (Week 8 and 9)
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

The team closed week 8 and 9. In week 8, the product reaches the shape it ships in: character limits, payout figures, offline check-in sync and the remaining defects; In week 9, every phase specification is run against the running system and the failures found are fixed. 21 issues were closed, each by exactly one commit and one reviewed pull request. The team is on track.

## Progress this period

### Previous commitments

- **Previous commitment:** Close Week 6 (Refunds, support and administration)
  - **Status:** Completed
  - **Result or reason:** What happens when things go wrong: refunds, cancellations, the support desk and the admin portal.
  - **Evidence:** [branch `week-06`](https://github.com/ansarzeinulla/ticket-project/tree/week-06), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-06)

- **Previous commitment:** Close Week 7 (Analytics, campaigns and localisation)
  - **Status:** Completed
  - **Result or reason:** Organizers see numbers and run campaigns, and the site speaks Kazakh and Russian.
  - **Evidence:** [branch `week-07`](https://github.com/ansarzeinulla/ticket-project/tree/week-07), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-07)

### Other progress

- **Outcome or deliverable:** Closed Week 8 (Release shape), 2–8 ноября
  - **Status:** Done
  - **Evidence or result:** 7 issues closed and 6 defects fixed. [Commits on `week-08`](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-08).
  - **Details:** `S1` closed [`BF-77`](<JIRA-BOARD-URL>) Count text limits in the remaining handlers; `S1` closed [`BF-B5`](<JIRA-BOARD-URL>) Stop sharing notification state on the server; `S1` closed [`BF-B6`](<JIRA-BOARD-URL>) Make the seed re-runnable after a purchase; `S2` closed [`BF-78`](<JIRA-BOARD-URL>) Cut Cyrillic text by characters on the ticket; and 9 more issues in the same week.

- **Outcome or deliverable:** Closed Week 9 (Verification), 9–15 ноября
  - **Status:** Done
  - **Evidence or result:** 8 issues closed. [Commits on `week-09`](https://github.com/ansarzeinulla/ticket-project/commits/week-09), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-09).
  - **Details:** `S1` closed [`BF-84`](<JIRA-BOARD-URL>) Say "required" instead of "too short"; `S2` closed [`BF-85`](<JIRA-BOARD-URL>) Refuse to publish an event that has ended; `S3` closed [`BF-86`](<JIRA-BOARD-URL>) Close the localisation gaps; `S3` closed [`BF-87`](<JIRA-BOARD-URL>) Add the test runner dependencies; and 4 more issues in the same week.

## Commitments for the next two weeks

- **Commitment or milestone:** Close Week 10 (Documentation and release)
  - **Owner(s):** All five team members (5 issues, 1-3 each)
  - **Due date:** 22 ноября, 2026
  - **Success criteria:** documentation, a demo rehearsal and the release tag; all issues merged and CI green.

## Team contributions and coordination

*Note: zones do not overlap. Every file in the repository has exactly one owner ([`plan/ownership.map`](https://github.com/ansarzeinulla/ticket-project/blob/main/plan/ownership.map)), so two people never edit the same file. When a change needs a file owned by someone else, the requester raises it with the owner instead of editing it. All pull requests are reviewed and merged by S1 (Captain).*

- **Team member:** Ansar Zeinulla (S1)
  - **Contribution this period:** completed BF-77 Count text limits in the remaining handlers, BF-B5 Stop sharing notification state on the server, BF-B6 Make the seed re-runnable after a purchase, BF-84 Say "required" instead of "too short".
  - **Evidence:** [BF-77](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-B5](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-B6](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-84](https://github.com/ansarzeinulla/ticket-project/commits/week-09).
  - **Next responsibility:** BF-92 in week 10.

- **Team member:** Alibi Takhtanov (S2)
  - **Contribution this period:** completed BF-78 Cut Cyrillic text by characters on the ticket, BF-B7 Give a suspended event its own code, BF-B8 Name the refund conflict, BF-85 Refuse to publish an event that has ended.
  - **Evidence:** [BF-78](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-B7](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-B8](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-85](https://github.com/ansarzeinulla/ticket-project/commits/week-09).
  - **Next responsibility:** BF-93 in week 10.

- **Team member:** Abylay Otaubay (S3)
  - **Contribution this period:** completed BF-79 Localise the remaining attendee screens, BF-86 Close the localisation gaps, BF-87 Add the test runner dependencies.
  - **Evidence:** [BF-79](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-86](https://github.com/ansarzeinulla/ticket-project/commits/week-09), [BF-87](https://github.com/ansarzeinulla/ticket-project/commits/week-09).
  - **Next responsibility:** BF-94 in week 10.

- **Team member:** Alinur Burlybayev (S4)
  - **Contribution this period:** completed BF-80 Show reserved stock, BF-B9 Name the reversal conflict, BF-81 Localise the remaining dashboard screens, BF-88 Make the money card answer the tier filter.
  - **Evidence:** [BF-80](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-B9](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-81](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-88](https://github.com/ansarzeinulla/ticket-project/commits/week-09).
  - **Next responsibility:** the demo.

- **Team member:** Olzhas Nurseit (S5)
  - **Contribution this period:** completed BF-B10 Name the check-in reversal conflict, BF-82 Add the Phase 8, 9 and 10 specifications, BF-83 Point the scanner at the queue, BF-89 Run the phase specifications and record results, BF-90 Correct the specifications after the run, BF-91 Add the Postman collection.
  - **Evidence:** [BF-B10](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-82](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-83](https://github.com/ansarzeinulla/ticket-project/commits/week-08), [BF-89](https://github.com/ansarzeinulla/ticket-project/commits/week-09), [BF-90](https://github.com/ansarzeinulla/ticket-project/commits/week-09), [BF-91](https://github.com/ansarzeinulla/ticket-project/commits/week-09).
  - **Next responsibility:** BF-95, BF-96 in week 10.

## Risks, blockers, and decisions needed

- **Risk, blocker, or decision:** The verification week depends on every earlier week being green
  - **Impact:** A defect found late leaves little time before the demo.
  - **Next action or support needed:** Feature freeze on November 18; after that only fixes.
  - **Owner:** Ansar Zeinulla (S1)

## Changes, reflection, and support

- **Scope or schedule changes:** None.
- **Team reflection:** Localisation touched almost every screen at once. One sweep per owner was right; splitting it per screen would have produced near-identical commits.
- **Instructor/TA help requested:** None at this time.
