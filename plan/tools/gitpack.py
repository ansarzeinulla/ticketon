#!/usr/bin/env python3
"""Build plan/weekNN/manifest.txt and per-ticket file versions from a work repo.

    gitpack.py <NN> <work-repo> <overrides-dir> <plan-file>

The work repo has one commit per ticket, its subject being the ticket id
("BF-37"), committed in the order the tickets are built. The plan file names,
per ticket in run order, who does it, what it is based on and the commit
message:

    BF-37 S1 week-04 "BF-37 Add organizer profiles and money handling"

For every ticket the files it adds or changes are taken from the diff between
its commit and the previous one, and written to <overrides-dir>/<ticket>/ at
the version they have in that commit; deletions and the old side of renames
become "-path" lines. The manifest and the overrides are then fed to weekpack.py.
"""
import os
import shlex
import subprocess
import sys


def git(repo, *args, binary=False):
    out = subprocess.run(["git", "-C", repo, *args], check=True, capture_output=True)
    return out.stdout if binary else out.stdout.decode()


def main():
    week, repo, overrides, plan_file = sys.argv[1:5]
    week = f"{int(week):02d}"
    plan = []
    for raw in open(plan_file, encoding="utf-8"):
        line = raw.strip()
        if line and not line.startswith("#"):
            plan.append(shlex.split(line))

    commits = {}
    for line in git(repo, "log", "--format=%H %s").splitlines():
        sha, subject = line.split(" ", 1)
        commits.setdefault(subject.strip(), sha)

    header = [f"# Неделя {int(week)} — собрано tools/gitpack.py из рабочего репозитория.",
              "# <задача> <студент> <база> \"<сообщение коммита>\", затем файлы задачи;",
              "# «-путь» — файл удаляется.", ""]
    body = []
    for ticket, who, base, message in plan:
        sha = commits[ticket]
        parent = git(repo, "rev-parse", sha + "^").strip()
        body.append(f'{ticket} {who} {base} "{message}"')
        for row in git(repo, "diff", "--name-status", "--no-renames", parent, sha).splitlines():
            status, path = row.split("\t", 1)
            if status == "D":
                body.append(f"    -{path}")
                continue
            body.append(f"    {path}")
            data = git(repo, "show", f"{sha}:{path}", binary=True)
            out = os.path.join(overrides, ticket, path)
            os.makedirs(os.path.dirname(out), exist_ok=True)
            with open(out, "wb") as f:
                f.write(data)
        body.append("")

    plan_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), f"week{week}")
    os.makedirs(plan_dir, exist_ok=True)
    with open(os.path.join(plan_dir, "manifest.txt"), "w", encoding="utf-8") as f:
        f.write("\n".join(header + body))
    print(f"manifest: {len(plan)} tickets")


if __name__ == "__main__":
    main()
