#!/usr/bin/env bash
# Forensic self-check for the BiletFlow history.
#
# Answers one question: does `git log` look like thirteen weeks of five people
# building something, or like a finished project poured into a few commits?
#
#   bash plan/tools/check-history.sh              # whole history
#   bash plan/tools/check-history.sh origin/main..HEAD

set -uo pipefail
RANGE="${1:-}"
cd "$(git rev-parse --show-toplevel)" || exit 1

pass=0; fail=0
ok()   { printf '  \033[32mPASS\033[0m  %-42s %s\n' "$1" "$2"; pass=$((pass+1)); }
bad()  { printf '  \033[31mFAIL\033[0m  %-42s %s\n' "$1" "$2"; fail=$((fail+1)); }
note() { printf '        %-42s %s\n' "$1" "$2"; }

# Lockfiles and binaries are legitimately committed whole; exclude them from
# the "no finished file lands at once" rule.
BULK='package-lock.json|go.sum|\.ttf$|\.png$|\.ico$|postman_collection\.json'

echo
echo "BiletFlow history check ${RANGE:+($RANGE)}"
echo "=================================================================="

# --- 1. the headline: are lines ever deleted? ----------------------------
read -r ins del < <(git log $RANGE --numstat --format='' -- . \
  | awk '$1 ~ /^[0-9]+$/ {i+=$1; d+=$2} END {print i+0, d+0}')
if [ "$ins" -gt 0 ]; then
  ratio=$(awk -v d="$del" -v i="$ins" 'BEGIN{printf "%.3f", d/i}')
  if awk -v r="$ratio" 'BEGIN{exit !(r>=0.20)}'; then
    ok "deleted/inserted lines >= 0.20" "$ratio  (${del} / ${ins})"
  else
    bad "deleted/inserted lines >= 0.20" "$ratio  (${del} / ${ins})  <- code is only ever added"
  fi
fi

# --- 2. do commits modify, or only add? ----------------------------------
total=$(git log $RANGE --format='%H' | wc -l | tr -d ' ')
withM=$(git log $RANGE --format='%H' --name-status \
  | awk '/^[0-9a-f]{40}$/{h=$0; next} /^M/{if(!seen[h]++) n++} END{print n+0}')
if [ "$total" -gt 0 ]; then
  pct=$(( withM * 100 / total ))
  [ "$pct" -ge 60 ] \
    && ok "commits touching an existing file >= 60%" "${pct}%  (${withM}/${total})" \
    || bad "commits touching an existing file >= 60%" "${pct}%  (${withM}/${total})"
fi

# --- 3. A / M / D / R mix ------------------------------------------------
# Ориентир ~46/50/3, а не ~25/70/5 как в зрелом репозитории. Причина
# арифметическая: 325 файлов рождаются с нуля, а коммитов около 110 (правило
# «один тикет — один коммит»), поэтому почти половина всех операций — создания.
# Ровно столько предсказывает plan/timeline.md: сверяйте факт с планом, а не с
# абстрактным идеалом. Точные числа плана: python3 plan/tools/check-plan.py
git log $RANGE --name-status --format='' \
  | awk '/^[AMDR]/{c[substr($1,1,1)]++} END{
      t=c["A"]+c["M"]+c["D"]+c["R"]; if(t==0) t=1
      printf "  ----  %-42s A %d%%  M %d%%  D %d%%  R %d\n", "file-operation mix (want ~46/50/3)", \
        c["A"]*100/t, c["M"]*100/t, c["D"]*100/t, c["R"]+0 }'
renames=$(git log $RANGE --name-status --format='' | grep -c '^R' || true)
[ "$renames" -ge 1 ] \
  && ok "at least one rename" "$renames" \
  || bad "at least one rename" "0  <- nothing was ever moved in 13 weeks"
deletes=$(git log $RANGE --name-status --format='' | grep -c '^D' || true)
[ "$deletes" -ge 1 ] \
  && ok "at least one file deleted" "$deletes" \
  || bad "at least one file deleted" "0  <- nothing was ever thrown away"

