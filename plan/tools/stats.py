"""Плановый объём работы: сколько чего делает каждый в каждой неделе.

    python3 plan/tools/stats.py            # таблица по неделям
    python3 plan/tools/stats.py --people   # итог по людям
    python3 plan/tools/stats.py --csv      # то же в CSV

Колонки: новых файлов · изменённых · удалённых · переименований ·
добавлено строк · удалено строк.

Откуда берутся строки. Размер нового файла — **настоящий**, из нынешнего
репозитория (`wc -l`). Но файл рождается меньше, чем он сейчас: если к нему
потом возвращаются k раз, при создании кладётся примерно L/(1+0,5k) строк, а
остальное дописывается правками. Где в `timeline.md` объём указан явно
(`-38/+14`), берётся он. Всё прочее — модель: правка дописывает свою долю и
переписывает часть уже написанного.

Это **план**, а не замер: столько работы предстоит сделать. Фактические числа
после семестра покажет `check-history.sh`.
"""
import collections
import os
import re
import subprocess
import sys
import fnmatch

# `weeks/` — снимки, порождённые из плана. Для инструментов это не код
# проекта: иначе они начнут планировать сами себя.
FILES = {f for f in subprocess.check_output(["git", "ls-files"], text=True).split()
         if not f.startswith("weeks/")}
OWNERS = ["S1", "S2", "S3", "S4", "S5"]
TRANSIENT_LINES = 45          # файлы, которых нет в репозитории: жили часть семестра
CHURN = 0.30                  # доля переписываемого при обычной правке


def lines_of(path):
    if path not in FILES:
        return TRANSIENT_LINES
    try:
        with open(path, "rb") as fh:
            return max(1, fh.read().count(b"\n"))
    except OSError:
        return TRANSIENT_LINES


def expand(ref):
    m = re.search(r"\{([^}]*)\}", ref)
    if not m:
        return [ref]
    return [ref[:m.start()] + a.strip() + ref[m.end():] for a in m.group(1).split(",")]


def resolve(ref):
    if ref in FILES:
        return ref
    cand = [f for f in FILES if f.endswith("/" + ref)]
    return cand[0] if len(cand) == 1 else ref


def read_rows():
    rows, phase, titles = [], None, {}
    for line in open("plan/timeline.md"):
        m = re.match(r"^## (W\d+) — (.+?)(?: →|$)", line)
        if m:
            phase, titles[m.group(1)] = m.group(1), m.group(2).strip()
            continue
        r = re.match(r"^\| (S[1-5]) \| `(BF-B?\d+) ([^`]+)` \| (.*?)\s*\|\s*$", line)
        if r and phase:
            rows.append((phase, r.group(1), r.group(2), r.group(3), r.group(4)))
    return rows, titles


def ops_of(body):
    """[(op, файл, явный (-x/+y) или None)]"""
    out = []
    for m in re.finditer(r"\*\*R\*\* `([^`]+)` → `([^`]+)`", body):
        out.append(("R", resolve(m.group(2)), None))
    body = re.sub(r"\*\*R\*\* `[^`]+` → `[^`]+`", "", body)
    for m in re.finditer(r"\*\*([AMD])\*\* `([^`]+)`((?:\s*\*\([^)]*\)\*)?)", body):
        op, ref, note = m.group(1), m.group(2), m.group(3) or ""
        explicit = re.search(r"`?-(\d+)/\+(\d+)`?", note)
        pair = (int(explicit.group(1)), int(explicit.group(2))) if explicit else None
        for one in expand(ref):
            if "*" in one:
                pat = one.replace("**", "*")
                for f in sorted(FILES):
                    if fnmatch.fnmatchcase(f, pat) or fnmatch.fnmatchcase(f, pat + "*"):
                        out.append((op, f, pair))
            else:
                out.append((op, resolve(one), pair))
    return out


def build(detail=False):
    """Сводка по (неделя, человек). При detail=True вторым значением идёт
    список записей на каждый коммит."""
    rows, titles = read_rows()
    touches = collections.Counter()          # сколько правок придётся на файл
    for _, _, _, _, body in rows:
        for op, f, _ in ops_of(body):
            if op == "M":
                touches[f] += 1

    born_lines, commits = {}, []
    stat = collections.defaultdict(lambda: collections.Counter())
    for phase, who, tid, title, body in rows:
        cell = stat[(phase, who)]
        cell["commits"] += 1
        rec = collections.Counter()
        files = {"new": [], "changed": [], "gone": [], "renamed": []}
        for op, f, pair in ops_of(body):
            L = lines_of(f)
            if op == "A":
                k = touches[f]
                first = max(10, round(L / (1 + 0.5 * k))) if k else L
                born_lines[f] = first
                rec["new"] += 1; rec["added"] += first
                rec["appended"] += first      # весь новый файл — дописанные строки
                files["new"].append(f)
            elif op == "M":
                if pair:
                    d, a = pair
                else:
                    k = max(1, touches[f])
                    a = max(2, round((L - born_lines.get(f, L)) / k))
                    d = max(1, round(a * CHURN))
                # Строку, которую переписали, git показывает как «−» и «+».
                # Развожу их: сколько дописано, сколько вырезано, сколько переписано.
                rec["rewritten"] += min(a, d)
                rec["appended"] += a - min(a, d)
                rec["removed"] += d - min(a, d)
                rec["added"] += a; rec["deleted"] += d
                rec["changed"] += 1
                files["changed"].append(f)
            elif op == "D":
                gone = born_lines.get(f, lines_of(f))
                rec["deleted"] += gone; rec["removed"] += gone
                rec["gone"] += 1
                files["gone"].append(f)
            elif op == "R":
                rec["renamed"] += 1
                files["renamed"].append(f)
        cell.update(rec)
        if detail:
            commits.append({"week": phase, "who": who, "tid": tid, "title": title,
                            "rec": rec, "files": files})
    return (stat, titles, commits) if detail else (stat, titles)


