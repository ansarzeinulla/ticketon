"""Раскладывает проект по папкам недель внутри репозитория.

    python3 plan/tools/build-weeks.py            # собрать все 13
    python3 plan/tools/build-weeks.py 3          # только W3
    python3 plan/tools/build-weeks.py --clean    # снести и собрать заново

Получается:

    weeks/
      README.md
      week-03-events/
        S1.md … S5.md   пошаговая инструкция каждому: ветка, файлы, команды, Jira
        captain.md      что делает капитан: открыть неделю, ревью, закрыть, тег
        code/           состояние проекта на конец недели

Папка `weeks/` в .gitignore. Это принципиально: если разложить недели внутри
коммитов, проверяющий увидит `weeks/week-01/` рядом с кодом, и реконструкция
станет очевидной. Папки нужны, чтобы СМОТРЕТЬ на рост проекта, а историю
показывает git.

`node_modules` не копируется — в каждой папке на него симлинк в корень, один
на всех. Иначе 11 недель заняли бы ~21 ГБ вместо ~60 МБ.

Содержимое файлов берётся из нынешнего кода. Это скелет «какие файлы должны
существовать к концу недели», а не готовый коммит: v1 файла пишется руками, см.
plan/file-histories.md.
"""
import fnmatch
import os
import re
import shutil
import subprocess
import sys

ROOT = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
os.chdir(ROOT)
OUT = os.path.join(ROOT, "weeks")
# `weeks/` — снимки, порождённые из плана. Для инструментов это не код
# проекта: иначе они начнут планировать сами себя.
FILES = {f for f in subprocess.check_output(["git", "ls-files"], text=True).split()
         if not f.startswith("weeks/")}

SLUG = {0: "discovery", 1: "skeleton", 2: "identity", 3: "events",
        4: "paid-tickets", 5: "gate", 6: "refunds", 7: "analytics",
        8: "release-shape", 9: "verification", 10: "release"}
COMMENT = {
    ".go": "//", ".ts": "//", ".tsx": "//", ".mts": "//", ".js": "//", ".mjs": "//",
    ".sql": "--", ".sh": "#", ".yml": "#", ".yaml": "#", ".md": "<!--",
}


def expand(ref):
    m = re.search(r"\{([^}]*)\}", ref)
    if not m:
        return [ref]
    return [ref[:m.start()] + a.strip() + ref[m.end():] for a in m.group(1).split(",")]


def parse_timeline():
    """Возвращает (born, dead, rows) — в какой неделе файл появился и исчез."""
    born, dead, rows = {}, {}, []
    phase = None
    for line in open("plan/timeline.md"):
        m = re.match(r"^## W(\d+) — (.+)$", line)
        if m:
            phase = int(m.group(1))
            continue
        r = re.match(r"^\| (S[1-5]) \| `(BF-B?\d+) ([^`]+)` \| (.*?)\s*\|\s*$", line)
        if not r or phase is None:
            continue
        rows.append((phase,) + r.groups())
        body = r.group(4)
        # Переименование: источник умирает, цель рождается.
        for src, dst in re.findall(r"\*\*R\*\* `([^`]+)` → `([^`]+)`", body):
            dead[resolve(src)] = phase
            born.setdefault(resolve(dst), phase)
        for op, ref in re.findall(r"\*\*([AMD])\*\* `([^`]+)`", body):
            for one in expand(ref):
                if "*" in one:
                    if op == "A":
                        pat = one.replace("**", "*")
                        for f in FILES:
                            if fnmatch.fnmatchcase(f, pat) or fnmatch.fnmatchcase(f, pat + "*"):
                                born.setdefault(f, phase)
                    continue
                f = resolve(one)
                if op == "A":
                    born.setdefault(f, phase)
                elif op == "D":
                    dead[f] = phase
    return born, dead, rows


def resolve(ref):
    if ref in FILES:
        return ref
    cand = [f for f in FILES if f.endswith("/" + ref)]
    return cand[0] if len(cand) == 1 else ref


def alive_at(born, dead, n):
    return sorted(f for f, b in born.items() if b <= n and dead.get(f, 99) > n)


def stub(path, born_phase, dead_phase):
    """Файл, которого нет в нынешнем коде: он жил только часть семестра."""
    ext = os.path.splitext(path)[1]
    c = COMMENT.get(ext, "#")
    close = " -->" if c == "<!--" else ""
    return (
        f"{c} Переходный файл: существовал с W{born_phase} по W{dead_phase}.{close}\n"
        f"{c} В нынешнем репозитории его нет — он был вытеснен, см. plan/timeline.md.{close}\n"
        f"{c} Содержимое пишется руками: plan/file-histories.md.{close}\n"
    )


def frozen():
    out = set()
    try:
        for line in open("plan/frozen.txt"):
            line = line.split("#")[0].strip()
            if line.isdigit():
                out.add(int(line))
    except FileNotFoundError:
        pass
    return out


