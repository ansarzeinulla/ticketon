"""Семь проверок самого плана. Одна команда вместо пяти скриптов.

    python3 plan/tools/check-plan.py           # всё
    python3 plan/tools/check-plan.py -q        # только итог

Что проверяется:

  1. у каждого файла репозитория есть владелец в ownership.map;
  2. ни один файл в реальной git-истории не трогали два разных автора;
  3. timeline.md не противоречит карте владения;
  4. каждый файл где-то создаётся, иначе план не дойдёт до нынешней версии;
  5. ссылки `BF-*` в остальных файлах плана существуют;
  6. смесь операций A/M/D/R похожа на живую разработку, и переходные файлы
     исчезают до контрольной точки W10;
  7. переходы между неделями физически возможны (см. diff-weeks.py).

Возвращает 1, если хоть одна проверка не прошла.
"""
import collections
import fnmatch
import importlib.util
import os
import re
import subprocess
import sys

ROOT = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
os.chdir(ROOT)
QUIET = "-q" in sys.argv
# `weeks/` — снимки, порождённые из плана. Для инструментов это не код
# проекта: иначе они начнут планировать сами себя.
FILES = {f for f in subprocess.check_output(["git", "ls-files"], text=True).split()
         if not f.startswith("weeks/")}
OWNERS = ["S1", "S2", "S3", "S4", "S5"]
G, R, Y, Z = "\033[32m", "\033[31m", "\033[33m", "\033[0m"

_spec = importlib.util.spec_from_file_location(
    "diffphases", os.path.join(ROOT, "plan/tools/diff-weeks.py"))
dp = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(dp)

fails = []


def ok(text, detail=""):
    if not QUIET:
        print(f"  {G}OK{Z}    {text}" + (f" — {detail}" if detail else ""))


def bad(text, items=()):
    fails.append(text)
    print(f"  {R}НАРУШЕНИЕ{Z}  {text}")
    for i in list(items)[:10]:
        print(f"          {i}")


def note(text, items=()):
    if QUIET:
        return
    print(f"  {Y}NOTE{Z}  {text}")
    for i in list(items)[:6]:
        print(f"          {i}")


# ---------------------------------------------------------------- карта
rules = []
for line in open("plan/ownership.map"):
    line = line.rstrip("\n")
    if line.strip() and not line.lstrip().startswith("#") and "\t" in line:
        a, b = line.split("\t", 1)
        rules.append((a, b))


def owner_of(path):
    hit, extra = None, 0
    for o, g in rules:
        if fnmatch.fnmatchcase(path, g.replace("**", "*")):
            if hit is None:
                hit = o
            elif o != hit:
                extra += 1
    return hit, extra


# --- 1. покрытие ------------------------------------------------------------
counts, orphans, ambig = collections.Counter(), [], []
for f in sorted(FILES):
    o, extra = owner_of(f)
    if o is None:
        orphans.append(f)
    else:
        counts[o] += 1
        if extra:
            ambig.append(f"{f:52} -> {o}")
if not QUIET:
    print(f"\nПроверка плана\n{'=' * 66}\n  файлов в репозитории: {len(FILES)}")
    for o in OWNERS:
        print(f"    {o:3} {counts[o]:4} файлов  ({counts[o] * 100 // len(FILES)}%)")
if orphans:
    bad(f"{len(orphans)} файлов без владельца", orphans)
else:
    ok(f"у всех {len(FILES)} файлов есть владелец")
if ambig:
    note(f"{len(ambig)} файлов подходят под правила разных владельцев "
         "(побеждает первое — проверьте порядок в карте)", ambig)

# --- 2. непересечение авторов в реальной истории ----------------------------
log = subprocess.run(["git", "log", "--format=%H%x09%an", "--name-only"],
                     capture_output=True, text=True).stdout
author, byfile = None, collections.defaultdict(set)
for line in log.split("\n"):
    if "\t" in line:
        author = line.split("\t", 1)[1]
    elif line.strip() and author:
        byfile[line.strip()].add(author)
shared = {f: a for f, a in byfile.items() if len(a) > 1}
if shared:
    bad(f"{len(shared)} файлов трогали разные авторы",
        [f"{f} — {', '.join(sorted(a))}" for f, a in sorted(shared.items())])
else:
    ok("ни один файл не трогали два разных автора")

# --- разбор таймлайна один раз ----------------------------------------------
ops, titles = dp.read_timeline()

