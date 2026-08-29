#!/usr/bin/env bash
#
# BiletFlow Phase 1 database test suite.
#
#   ./db/tests/run_tests.sh            run every test file
#   ./db/tests/run_tests.sh 04         run only files whose name starts with 04
#
# Exits non-zero if any assertion fails, so it can be wired into CI later.

set -uo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT" || exit 1

# shellcheck disable=SC1091
[ -f .env ] && set -a && . ./.env && set +a

DB_USER="${POSTGRES_USER:-biletflow}"
DB_NAME="${POSTGRES_DB:-biletflow}"
SERVICE="db"
CONTAINER_TESTS_DIR="/opt/biletflow/tests"
FILTER="${1:-}"

if [ -t 1 ]; then
    RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; BOLD=$'\033[1m'; OFF=$'\033[0m'
else
    RED=""; GREEN=""; YELLOW=""; BOLD=""; OFF=""
fi

fail() { printf '%s\n' "${RED}${BOLD}FAILED${OFF} $*"; }
pass() { printf '%s\n' "${GREEN}PASSED${OFF} $*"; }

psql_run() {
    docker compose exec -T "$SERVICE" \
        psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -X -q "$@"
}

# -----------------------------------------------------------------------------
# Criterion 1: the container is up and the database accepts connections.
# -----------------------------------------------------------------------------
printf '%s\n' "${BOLD}BiletFlow database tests${OFF}"
printf '%s\n' "-----------------------------------------------------------"

if ! docker compose ps --status running --services 2>/dev/null | grep -qx "$SERVICE"; then
    fail "the '$SERVICE' service is not running. Start it with: docker compose up -d"
    exit 1
fi

health="$(docker inspect -f '{{.State.Health.Status}}' biletflow-db 2>/dev/null || echo unknown)"
if [ "$health" != "healthy" ]; then
    printf '%s\n' "${YELLOW}waiting for the database to become healthy...${OFF}"
    for _ in $(seq 1 30); do
        health="$(docker inspect -f '{{.State.Health.Status}}' biletflow-db 2>/dev/null || echo unknown)"
        [ "$health" = "healthy" ] && break
        sleep 2
    done
fi

if [ "$health" != "healthy" ]; then
    fail "container health check reports '$health'"
    exit 1
fi
pass "00_container       - PostgreSQL container is running and healthy"

# -----------------------------------------------------------------------------
# SQL test files
# -----------------------------------------------------------------------------
total_files=0
failed_files=0
total_asserts=0
failed_names=()

for file in "$PROJECT_ROOT"/db/tests/[0-9][0-9]_*.sql; do
    name="$(basename "$file" .sql)"
    [ -n "$FILTER" ] && case "$name" in "$FILTER"*) ;; *) continue ;; esac

    total_files=$((total_files + 1))
    output="$(psql_run -f "$CONTAINER_TESTS_DIR/$(basename "$file")" 2>&1)"
    status=$?
    asserts="$(printf '%s' "$output" | grep -c 'NOTICE:.*ok - ')"
    total_asserts=$((total_asserts + asserts))

    # A file that ran without error but produced no assertions is a silent
    # failure (e.g. the DO block never executed), so it counts as failed.
    if [ $status -eq 0 ] && [ "$asserts" -gt 0 ] && ! printf '%s' "$output" | grep -q 'FAIL:'; then
        pass "$name - $asserts assertions"
        [ "${VERBOSE:-0}" = "1" ] && printf '%s\n' "$output"
    else
        failed_files=$((failed_files + 1))
        failed_names+=("$name")
        fail "$name"
        printf '%s\n' "$output" | sed 's/^/    /'
    fi
done

printf '%s\n' "-----------------------------------------------------------"
if [ $failed_files -eq 0 ]; then
    printf '%s\n' "${GREEN}${BOLD}All ${total_files} test files passed (${total_asserts} assertions).${OFF}"
    exit 0
fi

printf '%s\n' "${RED}${BOLD}${failed_files}/${total_files} test files failed:${OFF} ${failed_names[*]}"
exit 1
