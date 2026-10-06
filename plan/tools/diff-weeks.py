"""Сравнивает соседние недели и ищет невозможные переходы.

    python3 plan/tools/diff-weeks.py            # сводка по всем переходам
    python3 plan/tools/diff-weeks.py 5          # подробно W4 → W5
    python3 plan/tools/diff-weeks.py --check    # только ошибки, код возврата

Сравнение снимков `weeks/` было бы замкнутым кругом: они собраны из
`timeline.md`, поэтому совпали бы всегда. Здесь проверяется сам план — то,
что генератор не контролировал:

  * файл изменяют раньше, чем создают;
  * файл трогают после удаления;
  * файл создают дважды или удаляют дважды;
  * удаляют то, чего не создавали;
  * переименовывают файл, которого нет, или трогают старое имя после `git mv`.

Внутри одной недели пятеро работают параллельно в своих ветках, поэтому порядок
строк между разными владельцами ничего не значит. Строгий порядок требуется
только внутри работы одного человека.
"""
import re
import subprocess
import sys
import fnmatch
import collections

# `weeks/` — снимки, порождённые из плана. Для инструментов это не код
# проекта: иначе они начнут планировать сами себя.
FILES = {f for f in subprocess.check_output(["git", "ls-files"], text=True).split()
         if not f.startswith("weeks/")}
OWNERS = ["S1", "S2", "S3", "S4", "S5"]


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


def glob_files(ref):
    pat = ref.replace("**", "*")
    return sorted(f for f in FILES
                  if fnmatch.fnmatchcase(f, pat) or fnmatch.fnmatchcase(f, pat + "*"))


def read_timeline():
    """Список операций в порядке файла: (неделя, №строки, владелец, тикет, коммит, op, файл)."""
    ops, phase, idx = [], None, 0
    titles = {}
    for line in open("plan/timeline.md"):
        m = re.match(r"^## W(\d+) — (.+?)(?: →|$)", line)
        if m:
            phase = int(m.group(1))
            titles[phase] = m.group(2).strip()
            continue
        r = re.match(r"^\| (S[1-5]) \| `(BF-B?\d+) ([^`]+)` \| (.*?)\s*\|\s*$", line)
        if not r or phase is None:
            continue
        idx += 1
        who, tid, commit, body = r.groups()
        for src, dst in re.findall(r"\*\*R\*\* `([^`]+)` → `([^`]+)`", body):
            ops.append((phase, idx, who, tid, commit, "R", resolve(src), resolve(dst)))
        body_no_rename = re.sub(r"\*\*R\*\* `[^`]+` → `[^`]+`", "", body)
        for op, ref in re.findall(r"\*\*([AMD])\*\* `([^`]+)`", body_no_rename):
            for one in expand(ref):
                if "*" in one:
                    for f in glob_files(one):
                        ops.append((phase, idx, who, tid, commit, op, f, None))
                else:
                    ops.append((phase, idx, who, tid, commit, op, resolve(one), None))
    return ops, titles


def analyse(ops):
    """Проигрывает план по порядку и записывает всё, что физически невозможно."""
    born, dead, problems = {}, {}, []

    def say(o, text):
        problems.append((o[0], f"W{o[0]} · {o[2]} · `{o[3]} {o[4]}` — {text}"))

    for o in ops:
        phase, idx, who, tid, commit, op, f, dst = o

        if op == "A":
            if f in born and f not in dead:
                b = born[f]
                say(o, f"создаёт `{f}`, который уже создан в W{b[0]} ({b[2]})")
            born[f] = (phase, idx, who)
            dead.pop(f, None)

        elif op == "M":
            if f not in born:
                say(o, f"изменяет `{f}`, который нигде не создан")
            else:
                b = born[f]
                if b[0] > phase:
                    say(o, f"изменяет `{f}` раньше, чем тот создан (W{b[0]})")
                elif b[0] == phase and b[2] == who and b[1] > idx:
                    say(o, f"изменяет `{f}` выше по неделе, чем сам же его создаёт")
            if f in dead:
                say(o, f"изменяет `{f}`, удалённый в W{dead[f][0]}")

        elif op == "D":
            if f not in born:
                say(o, f"удаляет `{f}`, который нигде не создан")
            elif f in dead:
                say(o, f"удаляет `{f}` повторно — уже удалён в W{dead[f][0]}")
            else:
                b = born[f]
                if b[0] > phase:
                    say(o, f"удаляет `{f}` раньше, чем тот создан (W{b[0]})")
            dead[f] = (phase, idx, who)

        elif op == "R":
            if f not in born:
                say(o, f"переименовывает `{f}`, который нигде не создан")
            elif f in dead:
                say(o, f"переименовывает `{f}`, удалённый в W{dead[f][0]}")
            if dst in born and dst not in dead:
                say(o, f"переименовывает в `{dst}`, который уже существует")
            dead[f] = (phase, idx, who)
            born[dst] = (phase, idx, who)
    return born, dead, problems


def snapshot(born, dead, n):
    return {f for f, b in born.items() if b[0] <= n and dead.get(f, (99,))[0] > n}


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    only_check = "--check" in sys.argv
    ops, titles = read_timeline()
    born, dead, problems = analyse(ops)

    if args:
        n = int(args[0])
        prev, cur = snapshot(born, dead, n - 1), snapshot(born, dead, n)
        print(f"\nW{n - 1} → W{n}  ({titles.get(n, '')})\n")
        renames = {(f, d) for p, i, w, t, c, op, f, d in ops if op == "R" and p == n}
        for f, d in sorted(renames):
            print(f"  R  {f}  →  {d}")
        for f in sorted(cur - prev):
            if any(d == f for _, d in renames):
                continue
            print(f"  A  {f}")
        for f in sorted(prev - cur):
            if any(s == f for s, _ in renames):
                continue
            print(f"  D  {f}")
        touched = {f for p, i, w, t, c, op, f, d in ops if p == n and op == "M"}
        print(f"\n  {len(cur - prev)} создано · {len(prev - cur)} удалено · "
              f"{len(touched)} изменено · всего файлов {len(cur)}")
        return 0

    if not only_check:
        print("\n  переход      создано  удалено  изменено   всего")
        for n in range(11):
            prev = snapshot(born, dead, n - 1) if n else set()
            cur = snapshot(born, dead, n)
            touched = {f for p, i, w, t, c, op, f, d in ops if p == n and op == "M"}
            print(f"  W{n - 1 if n else '–'}→W{n:<2}     {len(cur - prev):6}   "
                  f"{len(prev - cur):6}   {len(touched):7}   {len(cur):5}")

    print()
    if problems:
        print(f"  {len(problems)} невозможных переходов:\n")
        for _, text in sorted(problems):
            print("   ", text)
    else:
        print("  невозможных переходов нет: ничего не меняют до создания,")
        print("  не трогают после удаления, не создают и не удаляют дважды")
    print()
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
