# Final verification results

Measured on 2026-08-29, after all gap-closure work. Every number here was
observed in a run, not carried over from documentation.

## The suites

| Suite | Command | Result |
| --- | --- | --- |
| Database schema and business rules | `make test` | **6/6 files, 147 assertions** |
| Go — gofmt, vet, unit + integration | `make api-check` | **all packages `ok`**, no output from gofmt or vet |
| API acceptance (cURL, over real HTTP) | `make api-smoke` | **168 checks, 0 failures** |
| Frontend unit (Vitest) | `make web-test` | **32 tests, 3 files** |
| Frontend lint + typecheck | `make web-check` | clean |
| Browser end-to-end (Playwright) | `make web-e2e` | **31 specs** |
| Scanner app typecheck | `make scan-check` | clean |

`make verify` runs everything that needs neither a browser nor a running
server, in one command.

## Run both ways

The end-to-end suite was run against two different deployments, because they
exercise different code paths:

| Deployment | Result |
| --- | --- |
| Host (`make api-run` + `make web-dev`) | 31/31, twice consecutively |
| Containers (`docker compose --profile app up`) | 31/31 |

Running it against containers is what caught the one defect the host path
could not: Server Components render *inside* the web container, where
`localhost:8080` is the web container rather than the API. Every client-side
call kept working while the public event page and catalogue failed. Fixed with
a separate, non-public `API_INTERNAL_BASE_URL`.

The suite was also run three times consecutively on the host to confirm it is
repeatable rather than order-dependent — 31/31 each time.

One caveat worth stating: on a **cold** `next dev` server the first run can
fail a spec on compile latency rather than on behaviour. It passes on a warm
server and in the production container build. This is a dev-mode artifact, not
a product defect.

## Verified by hand

Two things no headless runner reaches. Screenshots in `evidence/`.

| What | Evidence |
| --- | --- |
| The scanner refuses a second scan of the same ticket, naming the attendee and the first-use time | `evidence/scanner-already-used.png` |
| The gate refuses a Campaign QR with an actionable message (§11 criterion 8) | `evidence/scanner-campaign-qr-refused.png` |
| Undo check-in returns the ticket to `valid` while **retaining** the reversed `check_in_records` row (SRS 4.8) | `evidence/scanner-attendee-search-undo.png`, confirmed in PostgreSQL |
| The event selector shows only events the account is assigned to | `evidence/scanner-event-selector.png` |

The iOS Simulator has no camera, so admission was driven through the app's own
manual-entry sheet — which runs the *same* transaction as a camera scan, so it
is neither a weaker path nor a way around duplicate protection.

## Corrections to previously documented numbers

The README's counts were out of date before this work started:

| README said | Actually was | Now |
| --- | --- | --- |
| 144 database assertions | 145 | 147 |
| 276 Go tests | 214 top-level `func Test*` | 214 + 45 added |
| 68 cURL acceptance checks | 137 | 168 |
