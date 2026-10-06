#!/usr/bin/env bash
# Общая часть: скопировать готовые файлы задачи в чистую временную копию
# репозитория, закоммитить под именем студента и запушить его ветку.
#
#   _commit.sh <задача> <S1..S5> <ветка> <база> "<сообщение коммита>"
#
# Основное рабочее дерево ~/Desktop/ticket-project не трогается: всё делается в
# отдельном git worktree, который удаляется в конце.
set -euo pipefail

TICKET="$1" WHO="$2" BRANCH="$3" BASE="$4" MESSAGE="$5"
REPO="${REPO:-$HOME/Desktop/ticket-project}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$HERE/code/$TICKET"

case "$WHO" in
  S1) NAME="Ansar Zeinulla";    EMAIL="ansar.zeinulla.a@gmail.com" ;;
  S2) NAME="Alibi Takhtanov";   EMAIL="takhtanovalb@gmail.com" ;;
  S3) NAME="Abylay Otaubay";    EMAIL="o.abyl247@gmail.com" ;;
  S4) NAME="Alinur Burlybayev"; EMAIL="${ALINUR_EMAIL:-alinur.burlybayev@nu.edu.kz}" ;;
  S5) NAME="Olzhas Nurseit";    EMAIL="olzhasnrseit@gmail.com" ;;
  *) echo "неизвестный студент: $WHO" >&2; exit 2 ;;
esac
if [ -z "$EMAIL" ]; then
  echo "У $NAME нет почты GitHub. Запустите так: ALINUR_EMAIL=почта@аккаунта bash $0" >&2
  exit 2
fi
[ -d "$SRC" ] || { echo "нет готовых файлов: $SRC" >&2; exit 2; }

cd "$REPO"
git fetch -q origin --prune
if ! git rev-parse -q --verify "$BASE" >/dev/null; then
  # Ветку базы могли уже влить и удалить (GitHub удаляет ветку после merge).
  # Тогда её коммит уже лежит в интеграционной ветке недели, и строиться нужно
  # поверх неё: PR покажет ровно один коммит — свой.
  INTEGRATION="$(printf '%s' "$BRANCH" | sed -E 's/^((week|phase)-[0-9]+).*/\1/')"
  BASE_TICKET="$(printf '%s' "$BASE" | sed -E 's/.*-(BF-[A-Za-z0-9-]+)$/\1/')"
  # В переменную, а не через конвейер: под pipefail `git log | grep -q` даёт 141.
  WEEK_SUBJECTS=""
  if git rev-parse -q --verify "origin/$INTEGRATION" >/dev/null; then
    WEEK_SUBJECTS="$(git log --format=%s "origin/$INTEGRATION")"
  fi
  if grep -q "^$BASE_TICKET " <<<"$WEEK_SUBJECTS"; then
    echo "База $BASE уже влита в $INTEGRATION и удалена — строим поверх origin/$INTEGRATION."
    BASE="origin/$INTEGRATION"
  else
    echo "Нет базы $BASE. Сначала должна быть запушена задача, от которой эта зависит." >&2
    exit 1
  fi
fi

WT="$(mktemp -d /tmp/${TICKET}.XXXX)"
trap 'git -C "$REPO" worktree remove --force "$WT" >/dev/null 2>&1 || true' EXIT
git worktree add -q -B "$BRANCH" "$WT" "$BASE"

cp -R "$SRC/." "$WT/"
cd "$WT"
FILES="$(cd "$SRC" && find . -type f | sed 's|^\./||' | sort)"
echo "$FILES" | while IFS= read -r f; do git add -A -- "$f"; done
# удалённые в задаче файлы (если были) тоже должны попасть в коммит
git add -u

if git diff --cached --quiet; then
  echo "Нечего коммитить: файлы $TICKET уже в $BASE." ; exit 0
fi

echo "── $TICKET · $NAME <$EMAIL> → $BRANCH (от $BASE)"
git diff --cached --stat | tail -20
git -c user.name="$NAME" -c user.email="$EMAIL" commit -q -m "$MESSAGE"
git log -1 --format='   коммит %h · автор %an · коммиттер %cn · %s'
if [ "${DRY_RUN:-0}" = "1" ]; then
  echo "   DRY_RUN: коммит создан, но не запушен; локальная ветка удаляется"
  cd "$REPO"; git worktree remove --force "$WT"; git branch -D "$BRANCH" >/dev/null
  exit 0
fi
git push -q -u origin "$BRANCH"
echo "   запушено: $BRANCH"
echo "   PR: https://github.com/ansarzeinulla/ticket-project/pull/new/$BRANCH (base: phase-01)"