def build(n, born, dead):
    name = f"week-{n:02d}-{SLUG[n]}"
    d = os.path.join(OUT, name)
    code = os.path.join(d, "code")
    shutil.rmtree(d, ignore_errors=True)
    os.makedirs(code, exist_ok=True)

    files = alive_at(born, dead, n)
    copied = stubs = 0
    for f in files:
        dst = os.path.join(code, f)
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        # Файл может числиться в git, но быть удалён с диска — тогда копировать
        # нечего, и он попадает в снимок заглушкой.
        if f in FILES and os.path.exists(os.path.join(ROOT, f)):
            shutil.copy2(os.path.join(ROOT, f), dst)
            copied += 1
        else:
            with open(dst, "w") as fh:
                fh.write(stub(f, born[f], dead.get(f, 99)))
            stubs += 1

    # node_modules общий: симлинк вместо копии.
    for mod in ("web", "mobile"):
        src = os.path.join(ROOT, mod, "node_modules")
        if os.path.isdir(src) and os.path.isdir(os.path.join(code, mod)):
            link = os.path.join(code, mod, "node_modules")
            if not os.path.lexists(link):
                os.symlink(src, link)

    return name, len(files), copied, stubs


def main():
    args = [a for a in sys.argv[1:] if a != "--clean"]
    if "--clean" in sys.argv:
        keep = frozen()
        for name in os.listdir(OUT) if os.path.isdir(OUT) else []:
            m = re.match(r"week-(\d+)-", name)
            if m and int(m.group(1)) in keep:
                continue
            path = os.path.join(OUT, name)
            shutil.rmtree(path, ignore_errors=True) if os.path.isdir(path) else os.remove(path)
    born, dead, _ = parse_timeline()
    todo = [int(args[0])] if args else list(range(11))
    os.makedirs(OUT, exist_ok=True)

    skip = frozen()
    stats = []
    for n in todo:
        if n in skip and os.path.isdir(os.path.join(OUT, f"week-{n:02d}-{SLUG[n]}")):
            print(f"  week-{n:02d}-{SLUG[n]:<20} заморожена, не трогаю")
            files = alive_at(born, dead, n)
            stats.append((f"week-{n:02d}-{SLUG[n]}", len(files), 0, 0))
            continue
        stats.append(build(n, born, dead))
        print(f"  {stats[-1][0]:26} {stats[-1][1]:4} файлов "
              f"({stats[-1][3]} переходных заглушек)")

    if not args:
        open(os.path.join(OUT, "README.md"), "w").write(readme(stats, born, dead))
        print("\n  weeks/README.md — общая картина")
    subprocess.run([sys.executable, "plan/tools/gen-week-docs.py"], check=True)
    print(f"\n  всё в {OUT} (в .gitignore: в коммиты не попадёт)")


def readme(stats, born, dead):
    rows = []
    prev = 0
    meta = {}
    for line in open("plan/timeline.md"):
        m = re.match(r"^## W(\d+) — (.+?) \(([^)]+)\)", line)
        if m:
            meta[int(m.group(1))] = (m.group(2).strip(), m.group(3))
    for n, (name, total, copied, stubs) in enumerate(stats):
        title, dates = meta[n]
        rows.append(f"| [{name}]({name}/) | {title} | {dates} | {total} | {total - prev:+d} |")
        prev = total
    return "\n".join([
        "# Проект по неделям",
        "",
        "Одиннадцать снимков одного проекта: от пустого репозитория до нынешней версии.",
        "Папка собирается скриптом и лежит в `.gitignore` — смотреть, а не коммитить.",
        "",
        "```bash",
        "python3 plan/tools/build-weeks.py          # пересобрать всё",
        "python3 plan/tools/build-weeks.py 5        # только W5",
        "```",
        "",
        "## Что внутри каждой папки",
        "",
        "| | |",
        "| --- | --- |",
        "| `S1.md` … `S5.md` | пошаговая инструкция каждому: ветка, файлы, команды, Jira |",
        "| `captain.md` | что делает капитан: открыть неделю, ревью, закрыть, тег |",
        "| `code/` | состояние проекта на конец недели |",
        "",
        "`node_modules` не копируется — в каждой папке симлинк в корень репозитория,",
        "один на всех. Иначе 11 недель заняли бы ~21 ГБ вместо ~60 МБ.",
        "",
        "## Рост проекта",
        "",
        "| Папка | Неделя | Даты | Файлов | Прибавка |",
        "| --- | --- | --- | --- | --- |",
        *rows,
        "",
        "## Чем это НЕ является",
        "",
        "Содержимое файлов здесь — нынешнее, финальное. Это скелет «какие файлы должны",
        "существовать к концу недели», а не готовый коммит. Если скопировать эти файлы в",
        "коммиты, вернётся ровно та проблема, из-за которой всё затевалось: история из",
        "одних добавлений, где каждая строка сразу финального качества.",
        "",
        "Как писать v1 — в `plan/file-histories.md`. Там лежат исходники первых версий",
        "для файлов, которые позже переписываются.",
        "",
        "## Порядок работы",
        "",
        "1. Капитан открывает `captain.md` очередной недели и делает всё с начала.",
        "2. Каждый открывает свой `S1.md`…`S5.md` и идёт по шагам сверху вниз.",
        "3. Команды оттуда копируются как есть — думать не надо.",
        "4. В конце недели капитан закрывает её по последнему разделу `captain.md`.",
        "",
        "Зоны не пересекаются (`plan/ownership.map`), поэтому шаг 3 пятеро делают",
        "параллельно и не конфликтуют.",
    ]) + "\n"


if __name__ == "__main__":
    main()