# --- 4. hub churn --------------------------------------------------------
for hub in api/internal/api/server.go db/init/02_schema.sql web/src/lib/api.ts; do
  n=$(git log $RANGE --oneline --follow -- "$hub" 2>/dev/null | wc -l | tr -d ' ')
  want=10; [ "$hub" = "api/internal/api/server.go" ] && want=30
  [ "$n" -ge "$want" ] \
    && ok "commits touching $(basename "$hub") >= $want" "$n" \
    || bad "commits touching $(basename "$hub") >= $want" "$n"
done

# --- 5. commit shape -----------------------------------------------------
med=$(git log $RANGE --format='%H' --name-status \
  | awk '/^[0-9a-f]{40}$/{if(n)print n; n=0; next} /^[AMDR]/{n++} END{if(n)print n}' \
  | sort -n | awk '{a[NR]=$1} END{if(NR)print (NR%2)?a[(NR+1)/2]:int((a[NR/2]+a[NR/2+1])/2)}')
if [ -n "${med:-}" ]; then
  [ "$med" -ge 2 ] && [ "$med" -le 6 ] \
    && ok "median files per commit in 2..6" "$med" \
    || bad "median files per commit in 2..6" "$med"
fi

big=$(git log $RANGE --format='%H %s' | while read -r h s; do
        n=$(git show --numstat --format='' "$h" -- . \
            | grep -Ev "$BULK" | awk '$1 ~ /^[0-9]+$/{t+=$1+$2} END{print t+0}')
        echo "$n $h ${s:0:44}"
      done | sort -rn | head -1)
bign=${big%% *}
[ "${bign:-0}" -lt 600 ] \
  && ok "largest non-bulk commit < 600 lines" "$bign" \
  || bad "largest non-bulk commit < 600 lines" "$bign  (${big#* })"

# --- 6. is this a team? --------------------------------------------------
authors=$(git log $RANGE --format='%an' | sort -u | wc -l | tr -d ' ')
[ "$authors" -ge 5 ] \
  && ok "distinct authors >= 5" "$authors" \
  || bad "distinct authors >= 5" "$authors"
git log $RANGE --format='%an' | sort | uniq -c | sort -rn \
  | awk -v t="$total" '{printf "        %-42s %s (%d%%)\n", (NR==1?"author share":""), $2" "$3, $1*100/t}'

merges=$(git log $RANGE --merges --oneline | wc -l | tr -d ' ')
[ "$merges" -ge 1 ] \
  && ok "merge commits present" "$merges" \
  || bad "merge commits present" "0  <- branch-per-issue was claimed"

# --- 7. steady work ------------------------------------------------------
# days_from_civil, so this works with BSD awk too (no mktime).
gap=$(git log $RANGE --format='%ad' --date=short | sort -u | awk '
  function days(y,m,d,  era,yoe,doy,doe) {
    if (m <= 2) y--
    era = int((y >= 0 ? y : y-399) / 400)
    yoe = y - era * 400
    doy = int((153 * (m + (m > 2 ? -3 : 9)) + 2) / 5) + d - 1
    doe = yoe * 365 + int(yoe/4) - int(yoe/100) + doy
    return era * 146097 + doe - 719468
  }
  { split($1, p_, "-"); t = days(p_[1]+0, p_[2]+0, p_[3]+0)
    if (prev != "") { g = t - prev; if (g > m) m = g }
    prev = t } END { print m+0 }')
[ "${gap:-0}" -le 4 ] \
  && ok "longest gap between commit days <= 4" "${gap}d" \
  || bad "longest gap between commit days <= 4" "${gap}d"
span=$(git log $RANGE --format='%ad' --date=short | sort -u | wc -l | tr -d ' ')
note "distinct days with commits" "$span"

echo "=================================================================="
printf "  %d passed, %d failed\n\n" "$pass" "$fail"
[ "$fail" -eq 0 ]