# --- 3. план не противоречит карте ------------------------------------------
byname = collections.defaultdict(list)
for f in FILES:
    byname[f.split("/")[-1]].append(f)
viol, checked = [], 0
for phase, idx, who, tid, commit, op, f, dst in ops:
    for path in filter(None, (f, dst)):
        checked += 1
        if path not in FILES and "/" not in path and len(byname.get(path, [])) > 1:
            viol.append(f"{who}: неоднозначная ссылка `{path}`")
            continue
        o, _ = owner_of(path)
        if o is None:
            viol.append(f"{who}: нет правила владения для `{path}`")
        elif o != who:
            viol.append(f"{who} vs владелец {o}: {path}")
if viol:
    bad("timeline.md противоречит карте владения", viol)
else:
    ok("timeline.md не противоречит карте", f"{checked} ссылок")

# --- 4. каждый файл где-то создаётся ----------------------------------------
created = {f for *_, op, f, dst in ops if op == "A"} | {dst for *_, op, f, dst in ops
                                                        if op == "R" and dst}
missing = sorted(FILES - created)
if missing:
    bad(f"{len(missing)} файлов план никогда не создаёт", missing)
else:
    ok(f"каждый файл где-то создаётся — {len(FILES)} из {len(FILES)}")

# --- 5. ссылки BF-* в остальных файлах плана --------------------------------
known = {}
for phase, idx, who, tid, commit, *_ in ops:
    known.setdefault(tid, set()).add(commit)
broken = []
for name in sorted(os.listdir("plan")) + \
        [f"reports/{n}" for n in sorted(os.listdir("plan/reports"))]:
    path = os.path.join("plan", name)
    if not path.endswith(".md") or name == "timeline.md":
        continue
    for tid, text in re.findall(r"`(BF-B?\d+) ([^`]+)`", open(path).read()):
        if tid not in known:
            broken.append(f"{path}: {tid} — нет такого тикета")
        elif text not in known[tid]:
            broken.append(f"{path}: {tid} «{text}» — нет такого коммита")
if broken:
    bad("битые ссылки на тикеты", broken)
else:
    ok("ссылки на тикеты сходятся", f"{len(known)} тикетов")

# --- 6. смесь операций и инвариант W10 --------------------------------------
mix = collections.Counter(op for *_, op, f, dst in ops)
total = sum(mix.values()) or 1
pct = {k: mix[k] * 100 / total for k in "AMDR"}
commits = len({(p, i) for p, i, *_ in ops})
problems = []
# Порог 52%, а не 25% как в зрелом репозитории. Причина арифметическая: при
# правиле «один тикет — один коммит» коммитов около 110, а файлов 325, поэтому
# примерно половина всех операций — неизбежно создания. Выше 52% это уже
# означает, что файлы рождаются готовыми и к ним не возвращаются.
if pct["A"] > 52:
    problems.append(f"слишком много созданий: A {pct['A']:.0f}% — ищите файлы, "
                    "которые должны были меняться, а не рождаться готовыми")
if pct["M"] < 40:
    problems.append(f"слишком мало правок: M {pct['M']:.0f}%")
if pct["D"] < 2:
    problems.append(f"почти ничего не удаляется: D {pct['D']:.1f}%")
if mix["R"] < 1:
    problems.append("ни одного переименования за весь семестр")
dead = {}
for phase, idx, who, tid, commit, op, f, dst in ops:
    if op in ("D", "R"):
        dead[f] = phase
for f, p in dead.items():
    if p > 10:
        problems.append(f"{f} исчезает в W{p} — после W10 дифф с нынешним кодом пуст")
line = (f"A {pct['A']:.1f}%  M {pct['M']:.1f}%  D {pct['D']:.1f}%  "
        f"R {mix['R']}  ({commits} коммитов)")
if problems:
    bad("смесь операций: " + line, problems)
else:
    ok("смесь операций", line)

# --- 7. переходы между неделями -----------------------------------------------
_, _, impossible = dp.analyse(ops)
if impossible:
    bad("невозможные переходы между неделями", [t for _, t in sorted(impossible)])
else:
    ok("переходы между неделями возможны",
       "ничего не меняют до создания и не трогают после удаления")

print("=" * 66)
print(f"  {R}есть замечания: {len(fails)}{Z}" if fails else f"  {G}всё чисто{Z}")
print()
sys.exit(1 if fails else 0)
