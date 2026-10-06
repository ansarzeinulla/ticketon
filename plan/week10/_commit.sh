#!/usr/bin/env bash
# Общая часть скриптов недели 10: скопировать готовые файлы задачи в чистую
# временную копию репозитория, закоммитить под именем студента и запушить его
# ветку. Время коммита — настоящее, соавторов нет.
#
#   _commit.sh <задача> <S1..S5> <ветка> <база> "<сообщение коммита>"
#
# Основное рабочее дерево ~/Desktop/ticket-project не трогается: всё делается в
# отдельном git worktree, который удаляется в конце.
#
#   DRY_RUN=1  закоммитить и сразу удалить ветку, ничего не пушить
#   YES=1      не спрашивать «вы — такой-то?» (для проверки скриптов)
#   REPO=...   другой клон репозитория (по умолчанию ~/Desktop/ticket-project)
set -euo pipefail

WEEK_BRANCH="week-10"
TICKET="$1" WHO="$2" BRANCH="$3" BASE="$4" MESSAGE="$5"
REPO="${REPO:-$HOME/Desktop/ticket-project}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$HERE/code/$TICKET"
DELETE_LIST="$HERE/code/$TICKET.delete"

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
[ -d "$SRC" ] || [ -f "$DELETE_LIST" ] || { echo "нет готовых файлов: $SRC" >&2; exit 2; }

echo "Задача:  $MESSAGE"
echo "Автор:   $NAME <$EMAIL>  ($WHO)"
echo "Ветка:   $BRANCH  (от $BASE)"
if [ "${YES:-0}" != "1" ]; then
  printf 'Вы — %s? Коммит будет подписан вашим именем. [y/N] ' "$NAME"
  read -r answer
  case "$answer" in
    y|Y|yes|да|Да) ;;
    *) echo "Отменено. Этот скрипт должен запускать сам $NAME."; exit 1 ;;
  esac
fi

cd "$REPO"
git fetch -q origin --prune
if ! git rev-parse -q --verify "$BASE" >/dev/null; then
  # GitHub удаляет ветку после слияния PR. Если базу уже влили в ветку недели,
  # её коммит там есть, и строиться надо поверх недели: PR тогда покажет ровно
  # один коммит — свой.
  BASE_TICKET="$(printf '%s' "$BASE" | sed -E 's/.*-(BF-[A-Za-z0-9-]+)$/\1/')"
  # Темы коммитов недели читаются в переменную, а не через `git log | grep -q`:
  # под `set -o pipefail` grep закрывает канал первым, git log получает SIGPIPE,
  # и весь конвейер возвращает 141 — условие всегда оказывалось ложным.
  WEEK_SUBJECTS=""
  if git rev-parse -q --verify "origin/$WEEK_BRANCH" >/dev/null; then
    WEEK_SUBJECTS="$(git log --format=%s "origin/$WEEK_BRANCH")"
  fi
  if [ "$BASE" != "origin/$WEEK_BRANCH" ] \
     && grep -q "^$BASE_TICKET " <<<"$WEEK_SUBJECTS"; then
    echo "База ${BASE#origin/} уже влита в $WEEK_BRANCH и удалена — строим поверх origin/$WEEK_BRANCH."
    BASE="origin/$WEEK_BRANCH"
  else
    echo "Нет базы $BASE." >&2
    if [ "$BASE" = "origin/$WEEK_BRANCH" ]; then
      echo "Капитан ещё не создал ветку недели $WEEK_BRANCH." >&2
    else
      echo "Эта задача строится поверх ${BASE#origin/}: сначала её владелец должен запустить свой скрипт." >&2
    fi
    exit 1
  fi
fi
if git rev-parse -q --verify "origin/$BRANCH" >/dev/null; then
  echo "Ветка $BRANCH уже есть на GitHub — задача уже запушена. Повторно не запускайте." >&2
  exit 1
fi

WT="$(mktemp -d "/tmp/${TICKET}.XXXX")"
trap 'git -C "$REPO" worktree remove --force "$WT" >/dev/null 2>&1 || true' EXIT
git worktree add -q -B "$BRANCH" "$WT" "$BASE"
cd "$WT"

# Сначала удаления (и старые имена переименованных файлов), потом новые файлы:
# git сам распознает переименование, когда содержимое похоже.
if [ -f "$DELETE_LIST" ]; then
  grep -v '^[[:space:]]*\(#\|$\)' "$DELETE_LIST" | while IFS= read -r path; do
    git rm -r -q --ignore-unmatch -- "$path"
  done
fi
if [ -d "$SRC" ]; then
  cp -R "$SRC/." "$WT/"
  (cd "$SRC" && find . -type f | sed 's|^\./||' | sort) | while IFS= read -r f; do
    git add -A -- "$f"
  done
fi

if git diff --cached --quiet; then
  echo "Нечего коммитить: файлы $TICKET уже в $BASE."; exit 0
fi

git diff --cached --stat -M | tail -25
git -c user.name="$NAME" -c user.email="$EMAIL" commit -q -m "$MESSAGE"
git log -1 --format='   коммит %h · автор %an <%ae> · %ad' --date=format:'%d.%m %H:%M'
if [ "${DRY_RUN:-0}" = "1" ]; then
  echo "   DRY_RUN: коммит создан, но не запушен; локальная ветка удаляется"
  cd "$REPO"; git worktree remove --force "$WT"; git branch -D "$BRANCH" >/dev/null
  exit 0
fi
git push -q -u origin "$BRANCH"
cd "$REPO"; git worktree remove --force "$WT"; git branch -D "$BRANCH" >/dev/null 2>&1 || true
echo "   запушено: $BRANCH"
echo "   PR: https://github.com/ansarzeinulla/ticket-project/compare/$WEEK_BRANCH...$BRANCH?expand=1"