NAME = {"S1": "Ansar Zeinulla", "S2": "Alibi Takhtanov", "S3": "Abylay Otaubay",
        "S4": "Alinur Burlybayev", "S5": "Olzhas Nurseit"}
DATES = {}          # заполняется в write_csv


def week_dates():
    out = {}
    for line in open("plan/timeline.md"):
        m = re.match(r"^## (W\d+) — (.+?) \(([^)]+)\)", line)
        if m:
            out[m.group(1)] = (m.group(2).strip(), m.group(3))
    return out


def write_csv():
    """Две таблицы: сводка по неделям и построчная по каждому коммиту."""
    import csv
    stat, _titles, commits = build(detail=True)
    meta = week_dates()
    weeks = sorted({p for p, _ in stat}, key=lambda x: int(x[1:]))

    p1 = "plan/stats-by-week.csv"
    with open(p1, "w", newline="", encoding="utf-8") as fh:
        w = csv.writer(fh)
        w.writerow(["week", "week_title", "week_dates", "student", "name",
                    "commits", "files_new", "files_changed", "files_deleted",
                    "files_renamed", "files_touched",
                    "lines_appended", "lines_rewritten", "lines_removed",
                    "lines_added_total", "lines_deleted_total", "lines_net"])
        for wk in weeks:
            title, dates = meta.get(wk, ("", ""))
            for s in OWNERS:
                c = stat.get((wk, s), collections.Counter())
                touched = c["new"] + c["changed"] + c["gone"] + c["renamed"]
                w.writerow([wk, title, dates, s, NAME[s],
                            c["commits"], c["new"], c["changed"], c["gone"],
                            c["renamed"], touched,
                            c["appended"], c["rewritten"], c["removed"],
                            c["added"], c["deleted"], c["added"] - c["deleted"]])

    p2 = "plan/stats-by-commit.csv"
    with open(p2, "w", newline="", encoding="utf-8") as fh:
        w = csv.writer(fh)
        w.writerow(["week", "week_dates", "student", "name", "ticket",
                    "commit_message", "kind",
                    "files_new", "files_changed", "files_deleted", "files_renamed",
                    "lines_appended", "lines_rewritten", "lines_removed",
                    "lines_added_total", "lines_deleted_total",
                    "new_files_list", "changed_files_list",
                    "deleted_files_list", "renamed_files_list"])
        for rec in commits:
            c, f = rec["rec"], rec["files"]
            _t, dates = meta.get(rec["week"], ("", ""))
            w.writerow([rec["week"], dates, rec["who"], NAME[rec["who"]], rec["tid"],
                        f'{rec["tid"]} {rec["title"]}',
                        "defect" if rec["tid"].startswith("BF-B") else "story",
                        c["new"], c["changed"], c["gone"], c["renamed"],
                        c["appended"], c["rewritten"], c["removed"],
                        c["added"], c["deleted"],
                        " ".join(f["new"]), " ".join(f["changed"]),
                        " ".join(f["gone"]), " ".join(f["renamed"])])

    print(f"  {p1}    {len(weeks) * len(OWNERS)} строк (неделя × человек)")
    print(f"  {p2}  {len(commits)} строк (по коммиту)")


HEAD = ("кто", "комм", "новых", "изм", "удал", "переим", "+строк", "−строк")


def row_fmt(who, c):
    return (f"  {who:4} {c['commits']:5} {c['new']:7} {c['changed']:6} "
            f"{c['gone']:6} {c['renamed']:7} {c['added']:8} {c['deleted']:8}")


def main():
    stat, titles = build()
    phases = sorted({p for p, _ in stat}, key=lambda x: int(x[1:]))

    if "--csv" in sys.argv:
        write_csv()
        return

    if "--people" in sys.argv:
        print("\n  Итог за семестр\n")
        print("  " + f"{HEAD[0]:4} {HEAD[1]:>5} {HEAD[2]:>7} {HEAD[3]:>6} "
                     f"{HEAD[4]:>6} {HEAD[5]:>7} {HEAD[6]:>8} {HEAD[7]:>8}")
        tot = collections.Counter()
        for w in OWNERS:
            c = collections.Counter()
            for p in phases:
                c.update(stat.get((p, w), collections.Counter()))
            tot.update(c)
            print(row_fmt(w, c))
        print("  " + "-" * 60)
        print(row_fmt("все", tot))
        bulk = sum(lines_of(f) for f in FILES
                   if f.endswith(("package-lock.json", ".ttf", ".png", ".ico"))
                   or f.endswith("postman_collection.json"))
        print(f"\n  из +{tot['added']} строк примерно {bulk} — это lock-файлы, шрифты,")
        print("  иконки и Postman-коллекция: они кладутся целиком, их не пишут руками.")
        print(f"  Рукописного кода остаётся около {tot['added'] - bulk} строк.\n")
        return

    for p in phases:
        print(f"\n  {p} — {titles.get(p, '')}")
        print("  " + f"{HEAD[0]:4} {HEAD[1]:>5} {HEAD[2]:>7} {HEAD[3]:>6} "
                     f"{HEAD[4]:>6} {HEAD[5]:>7} {HEAD[6]:>8} {HEAD[7]:>8}")
        tot = collections.Counter()
        for w in OWNERS:
            c = stat.get((p, w), collections.Counter())
            tot.update(c)
            print(row_fmt(w, c))
        print(row_fmt("итог", tot))
    print()


if __name__ == "__main__":
    main()
