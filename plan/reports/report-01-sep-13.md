# Biweekly Team Progress Report

CSCI 361 - Fall 2026

**Cover page**

## Report details

- Team name: <TEAM-NAME>
- Reporting period: August 31, 2026 - September 13, 2026
- Submitted by: Student A
- Current stage: Build (Discovery completed in week 1)
- Overall status: On track

### Project links

- Source code repository: <REPO-URL>
- Task management tool: <JIRA-URL>

## Team members

| Full name | Student ID | Email |
| --- | --- | --- |
| Student A | <STUDENT-ID-A> | <EMAIL-A> |
| Student B | <STUDENT-ID-B> | <EMAIL-B> |
| Student C | <STUDENT-ID-C> | <EMAIL-C> |
| Student D | <STUDENT-ID-D> | <EMAIL-D> |
| Student E | <STUDENT-ID-E> | <EMAIL-E> |

{{< pagebreak >}}

## Progress snapshot

We settled the scope against the BiletFlow SRS, split it into five ownership
lanes, and got a running skeleton on every machine: PostgreSQL 17 in Docker with
the full core schema, a Go API that boots, a Next.js app that renders, and CI
that builds and tests on every push. The database is the real deliverable this
period - the schema already carries the constraints that later correctness work
depends on, so we are not planning to retrofit integrity later. On track.

## Progress this period

### Previous commitments

Not applicable: first report.

### Other progress

- Outcome: Database schema and container
  - Status: Done
  - Evidence or result: `docker-compose.yml`, `db/init/01_extensions.sql`,
    `db/init/02_schema.sql`. 25 tables covering every entity in SRS §6, with
    enums, check constraints and foreign keys written up front rather than
    added later. `make up` brings it up from nothing. (BF-1, BF-2, BF-10)

- Outcome: Schema test suite
  - Status: Done
  - Evidence or result: `db/tests/` with a runner (`make test`). It asserts the
    expected tables, enums and constraints exist, so a careless migration fails
    loudly. (BF-11)

- Outcome: Go API skeleton and JSON contract
  - Status: Done
  - Evidence or result: `api/cmd/api/main.go`, `api/internal/config`,
    `api/internal/httpx`. A single error envelope - `{"error":{"code","message",
    "fields"}}` - agreed now so every later endpoint is consistent. (BF-3, BF-4)

- Outcome: Next.js application shell
  - Status: Done
  - Evidence or result: `web/` with design tokens, UI primitives and the
    signed-in shell. Static login and dashboard pages render. (BF-5, BF-6, BF-7)

- Outcome: Tooling and CI
  - Status: Done
  - Evidence or result: `Makefile` (`up`, `seed`, `test`, `api-run`, `web-dev`),
    `.github/workflows/ci.yml` running build, vet and tests on every push. (BF-8, BF-9)

- Outcome: Demo dataset and Phase 1 test specification
  - Status: Done
  - Evidence or result: `db/seed/01_demo_data.sql` (organizers, attendees, a
    venue layout, events) and `1.md`, the acceptance specification for accounts
    and authentication that Sprint 2 will be graded against. (BF-12, BF-14)

## Commitments for the next two weeks

- Commitment: Registration, sign-in and sessions working end to end
  - Owner(s): Student A (API), Student C (web)
  - Due date: September 20, 2026
  - Success criteria: A new account can register and sign in through the browser;
    the JWT is issued and accepted; `1.md` cases REG-01…REG-05 and LOG-01…LOG-07 pass.

- Commitment: Password reset and email verification
  - Owner(s): Student A
  - Due date: September 27, 2026
  - Success criteria: Reset and verification tokens are stored as SHA-256
    hashes, single-use, and expire; the mock email prints to the API console;
    `1.md` RST-* and VRF-* pass.

- Commitment: Event creation, editing and publication
  - Owner(s): Student B (API), Student D (dashboard)
  - Due date: September 27, 2026
  - Success criteria: An organizer creates an event, it is a draft, gets a
    unique slug, and can be published and unpublished from the dashboard.

- Commitment: Ticket types with free and paid tiers
  - Owner(s): Student B
  - Due date: September 27, 2026
  - Success criteria: Prices are stored as `numeric(14,2)` and travel as decimal
    strings; a tier can be hidden without deletion.

- Commitment: Phase 2 test specification
  - Owner(s): Student E
  - Due date: September 27, 2026
  - Success criteria: `2.md` covers event lifecycle and ticket management, and
    is used to sign off the sprint.

## Team contributions and coordination

- Team member: Student A
  - Contribution this period: Wrote the database schema, the seed dataset and
    the schema test suite; set up the Jira project, ten epics and the sprint
    board.
  - Evidence: <JIRA-URL>/browse/BF-2, <JIRA-URL>/browse/BF-11, <REPO-URL>/pull/2
  - Next responsibility: Authentication - bcrypt hashing, JWT issuing, and the
    register/login endpoints.

- Team member: Student B
  - Contribution this period: Scaffolded the Go service, configuration loading
    and the shared HTTP error envelope; added the pgx connection pool.
  - Evidence: <JIRA-URL>/browse/BF-3, <JIRA-URL>/browse/BF-13, <REPO-URL>/pull/3
  - Next responsibility: Events and ticket types - CRUD, slugs and lifecycle.

- Team member: Student C
  - Contribution this period: Scaffolded the Next.js app, established design
    tokens and the shared UI primitives, and built the static auth pages.
  - Evidence: <JIRA-URL>/browse/BF-5, <JIRA-URL>/browse/BF-6, <REPO-URL>/pull/4
  - Next responsibility: Wiring sign-in and registration to the API, then the
    public catalogue and event page.

- Team member: Student D
  - Contribution this period: Built the signed-in shell, header and navigation,
    and the empty-state dashboard.
  - Evidence: <JIRA-URL>/browse/BF-7, <REPO-URL>/pull/5
  - Next responsibility: The event create form and the organizer event dashboard.

- Team member: Student E
  - Contribution this period: Wrote the Makefile and CI workflow, and authored
    `1.md`, the Phase 1 acceptance specification.
  - Evidence: <JIRA-URL>/browse/BF-9, <JIRA-URL>/browse/BF-14, <REPO-URL>/pull/6
  - Next responsibility: Database CRUD and constraint tests, then the Phase 2
    specification.

## Risks, blockers, and decisions needed

- Risk: Scope. The SRS lists eighteen required MVP features plus five bonus
  ones, and thirteen weeks is not long.
  - Impact: Attempting the bonus features early would put required ones at risk.
  - Next action: Bonus items (seat map, `.ics` export, GA4, offline sync,
    support attachments) are explicitly deferred to Sprint 6 and only if the
    required backlog is clear. Recorded in `plan/timeline.md`.
  - Owner: Student A

- Decision needed: None outstanding.

## Changes, reflection, and support

- Scope or schedule changes: None. Sprint 1 finished as planned.
- Team reflection: Writing the whole schema in week 1 rather than growing it
  per feature already paid off - the constraints are forcing correct thinking
  about refunds and inventory before that code exists. Repeat. What to change:
  two people worked Thursday-Friday only in week 1; the weekly minimum of three
  commits across two days is now explicit in `plan/README.md`.
- Instructor/TA help requested: None.
