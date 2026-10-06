# Biweekly Team Progress Report

CSCI 361 - Fall 2026

## Report details

- **Team name:** AAOAA
- **Reporting period:** September 14 - September 24, 2026
- **Report date:** September 24, 2026
- **Submitted by:** Ansar Zeinulla
- **Current stage:** Build (Week 1)
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

The team closed week 1. In week 1, an empty but running system: the database, API service, web app, CI pipeline and container images all start with one command. 8 issues were closed, each by exactly one commit and one reviewed pull request. The team is on track.

## Progress this period

### Previous commitments

- **Previous commitment:** Close Week 1 (Running skeleton)
  - **Status:** Completed
  - **Result or reason:** An empty but running system: the database, API service, web app, CI pipeline and container images all start with one command.
  - **Evidence:** [branch `week-01`](https://github.com/ansarzeinulla/ticket-project/tree/week-01), [merged pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-01)

### Other progress

- **Outcome or deliverable:** Closed Week 1 (Running skeleton), 14–20 сентября
  - **Status:** Done
  - **Evidence or result:** 8 issues closed. [Commits on `week-01`](https://github.com/ansarzeinulla/ticket-project/commits/week-01), [pull requests](https://github.com/ansarzeinulla/ticket-project/pulls?q=is%3Apr+base%3Aweek-01).
  - **Details:** `S1` closed [`BF-6`](<JIRA-BOARD-URL>) Add extensions and the users table; `S1` closed [`BF-7`](<JIRA-BOARD-URL>) Add config and the error envelope; `S1` closed [`BF-8`](<JIRA-BOARD-URL>) Add the pool and a health route; `S2` closed [`BF-9`](<JIRA-BOARD-URL>) Document the API layout; and 4 more issues in the same week.

## Commitments for the next two weeks

- **Commitment or milestone:** Close Week 2 (Identity and the mobile shell)
  - **Owner(s):** All five team members (9 issues, 1-3 each)
  - **Due date:** 27 сентября, 2026
  - **Success criteria:** accounts exist: people register, sign in and reset a password, and the Expo app runs on a device and signs in against the same API; all issues merged and CI green.

- **Commitment or milestone:** Mobile - the Expo app builds and signs in against the API
  - **Owner(s):** Olzhas Nurseit (S5)
  - **Due date:** 27 сентября, 2026
  - **Success criteria:** `npx expo start` runs on a device and a staff account can log in.

- **Commitment or milestone:** Close Week 3 (Events, catalogue and free registration)
  - **Owner(s):** All five team members (14 issues, 1-3 each)
  - **Due date:** 4 октября, 2026
  - **Success criteria:** the first end-to-end path: an organizer creates an event and a visitor registers for it from the public catalogue; all issues merged and CI green.

- **Commitment or milestone:** Mobile - the device lists the events assigned to the signed-in staff member
  - **Owner(s):** Olzhas Nurseit (S5)
  - **Due date:** 4 октября, 2026
  - **Success criteria:** the events screen loads real rows from the API.

## Team contributions and coordination

*Note: zones do not overlap. Every file in the repository has exactly one owner ([`plan/ownership.map`](https://github.com/ansarzeinulla/ticket-project/blob/main/plan/ownership.map)), so two people never edit the same file. When a change needs a file owned by someone else, the requester raises it with the owner instead of editing it. All pull requests are reviewed and merged by S1 (Captain).*

- **Team member:** Ansar Zeinulla (S1)
  - **Contribution this period:** completed BF-6 Add extensions and the users table, BF-7 Add config and the error envelope, BF-8 Add the pool and a health route.
  - **Evidence:** [BF-6](https://github.com/ansarzeinulla/ticket-project/commits/week-01), [BF-7](https://github.com/ansarzeinulla/ticket-project/commits/week-01), [BF-8](https://github.com/ansarzeinulla/ticket-project/commits/week-01).
  - **Next responsibility:** BF-14, BF-15, BF-16, BF-23, BF-24 in week 2.

- **Team member:** Alibi Takhtanov (S2)
  - **Contribution this period:** completed BF-9 Document the API layout.
  - **Evidence:** [BF-9](https://github.com/ansarzeinulla/ticket-project/commits/week-01).
  - **Next responsibility:** BF-17, BF-25, BF-26, BF-27 in week 2.

- **Team member:** Abylay Otaubay (S3)
  - **Contribution this period:** completed BF-10 Scaffold Next.js.
  - **Evidence:** [BF-10](https://github.com/ansarzeinulla/ticket-project/commits/week-01).
  - **Next responsibility:** BF-18, BF-19, BF-28, BF-29, BF-30 in week 2.

- **Team member:** Alinur Burlybayev (S4)
  - **Contribution this period:** completed BF-11 Add the signed-in shell.
  - **Evidence:** [BF-11](https://github.com/ansarzeinulla/ticket-project/commits/week-01).
  - **Next responsibility:** BF-20, BF-31, BF-32, BF-33 in week 2.

- **Team member:** Olzhas Nurseit (S5)
  - **Contribution this period:** completed BF-12 Add Postgres to compose, BF-13 Add CI.
  - **Evidence:** [BF-12](https://github.com/ansarzeinulla/ticket-project/commits/week-01), [BF-13](https://github.com/ansarzeinulla/ticket-project/commits/week-01).
  - **Next responsibility:** BF-21, BF-22, BF-34, BF-35, BF-36 in week 2.

## Risks, blockers, and decisions needed

- **Risk, blocker, or decision:** Scope of the initial requirements against the time available
  - **Impact:** the team could run out of time before paid checkout and ticket generation work end to end.
  - **Next action or support needed:** the estimate behind this assessment: the requirements decompose into 96 issues across about 320 files, which is roughly two issues per person per week for ten weeks with no slack. Four items are marked bonus and are cut first if we fall behind - offline sync, `.ics` export, the interactive seat map and GA4 - because none of them is needed for a visitor to buy a ticket and get through the gate. The core checkout and ticket generation work is protected and scheduled in weeks 4 and 5, deliberately early.
  - **Owner:** Ansar Zeinulla (S1)

- **Risk, blocker, or decision:** Alinur (S4) has no commits yet
  - **Impact:** one member's contribution is not verifiable from the repository.
  - **Next action or support needed:** `BF-11` is assigned to S4 this week and a GitHub account is being created.
  - **Owner:** Ansar Zeinulla (S1)

## Changes, reflection, and support

- **Scope or schedule changes:** None.
- **Team reflection:** Opening every issue before the week starts felt bureaucratic on day one and paid for itself by day three: nobody had to ask what to work on.
- **Instructor/TA help requested:** None at this time.

## Response to feedback on Report 1

- **Jira access:** `oadiyatov@nu.edu.kz` invited; the board link is in Project links above.
- **Calendar deadlines:** the plan now runs in calendar weeks (Monday to Sunday) instead of phases, so every commitment above carries a real date.
- **Mobile development:** the Expo app moved forward to week 2 and now has a named deliverable in most weeks, owned by S5. See the Mobile commitment above.
- **Evidence links:** every entry above links to the branch, the pull requests or the file.
- **Scope assessment:** the estimate is written out in Risks above.
- **Ownership map:** linked in Project links. A change that needs somebody else's file goes through the owner. We treat "conflict-free" as a prediction to test rather than a fact: each week `git log` is checked for any file touched by two authors, and the count is reported here.
