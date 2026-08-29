# Baseline measurement — 2026-08-29

Taken before any gap-closure work, against a healthy `biletflow-db` container.

| Suite | Command | Result | Real number |
| --- | --- | --- | --- |
| Database | `make test` | PASS | 6/6 files, **145** assertions |
| Go fmt + vet + tests | `make api-check` | PASS | all packages `ok` |
| API acceptance | `make api-smoke` | PASS | **137** PASS, 0 FAIL |
| Web lint | `make web-lint` | PASS | no output |
| Web typecheck | `npx tsc --noEmit` | PASS | no output |
| Mobile typecheck | `make scan-check` | PASS | no output |

## Corrections to README.md claims

| README says | Actually |
| --- | --- |
| "144 database assertions" | 145 |
| "276 Go unit + integration tests" | 214 top-level `func Test*` (plus 28 `t.Run` subtests) |
| "68 cURL acceptance checks" | 137 `PASS` lines |

Everything that exists passes. The work that follows is about requirements that
have no implementation at all, not about repairing broken ones.
