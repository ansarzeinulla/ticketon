# BiletFlow — delivery plan (CSCI 361, Fall 2026)

Everything the team needs to run the project for the term: who owns what, what
lands in GitHub and Jira each week, and the seven biweekly reports.

## Files

| File | What it is |
| --- | --- |
| `roles.md` | The five roles, and which directories each person owns |
| `timeline.md` | Week-by-week schedule: commits, files, Jira transitions |
| `jira-backlog.md` | Epics and issues with keys, mapped to sprints |
| `reports/report-NN-*.md` | The seven reports, already drafted |

## Before you use this

Search-and-replace these placeholders once, across the whole folder:

| Placeholder | Replace with |
| --- | --- |
| `Student A` … `Student E` | Real names |
| `<STUDENT-ID-A>` … `<STUDENT-ID-E>` | Real student IDs |
| `<EMAIL-A>` … `<EMAIL-E>` | Real emails |
| `<TEAM-NAME>` | Your team name |
| `<REPO-URL>` | `https://github.com/<org>/<repo>` |
| `<JIRA-URL>` | Your Jira board URL |

```bash
# example
grep -rl "Student A" plan/ | xargs sed -i '' 's/Student A/Aigerim Zhaksy/g'
```

## Deadlines

Seven reports. Six fall on biweekly Sundays; the last one is the Saturday the
term ends, which is why it covers one week rather than two.

| # | Due | Covers | Sprint |
| --- | --- | --- | --- |
| 1 | Sun 13 Sep 2026 | Mon 31 Aug – Sun 13 Sep | Sprint 1 |
| 2 | Sun 27 Sep 2026 | Mon 14 Sep – Sun 27 Sep | Sprint 2 |
| 3 | Sun 11 Oct 2026 | Mon 28 Sep – Sun 11 Oct | Sprint 3 |
| 4 | Sun 25 Oct 2026 | Mon 12 Oct – Sun 25 Oct | Sprint 4 |
| 5 | Sun 08 Nov 2026 | Mon 26 Oct – Sun 08 Nov | Sprint 5 |
| 6 | Sun 22 Nov 2026 | Mon 09 Nov – Sun 22 Nov | Sprint 6 |
| 7 | Sat 28 Nov 2026 | Mon 23 Nov – Sat 28 Nov | Sprint 7 (final) |

> A strict biweekly cadence would put report 7 on Sun 6 Dec, past the 28 Nov end
> of term. It is scheduled for Sat 28 Nov instead and marked as the final
> report. If your instructor wants six reports only, drop report 7 and fold its
> content into report 6.

## The cadence rule that matters

The reports are biweekly. **GitHub and Jira are not.** Both are updated at least
weekly, and in practice several times a week, so the history shows steady work
rather than two big drops per fortnight.

Per person, per week, the minimum:

- **3 or more commits**, on at least **2 different days**
- **1 pull request** opened and merged
- **Jira issues moved** the same day the work happens — not in a batch on Sunday

Per team, per week:

- Monday: sprint check-in, issues pulled to *In Progress*
- Wednesday: mid-week sync, PRs in review
- Friday: PRs merged, issues closed, `main` is green
- Sunday (report weeks only): report submitted

## Commit and branch conventions

Branch per issue:

```
feature/BF-42-checkout-inventory-lock
fix/BF-77-refund-clears-checked-in-at
docs/BF-90-phase-3-test-spec
```

Commit subject carries the issue key so Jira links the work automatically:

```
BF-42 Lock ticket_types rows during checkout

Inventory is decremented inside the same transaction that issues the
tickets, so two buyers racing for the last seat cannot both win.
```

Rules that keep the history readable:

- One logical change per commit; never mix a feature with a reformat
- `main` always builds: `make api-check`, `make web-check` before merging
- Every PR needs one peer review — pair up across roles, not within them
