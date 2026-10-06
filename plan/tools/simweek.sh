#!/usr/bin/env bash
# Прогнать скрипты недели в песочнице и проверить результат.
#
#   simweek.sh <NN> <песочница> <база недели> [эталонное дерево]
#
# Песочница — локальный клон ticket-project с фейковым origin (bare-репозиторий
# рядом). На GitHub ничего не уходит. Шаги:
#   1. ветка week-NN в песочнице = <база недели> (коммит или ветка песочницы);
#   2. скрипты S<N>-BF-xx.sh запускаются в порядке tickets.tsv (YES=1, REPO=песочница);
#   3. каждая ветка задачи проверяется отдельно — она должна собираться сама;
#   4. ветки вливаются в week-NN в том же порядке (--no-ff, как PR на GitHub);
#   5. полный набор проверок на итоге; сравнение с эталонным деревом, если дано.
#
# Нужны: локальный PostgreSQL на :5433 (роль biletflow), Go, Node;
# node_modules для web/ и mobile/ берутся из NODE_MODULES_FROM (по умолчанию
# из эталонного дерева) копией-клоном APFS.
set -uo pipefail

WEEK="$1" SB="$2" BASE="$3" EXPECT="${4:-}"
PLAN="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WDIR="$PLAN/week$WEEK"
WB="week-$WEEK"
NM_FROM="${NODE_MODULES_FROM:-$EXPECT}"
export PATH="/opt/homebrew/opt/postgresql@17/bin:$HOME/go/bin:$PATH"
PSQL=(psql -h localhost -p 5433 -U biletflow -X -q)
LOG="$SB/logs/$WB"
mkdir -p "$LOG"

fail=0
say() { printf '%s\n' "$*"; }
bad() { printf 'FAIL %s\n' "$*"; fail=1; }

# --- 0. песочница --------------------------------------------------------------
if [ ! -d "$SB/repo" ]; then
  git clone -q --bare "$HOME/Desktop/ticket-project" "$SB/origin.git"
  git clone -q "$SB/origin.git" "$SB/repo"
fi
cd "$SB/repo"
git fetch -q origin --prune
base_sha="$(git rev-parse --verify "$BASE^{commit}")" || { echo "нет базы $BASE"; exit 2; }
git push -q -f origin "$base_sha:refs/heads/$WB"
# ветки задач прошлого прогона этой недели убираются, чтобы скрипты не отказались
for b in $(git ls-remote --heads origin "$WB-*" | awk '{print $2}'); do
  git push -q origin ":$b"
done
git fetch -q origin --prune

