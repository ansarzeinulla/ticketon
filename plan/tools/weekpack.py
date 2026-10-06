#!/usr/bin/env python3
"""Разложить готовое дерево недели по задачам и написать скрипты коммита.

    weekpack.py 02 --source /path/to/tree [--overrides DIR]

Читает plan/weekNN/manifest.txt:

    # комментарий
    BF-14 S1 week-02 "BF-14 Hash passwords with bcrypt"
        api/internal/auth/password.go
        -api/internal/config/env.go

Комментарии — только целой строкой, с «#» в начале. «-путь» — файл удаляется
в этой задаче.

База — ветка недели или номер задачи, поверх ветки которой строится эта.

Файл берётся из --overrides/<задача>/<путь>, если он там есть (промежуточная
версия файла, которую позже перепишет другая задача той же недели), иначе из
--source. Результат:

    plan/weekNN/code/<задача>/...     файлы задачи
    plan/weekNN/code/<задача>.delete  что задача удаляет
    plan/weekNN/S<N>-<задача>.sh      запускает владелец задачи
    plan/weekNN/_commit.sh            общая часть
    plan/weekNN/tickets.tsv           то же, для проверок и README
"""

import argparse
import os
import shlex
import shutil
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))  # plan/
NICK = {"S1": "ansar", "S2": "alibi", "S3": "abylay", "S4": "alinur", "S5": "olzhas"}
NAME = {"S1": "Ansar Zeinulla", "S2": "Alibi Takhtanov", "S3": "Abylay Otaubay",
        "S4": "Alinur Burlybayev", "S5": "Olzhas Nurseit"}


def parse(path):
    tickets = []
    for lineno, raw in enumerate(open(path, encoding="utf-8"), 1):
        line = raw.rstrip()
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if not raw[0].isspace():
            parts = shlex.split(line)
            if len(parts) != 4:
                sys.exit(f"{path}:{lineno}: ожидалось <задача> <SN> <база> \"<сообщение>\"")
            ticket, who, base, message = parts
            if who not in NICK:
                sys.exit(f"{path}:{lineno}: неизвестный студент {who}")
            tickets.append({"ticket": ticket, "who": who, "base": base,
                            "message": message, "add": [], "delete": []})
        else:
            entry = line.strip()
            if not tickets:
                sys.exit(f"{path}:{lineno}: файл до первой задачи")
            if entry.startswith("-"):
                tickets[-1]["delete"].append(entry[1:].strip())
            else:
                tickets[-1]["add"].append(entry)
    return tickets


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("week", type=int)
    ap.add_argument("--source", required=True)
    ap.add_argument("--overrides")
    args = ap.parse_args()

    week = f"{args.week:02d}"
    week_branch = f"week-{week}"
    wdir = os.path.join(ROOT, f"week{week}")
    tickets = parse(os.path.join(wdir, "manifest.txt"))
    by_id = {t["ticket"]: t for t in tickets}

    code = os.path.join(wdir, "code")
    if os.path.isdir(code):
        shutil.rmtree(code)
    os.makedirs(code)
    for name in os.listdir(wdir):
        if name.startswith("S") and name.endswith(".sh"):
            os.remove(os.path.join(wdir, name))

    template = open(os.path.join(ROOT, "tools", "commit-template.sh"), encoding="utf-8").read()
    commit = template.replace("__WEEK_BRANCH__", week_branch).replace("__WEEK__", week)
    with open(os.path.join(wdir, "_commit.sh"), "w", encoding="utf-8") as f:
        f.write(commit)
    os.chmod(os.path.join(wdir, "_commit.sh"), 0o755)

    rows = []
    problems = 0
    for t in tickets:
        t["branch"] = f"{week_branch}-{NICK[t['who']]}-{t['ticket']}"
    for t in tickets:
        if t["base"] == week_branch:
            base_ref = f"origin/{week_branch}"
        elif t["base"] in by_id:
            base_ref = f"origin/{by_id[t['base']]['branch']}"
        else:
            sys.exit(f"{t['ticket']}: неизвестная база {t['base']}")

        dest = os.path.join(code, t["ticket"])
        for rel in t["add"]:
            src = None
            if args.overrides:
                cand = os.path.join(args.overrides, t["ticket"], rel)
                if os.path.exists(cand):
                    src = cand
            if src is None:
                src = os.path.join(args.source, rel)
            if not os.path.exists(src):
                print(f"!! {t['ticket']}: нет файла {rel}")
                problems += 1
                continue
            out = os.path.join(dest, rel)
            if os.path.isdir(src):
                shutil.copytree(src, out, dirs_exist_ok=True)
            else:
                os.makedirs(os.path.dirname(out), exist_ok=True)
                shutil.copy2(src, out)
        if t["delete"]:
            with open(os.path.join(code, t["ticket"] + ".delete"), "w", encoding="utf-8") as f:
                f.write("# пути, которые задача удаляет (или старые имена переименованных)\n")
                f.write("\n".join(t["delete"]) + "\n")

        script = os.path.join(wdir, f"{t['who']}-{t['ticket']}.sh")
        with open(script, "w", encoding="utf-8") as f:
            f.write("#!/usr/bin/env bash\n")
            f.write(f"# {t['message']}\n")
            f.write(f"# Запускает {NAME[t['who']]} ({t['who']}) сам, с этого компьютера, из любой папки.\n")
            if base_ref != f"origin/{week_branch}":
                dep = by_id[t["base"]]
                f.write(f"# Строится поверх {dep['ticket']} ({NAME[dep['who']]}): сначала должен быть запущен его скрипт.\n")
            f.write('exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" '
                    + " ".join(shlex.quote(x) for x in
                               [t["ticket"], t["who"], t["branch"], base_ref, t["message"]])
                    + "\n")
        os.chmod(script, 0o755)
        rows.append([t["ticket"], t["who"], t["branch"], base_ref, t["message"],
                     str(len(t["add"])), str(len(t["delete"]))])

    with open(os.path.join(wdir, "tickets.tsv"), "w", encoding="utf-8") as f:
        f.write("ticket\twho\tbranch\tbase\tmessage\tadd\tdelete\n")
        for r in rows:
            f.write("\t".join(r) + "\n")

    for r in rows:
        print(f"{r[0]:7} {r[1]}  {r[2]:28} <- {r[3]:34} +{r[5]:>3} -{r[6]}")
    if problems:
        sys.exit(f"{problems} файлов не найдено")


if __name__ == "__main__":
    main()
