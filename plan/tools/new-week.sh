#!/usr/bin/env bash
# Создаёт папку и ветку под одну фазу.
#
#   bash plan/tools/new-week.sh 3            # W3: ветка week-03 + папка на столе
#   bash plan/tools/new-week.sh 3 --dry-run  # только показать, что будет сделано
#   bash plan/tools/new-week.sh 3 --dir ~/work/biletflow
#
# Папка — это git worktree, а не копия. Отличия принципиальные:
#   * 7.5 МБ на фазу вместо 1.6 ГБ (node_modules не дублируется);
#   * содержимое папки — это и есть состояние ветки, разойтись они не могут;
#   * `git log` внутри папки показывает историю именно этой недели.
#
# W0 начинается с ПУСТОГО дерева: смысл семестра в том, чтобы дойти до нынешней
# версии с нуля. Нынешний код остаётся в `main` как справочник — см.
# plan/file-histories.md о том, как писать v1, а не копировать финальный файл.
set -euo pipefail

N=""; DRY=0; DIR="$HOME/Desktop/biletflow-phases"
while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY=1 ;;
    --dir) DIR="$2"; shift ;;
    -h|--help) sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) N="$1" ;;
  esac
  shift
done

case "$N" in
  ''|*[!0-9]*) echo "укажите номер недели: bash $0 <0..12>" >&2; exit 2 ;;
esac
[ "$N" -le 12 ] || { echo "недель всего 13: W0…W12" >&2; exit 2; }

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
PAD=$(printf '%02d' "$N")
BRANCH="week-$PAD"
WT="$DIR/week-$PAD"

run() {
  if [ "$DRY" -eq 1 ]; then printf '  \033[2m$ %s\033[0m\n' "$*"; else "$@"; fi
}

echo
printf '\033[1mФ%s → ветка %s → %s\033[0m\n' "$N" "$BRANCH" "$WT"
echo

if git show-ref --verify --quiet "refs/heads/$BRANCH"; then
  echo "  ветка $BRANCH уже есть — переиспользуем"
elif [ "$N" -eq 0 ]; then
  echo "  W0 начинается с пустого дерева (orphan-ветка)"
  EMPTY=$(git hash-object -t tree /dev/null)
  if [ "$DRY" -eq 0 ]; then
    C=$(git commit-tree "$EMPTY" -m "Start the project")
    git branch "$BRANCH" "$C"
  else
    printf '  \033[2m$ git commit-tree %s -m "Start the project" | git branch %s\033[0m\n' \
      "$EMPTY" "$BRANCH"
  fi
else
  PREV="week-$(printf '%02d' $((N - 1)))"
  git show-ref --verify --quiet "refs/heads/$PREV" || {
    echo "  нет ветки $PREV — сначала создайте фазу $((N - 1))" >&2; exit 1; }
  # Ответвляемся от main: предыдущая неделя уже влита туда на разборе. Если нет —
  # значит фазу не закрыли, и начинать следующую рано.
  if git merge-base --is-ancestor "$PREV" main 2>/dev/null; then
    echo "  ответвляем от main (в нём уже есть $PREV)"
    run git branch "$BRANCH" main
  else
    echo "  $PREV ещё не влит в main — закройте фазу $((N - 1)) перед началом $N" >&2
    exit 1
  fi
fi

if git worktree list --porcelain | grep -qx "worktree $WT"; then
  echo "  папка уже подключена"
else
  run mkdir -p "$DIR"
  run git worktree add "$WT" "$BRANCH"
fi

echo
echo "  готово. Дальше:"
echo "    cd $WT"
echo "    открыть weeks/week-$PAD-*/captain.md   — капитану"
echo "    открыть weeks/week-$PAD-*/S<N>.md      — каждому своё"
echo
echo "  когда неделя закрыта — слить ветки в $BRANCH и создать следующую:"
echo "    bash plan/tools/new-week.sh $((N + 1))"
echo