# --- проверки --------------------------------------------------------------------
dbcheck() {  # $1 = дерево
  local root="$1" f out st n fl
  "${PSQL[@]}" -d postgres -c "DROP DATABASE IF EXISTS biletflow WITH (FORCE)" -c "CREATE DATABASE biletflow" >/dev/null
  for f in "$root"/db/init/*.sql; do
    "${PSQL[@]}" -d biletflow -v ON_ERROR_STOP=1 -f "$f" >/dev/null 2>"$LOG/db-init.err" || { bad "db/init $(basename "$f")"; cat "$LOG/db-init.err"; }
  done
  if [ -d "$root/db/seed" ]; then
    for f in "$root"/db/seed/*.sql; do
      "${PSQL[@]}" -d biletflow -v ON_ERROR_STOP=1 -f "$f" >/dev/null 2>"$LOG/db-seed.err" || { bad "db/seed $(basename "$f")"; cat "$LOG/db-seed.err"; }
    done
  fi
  ( cd "$root/db/tests" || exit 1
    for f in [0-9][0-9]_*.sql; do
      out=$("${PSQL[@]}" -d biletflow -v ON_ERROR_STOP=1 -f "$f" 2>&1); st=$?
      n=$(printf '%s' "$out" | grep -c 'ok - '); fl=$(printf '%s' "$out" | grep -c 'FAIL:')
      if [ $st -ne 0 ] || [ "$fl" -ne 0 ] || [ "$n" -eq 0 ]; then
        echo "FAIL db/tests/$f"; printf '%s\n' "$out" | grep -v 'ok - ' | tail -6
      else
        echo "  db/tests/$f: $n"
      fi
    done ) | tee "$LOG/db.txt"
  grep -q '^FAIL' "$LOG/db.txt" && fail=1
  return 0
}

gocheck() {  # $1 = дерево, $2 = full|build
  local root="$1"
  ( cd "$root/api" || exit 1
    unformatted="$(gofmt -l .)"; [ -z "$unformatted" ] || { echo "gofmt: $unformatted"; exit 1; }
    go vet ./... || exit 1
    "${PSQL[@]}" -d postgres -c "DROP DATABASE IF EXISTS biletflow_test WITH (FORCE)" >/dev/null
    go test -count=1 ./... 2>&1 | tail -25
    exit "${PIPESTATUS[0]}"
  ) >"$LOG/go.txt" 2>&1 || { bad "go ($root)"; tail -30 "$LOG/go.txt"; return; }
  say "  go: gofmt, vet, test ok"
}

webcheck() {  # $1 = дерево, $2 = full|quick
  local root="$1" mode="$2"
  [ -d "$root/web/node_modules" ] || cp -Rc "$NM_FROM/web/node_modules" "$root/web/"
  ( cd "$root/web" || exit 1
    npm run -s lint || exit 1
    npm run -s typecheck || exit 1
    if find src -name '*.test.ts*' | grep -q .; then npx vitest run || exit 1; fi
    if [ "$mode" = full ]; then npm run -s build >/dev/null || exit 1; fi
  ) >"$LOG/web.txt" 2>&1 || { bad "web ($mode)"; tail -30 "$LOG/web.txt"; return; }
  say "  web: lint, typecheck$([ "$mode" = full ] && echo ', build') ok"
}

mobilecheck() {
  local root="$1"
  [ -d "$root/mobile/node_modules" ] || cp -Rc "$NM_FROM/mobile/node_modules" "$root/mobile/"
  ( cd "$root/mobile" && npx tsc --noEmit ) >"$LOG/mobile.txt" 2>&1 || { bad "mobile tsc"; tail -20 "$LOG/mobile.txt"; return; }
  say "  mobile: tsc ok"
}

# --- 1–2. скрипты ----------------------------------------------------------------
tail -n +2 "$WDIR/tickets.tsv" | while IFS=$'\t' read -r ticket who branch base message _ _; do
  if ! ALINUR_EMAIL="${ALINUR_EMAIL:-alinur@example.test}" YES=1 REPO="$SB/repo" \
      bash "$WDIR/$who-$ticket.sh" >"$LOG/run-$ticket.txt" 2>&1; then
    echo "FAIL скрипт $who-$ticket.sh"; cat "$LOG/run-$ticket.txt"
  fi
done | tee "$LOG/runs.txt"
grep -q '^FAIL' "$LOG/runs.txt" && fail=1
git fetch -q origin --prune

# --- 3. каждая ветка отдельно ----------------------------------------------------
say "== ветки задач"
while IFS=$'\t' read -r ticket who branch base message _ _; do
  say "-- $ticket ($who) $branch"
  if ! git rev-parse -q --verify "origin/$branch" >/dev/null; then bad "$branch не запушена"; continue; fi
  author="$(git log -1 --format='%an <%ae>' "origin/$branch")"
  subject="$(git log -1 --format='%s' "origin/$branch")"
  body="$(git log -1 --format='%b' "origin/$branch")"
  [ "$subject" = "$message" ] || bad "$ticket: сообщение «$subject»"
  if printf '%s' "$body" | grep -qi 'co-authored-by'; then bad "$ticket: в коммите есть Co-Authored-By"; fi
  count="$(git rev-list --count "origin/${base#origin/}..origin/$branch")"
  [ "$count" = 1 ] || bad "$ticket: коммитов поверх базы $count, ожидался 1"
  say "   $author · $(git diff --shortstat -M "origin/${base#origin/}" "origin/$branch")"
  wt="$SB/wt-$ticket"; rm -rf "$wt"; git worktree prune
  git worktree add -q --detach "$wt" "origin/$branch"
  changed="$(git diff --name-only "origin/${base#origin/}" "origin/$branch")"
  if printf '%s\n' "$changed" | grep -q '^api/'; then gocheck "$wt"; fi
  if printf '%s\n' "$changed" | grep -q '^db/'; then dbcheck "$wt" >/dev/null; grep '^FAIL' "$LOG/db.txt" || say "  db: ok"; fi
  if printf '%s\n' "$changed" | grep -q '^web/'; then webcheck "$wt" quick; fi
  if printf '%s\n' "$changed" | grep -q '^mobile/'; then mobilecheck "$wt"; fi
  git worktree remove --force "$wt"
done < <(tail -n +2 "$WDIR/tickets.tsv")

# --- 4. слияние в порядке PR -------------------------------------------------------
say "== слияние в $WB"
mw="$SB/merged-$WB"; rm -rf "$mw"; git worktree prune
git worktree add -q -B "sim-$WB" "$mw" "origin/$WB"
n=100
while IFS=$'\t' read -r ticket who branch base message _ _; do
  n=$((n + 1))
  if ! git -C "$mw" -c user.name="Ansar Zeinulla" -c user.email="ansar.zeinulla.a@gmail.com" \
      merge -q --no-ff "origin/$branch" -m "Merge pull request #$n from ansarzeinulla/$branch" >"$LOG/merge.txt" 2>&1; then
    bad "конфликт при слиянии $branch"; cat "$LOG/merge.txt"; git -C "$mw" merge --abort
  fi
done < <(tail -n +2 "$WDIR/tickets.tsv")
git -C "$mw" push -q -f origin "sim-$WB:refs/heads/$WB"

# --- 5. итог ---------------------------------------------------------------------
say "== итог $WB"
gocheck "$mw"
dbcheck "$mw"
[ -d "$mw/web" ] && webcheck "$mw" full
[ -d "$mw/mobile" ] && mobilecheck "$mw"
if [ -n "$EXPECT" ]; then
  if diff -rq -x node_modules -x .next -x '*.tsbuildinfo' -x next-env.d.ts -x .git -x .expo \
      "$EXPECT" "$mw" >"$LOG/diff.txt" 2>&1; then
    say "  дерево совпадает с эталоном"
  else
    bad "дерево отличается от эталона:"; cat "$LOG/diff.txt"
  fi
fi

[ $fail -eq 0 ] && say "ВСЁ ЗЕЛЁНОЕ: $WB" || say "ЕСТЬ ОШИБКИ: $WB"
exit $fail
