"""Пишет пошаговые инструкции в папку каждой недели.

    python3 plan/tools/gen-week-docs.py

Для каждой недели создаёт шесть файлов по одному шаблону:

    weeks/week-05-paid-sales/S1.md … S5.md   что делает каждый, по дням
    weeks/week-05-paid-sales/captain.md      что делает капитан

Внутри — готовые команды: какую ветку создать, какие файлы тронуть, что
закоммитить, куда открыть PR и что передвинуть в Jira. Копируется и выполняется
без размышлений.

Всё выводится из plan/timeline.md, поэтому разъехаться с планом не может.
"""
import collections
import datetime
import os
import re
import shlex
import subprocess

# `weeks/` — снимки, порождённые из плана. Для инструментов это не код
# проекта: иначе они начнут планировать сами себя.
FILES = {f for f in subprocess.check_output(["git", "ls-files"], text=True).split()
         if not f.startswith("weeks/")}


def full(path):
    """`store/checkout.go` -> `api/internal/store/checkout.go`, иначе команду
    нельзя скопировать."""
    if path in FILES:
        return path
    cand = [f for f in FILES if f.endswith("/" + path)]
    return cand[0] if len(cand) == 1 else path

MONTH = {"января": 1, "февраля": 2, "марта": 3, "апреля": 4, "мая": 5, "июня": 6,
         "июля": 7, "августа": 8, "сентября": 9, "октября": 10, "ноября": 11,
         "декабря": 12}
SHORT = {9: "сентября", 10: "октября", 11: "ноября"}
DOW = ["понедельник", "вторник", "среда", "четверг", "пятница", "суббота", "воскресенье"]
OWNERS = ["S1", "S2", "S3", "S4", "S5"]
SLUG = {0: "discovery", 1: "skeleton", 2: "identity", 3: "events",
        4: "paid-tickets", 5: "gate", 6: "refunds", 7: "analytics",
        8: "release-shape", 9: "verification", 10: "release"}
# Ревью делает только капитан: он смотрит и вливает все PR, включая свои.
CAPTAIN = "S1"
# Короткие имена для веток — из plan/students.md
NICK = {"S1": "ansar", "S2": "alibi", "S3": "abylay",
        "S4": "alinur", "S5": "olzhas"}
ZONE = {"S1": "бэкенд-ядро, база данных, интеграция",
        "S2": "бэкенд коммерции: события, билеты, checkout, возвраты",
        "S3": "веб для зрителя, типы и клиент API, локализация",
        "S4": "веб организатора и админа, аналитика, поддержка",
        "S5": "мобильное приложение, QA, DevOps"}
OP = {"A": "создать", "M": "изменить", "D": "удалить", "R": "переименовать"}
EPIC = [(r"specification|Postman|Run the phase|record results|Correct the spec", "BF-E12"),
        (r"[Ll]ocalis|dictionary|language|locale", "BF-E11"),
        (r"support|inbox", "BF-E8"),
        (r"admin portal|report queue|moderation|admin search", "BF-E10"),
        (r"analytics|GA4|fees|payout|reserved|money card", "BF-E9"),
        (r"refund|cancel|reversal|my orders|own orders", "BF-E7"),
        (r"scanner|check-in|gate|admit|offline|staff", "BF-E6"),
        (r"QR|PDF|font|Cyrillic|ticket status|confirmation", "BF-E5"),
        (r"checkout|hold|fee|payment|activation|seat|oversell|inventory", "BF-E4"),
        (r"password|JWT|session|role|permission|register|login|reset", "BF-E2"),
        (r"catalogue|event page|order page|Next\.js|proxy|selector", "BF-E11"),
        (r"event|slug|ticket type|dashboard|create form", "BF-E3")]


def frozen():
    """Номера фаз, которые уже закрыты и переписыванию не подлежат."""
    out = set()
    try:
        for line in open("plan/frozen.txt"):
            line = line.split("#")[0].strip()
            if line.isdigit():
                out.add(int(line))
    except FileNotFoundError:
        pass
    return out


def epic_of(title):
    for pat, e in EPIC:
        if re.search(pat, title):
            return e
    return "BF-E1"


ACC = {"среда": "среду", "пятница": "пятницу", "суббота": "субботу"}


def day_acc(x):
    """«в пятницу 11 сентября», а не «в пятница»."""
    t = day_ru(x)
    head = t.split(" ")[0]
    return t.replace(head, ACC.get(head, head), 1)


def plural(n, one, few, many):
    if n % 10 == 1 and n % 100 != 11:
        return one
    if n % 10 in (2, 3, 4) and n % 100 not in (12, 13, 14):
        return few
    return many


def day_ru(x):
    return f"{DOW[x.weekday()]} {x.day} {SHORT[x.month]}"


def parse_span(span):
    a, b = re.split(r"\s*[–-]\s*", span)[:2]
    mb = re.search(r"([а-я]+)$", b.strip())
    m2, d2 = MONTH[mb.group(1)], int(re.match(r"(\d+)", b.strip()).group(1))
    ma = re.search(r"([а-я]+)$", a.strip())
    m1 = MONTH[ma.group(1)] if ma else m2
    d1 = int(re.match(r"(\d+)", a.strip()).group(1))
    return datetime.date(2026, m1, d1), datetime.date(2026, m2, d2)


def expand(ref):
    m = re.search(r"\{([^}]*)\}", ref)
    if not m:
        return [ref]
    return [ref[:m.start()] + a.strip() + ref[m.end():] for a in m.group(1).split(",")]


def files_of(body):
    """[(операция, путь, примечание)] в порядке появления."""
    out = []
    for m in re.finditer(r"\*\*R\*\* `([^`]+)` → `([^`]+)`((?:\s*\*\([^)]*\)\*)?)", body):
        out.append(("R", (full(m.group(1)), full(m.group(2))), clean(m.group(3))))
    rest = re.sub(r"\*\*R\*\* `[^`]+` → `[^`]+`", "", body)
    for m in re.finditer(r"\*\*([AMD])\*\* `([^`]+)`((?:\s*\*\([^)]*\)\*)?)", rest):
        note = clean(m.group(3))
        for one in expand(m.group(2)):
            out.append((m.group(1), full(one), note))
    return out


def clean(note):
    return re.sub(r"^\*\(|\)\*$", "", (note or "").strip()).strip()


# Makefile появляется в W1 (BF-12), а цели проверок — в BF-13. До конца W1
# на него рассчитывать нельзя, поэтому там даём прямые команды.
MAKE_FROM = 2
RAW = {"db": "bash db/tests/run_tests.sh",
       "api": "cd api && gofmt -l . && go vet ./... && go test ./...",
       "web": "cd web && npm run lint && npx tsc --noEmit",
       "mobile": "cd mobile && npx tsc --noEmit"}
MAKE = {"db": "make test          # тесты базы",
        "api": "make api-check     # формат, vet и тесты Go",
        "web": "make web-check     # линт, типы и тесты фронта",
        "mobile": "make scan-check    # типы мобильного приложения"}


def check_cmd(paths, phase_n):
    code = [p for p in paths if not p.endswith((".md", ".txt"))]
    hit = [k for k in ("db", "api", "web", "mobile")
           if any(p.startswith(k + "/") for p in code)]
    table = MAKE if phase_n >= MAKE_FROM else RAW
    return [table[k] for k in hit]


def lane(who, body):
    if "02_schema.sql" in body:
        return 1
    if who == "S1" and "server.go" in body:
        return 3
    if who in ("S2", "S4", "S5"):
        return 2
    if who == "S3" and ("api.ts" in body or "types.ts" in body):
        return 4
    return 5


def read():
    out, cur = [], None
    for line in open("plan/timeline.md"):
        m = re.match(r"^## (W(\d+)) — (.+?) \(([^)]+)\)", line)
        if m:
            cur = {"key": m.group(1), "n": int(m.group(2)), "title": m.group(3).strip(),
                   "span": parse_span(m.group(4)), "rows": []}
            out.append(cur)
            continue
        r = re.match(r"^\| (S[1-5]) \| `(BF-B?\d+) ([^`]+)` \| (.*?)\s*\|\s*$", line)
        if r and cur:
            cur["rows"].append({"who": r.group(1), "tid": r.group(2), "title": r.group(3),
                                "body": r.group(4), "lane": lane(r.group(1), r.group(4)),
                                "files": files_of(r.group(4))})
    return out


MANIFEST = {"web": "web/package.json", "api": "api/go.mod", "mobile": "mobile/package.json"}
SCAFFOLD = {
    "web/package.json": ("npx create-next-app@16 web --ts --eslint --tailwind --app "
                         "--src-dir --import-alias \"@/*\" --use-npm --yes"),
    "mobile/package.json": "npx create-expo-app@latest mobile --template blank-typescript",
}


def link_deps(phases):
    """Задача, чьи файлы живут в web/, api/ или mobile/, не соберётся, пока не
    влит манифест этой папки. Если манифест создаёт другой человек в ту же
    неделю — это зависимость: ветку надо растить от его ветки."""
    for ph in phases:
        creators = {}
        for r in ph["rows"]:
            for op, path, _ in r["files"]:
                if op == "A" and path in MANIFEST.values():
                    creators[path] = r
        for r in ph["rows"]:
            r["deps"] = []
            # Текстовым правкам сборка не нужна — зависимость только у кода.
            paths = [p for op, p, _ in r["files"]
                     if op != "R" and not p.endswith((".md", ".txt"))]
            for top, man in MANIFEST.items():
                c = creators.get(man)
                if c and c is not r and any(x.startswith(top + "/") for x in paths):
                    if c not in r["deps"]:
                        r["deps"].append(c)
            if any(op == "A" and path in MANIFEST.values() for op, path, _ in r["files"]):
                r["lane"] = 0          # манифест — самым первым


def assign_days(phase):
    start, end = phase["span"]
    length = (end - start).days + 1
    work = max(1, length - 2) if length >= 5 else max(1, length - 1)

    def num(t):
        m = re.match(r"BF-(B?)(\d+)", t)
        return (1 if m.group(1) else 0, int(m.group(2)))

    rows = sorted(phase["rows"], key=lambda r: (r["lane"], num(r["tid"]), r["who"]))
    quota = [len(rows) // work + (1 if i < len(rows) % work else 0) for i in range(work)]
    used, plan, day, placed = collections.defaultdict(set), [], 0, 0
    for r in rows:
        while day < work - 1 and placed >= quota[day]:
            day, placed = day + 1, 0
        d0 = day
        while d0 < work - 1 and (d0 + 1) in used[r["who"]]:
            d0 += 1
        if (d0 + 1) in used[r["who"]]:
            free = [x for x in range(1, work + 1) if x not in used[r["who"]]]
            if free:
                d0 = min(free, key=lambda x: abs(x - (d0 + 1))) - 1
        used[r["who"]].add(d0 + 1)
        plan.append((d0 + 1, r))
        placed += 1
    rank = {x: i + 1 for i, x in enumerate(sorted({x for x, _ in plan}))}
    return [(rank[x], r) for x, r in plan], work, length


# Ветка недели. W1 начали до перехода на недели и оставили как phase-01.
def week_branch(n):
    return "phase-01" if n == 1 else f"week-{n:02d}"


def branch(phase, tid, who):
    # Через дефис, а не слэш: пока существует ветка `week-01`, git отказывается
    # создавать `week-01/что-угодно` — «refs/heads/week-01 exists».
    return f"{week_branch(phase['n'])}-{NICK[who]}-{tid}"



# Имя и почта для `git config` — те, с которыми человек уже коммитит на GitHub.
# Каждый выставляет своё у себя на компьютере: иначе коммит не свяжется с профилем.
GIT_ID = {"S1": ("Ansar Zeinulla", "ansar.zeinulla.a@gmail.com"),
          "S2": ("Alibi Takhtanov", "takhtanovalb@gmail.com"),
          "S3": ("Abylay Otaubay", "o.abyl247@gmail.com"),
          "S4": ("Alinur Burlybayev", None),
          "S5": ("Olzhas Nurseit", "olzhasnrseit@gmail.com")}
FULL = {s: n for s, (n, _) in GIT_ID.items()}


def pr_day(phase):
    """Последний рабочий день недели: в этот вечер все открывают PR."""
    start, end = phase["span"]
    length = (end - start).days + 1
    work = max(1, length - 2) if length >= 5 else max(1, length - 1)
    return start + datetime.timedelta(days=work - 1)


def ticket_block(phase, day_date, r, idx, total):
    ph = week_branch(phase["n"])
    br = branch(phase, r["tid"], r["who"])
    paths = [p for op, p, _ in r["files"] if op != "R"]
    renames = [p for op, p, _ in r["files"] if op == "R"]
    uniq = sorted(set(paths))
    q = [shlex.quote(x) for x in uniq]
    add = " \\\n        ".join(" ".join(q[i:i + 3]) for i in range(0, len(q), 3)) or "."

    o = [f"## Задача {idx} из {total} · {day_ru(day_date)}", "",
         f"### `{r['tid']} {r['title']}`", "",
         "**1. Jira.** Перетащите задачу из `To Do` в `In Progress`. Делайте это в тот "
         "день, когда садитесь за неё, — не раньше.", "",
         "**2. Создайте ветку.** Имя уже подставлено, копируйте как есть:", ""]
    deps = r.get("deps", [])
    if deps:
        names = ", ".join(f"`{d['tid']} {d['title']}` ({d['who']})" for d in deps)
        many = len(deps) > 1
        o += [f"> **Эта задача зависит от {names}.** Без "
              + ("них" if many else "неё") + " ваши файлы не соберутся. Поэтому ветку "
              "растите **от " + ("их веток" if many else "его ветки") +
              f"**, а не от `{ph}`. Если какой-то ещё нет на GitHub — напишите автору и "
              "подождите.", "", "```bash", "git fetch origin",
              f"git checkout -b {br} origin/{branch(phase, deps[0]['tid'], deps[0]['who'])}"]
        for d in deps[1:]:
            o.append(f"git merge --no-edit origin/{branch(phase, d['tid'], d['who'])}")
        o += ["```", ""]
    else:
        o += ["```bash", f"git checkout {ph} && git pull", f"git checkout -b {br}",
              "```", ""]
    o += [
         "**3. Что сделать с файлами:**", "",
         "| | Файл | Примечание |", "| --- | --- | --- |"]
    for op, path, note in r["files"]:
        shown = f"`{path[0]}` → `{path[1]}`" if op == "R" else f"`{path}`"
        o.append(f"| {OP[op]} | {shown} | {note or '—'} |")
    o += ["",
          "Трогать можно **только эти файлы**. Если нужен чужой — напишите его "
          "владельцу, сами не правьте.", ""]

    for op, path, _ in r["files"]:
        if op == "A" and path in SCAFFOLD:
            o += [f"> **Как создать `{path.split('/')[0]}/` с нуля.** Руками "
                  "`package.json` и lock-файл не пишут — их делает генератор. Из корня "
                  "репозитория:", "", "```bash", SCAFFOLD[path], "```", "",
                  "> Генератор создаст больше файлов, чем в таблице. Лишние удалите "
                  "(`README.md`, картинки в `public/` и т.п.), а нужные поправьте по "
                  "`code/`. В коммит идут только файлы из таблицы.", ""]

    stubs = [p for op, p, _ in r["files"]
             if op == "A" and isinstance(p, str) and p not in FILES]
    if stubs:
        o += ["> **Пишете с нуля:** " + ", ".join(f"`{x}`" for x in stubs) +
              ". Этих файлов нет в готовом проекте — они временные и позже удаляются, "
              "поэтому в `code/` на их месте только пометка. Сделайте их маленькими и "
              "простыми, по смыслу из примечания.", ""]

    if any(isinstance(p, str) and p.endswith((".ts", ".tsx", ".go"))
           for op, p, _ in r["files"] if op in "AM"):
        o += ["> **Файлы из `code/` финальные** и импортируют то, чего в проекте ещё "
              "нет (i18n, авторизацию и т.п.). Такие импорты и код, который на них "
              "опирается, **уберите** — иначе проверка ниже не пройдёт. Главное, "
              "чтобы собиралось сейчас.", ""]

    if any("v1" in (n or "") for _, _, n in r["files"]):
        o += ["> Где в примечании **v1** — файл пишется в упрощённом виде и позже "
              "переписывается. Исходник первой версии — в `plan/file-histories.md`.", ""]

    cmds = check_cmd(paths, phase["n"])
    if cmds:
        inst = [f"cd {k} && npm install && cd .." for k in ("web", "mobile")
                if any(isinstance(p, str) and p.startswith(k + "/") for p in paths)]
        o += ["**4. Проверьте, что собирается:**", "", "```bash"] + inst + cmds + ["```", "",
              "Красный результат — не коммитим, чиним.", ""]
        if phase["n"] < MAKE_FROM:
            o += ["> Команды прямые, без `make`: `Makefile` с целями проверок появится "
                  "только к концу W1 (задачи `BF-12` и `BF-13`). С W2 то же самое "
                  "будет одной строкой `make api-check`.", ""]
    else:
        o += ["**4. Проверять сборку нечего** — в этом коммите только текст.", ""]

    mv = [f"git mv {shlex.quote(a)} {shlex.quote(b)}" for a, b in renames]
    o += ["**5. Коммит и пуш:**", "", "```bash"] + mv + [f"git add -A {add}",
          f'git commit -m "{r["tid"]} {r["title"]}"',
          f"git push -u origin {br}", "```", "",
          "Сообщение коммита — **дословно название задачи**, с номером впереди: по "
          "нему Jira сама свяжет коммит с задачей.", "",
          f"**PR сейчас не открывайте** — это делается в {day_acc(pr_day(phase))} "
          "вечером, для всех задач сразу. Карточка остаётся в `In Progress`.", "",
          "---", ""]
    return o


def student_doc(phase, who, plan, length):
    start, end = phase["span"]
    ph = week_branch(phase["n"])
    mine = sorted([(d, r) for d, r in plan if r["who"] == who], key=lambda x: x[0])
    name, mail = GIT_ID[who]
    prd = pr_day(phase)

    o = [f"# {who} · {phase['key']} — {phase['title']}", "",
         f"**{FULL[who]}** · {day_ru(start)} — {day_ru(end)}", "",
         f"Ваша зона: {ZONE[who]}.", "",
         "| | |", "| --- | --- |",
         f"| Задач у вас на этой неделе | **{len(mine)}** |",
         (f"| Значит коммитов | **{len(mine)}** — по одному на задачу |"
          if mine else "| Значит коммитов | **0** |"),
         f"| PR открываете | **в {day_acc(prd)} вечером**, все сразу |",
         ("| PR смотрит и вливает | **вы сами — вы капитан** |" if who == CAPTAIN else
          f"| PR смотрит и вливает | **капитан ({CAPTAIN})** |"), "", "---", ""]

    o += ["## Прежде чем начать", "",
          "**Один раз на компьютере** — чтобы коммит ушёл под вашим именем и "
          "связался с вашим профилем GitHub:", "", "```bash",
          f'git config user.name  "{name}"',
          (f'git config user.email "{mail}"' if mail else
           'git config user.email "почта-вашего-аккаунта-github"'), "```", ""]
    if not mail:
        o += ["> У вас ещё нет аккаунта GitHub. Заведите его, попросите капитана "
              "добавить вас в репозиторий и подставьте почту аккаунта в команду выше.", ""]
    o += ["**Ветку недели создаёт капитан** в понедельник утром. Если команда "
          f"`git checkout {ph}` пишет, что такой ветки нет, — подождите капитана, "
          "ничего не создавайте сами.", "",
          "**Задачи в Jira заводит капитан.** Вы их только двигаете по колонкам: "
          "`In Progress` когда начали, `In Review` когда открыли PR. В `Done` "
          "переводит капитан после слияния.", ""]

    if not mine:
        o += ["---", "", "## На этой неделе у вас нет своей задачи", "",
              "Так бывает только в самых коротких неделях. Что делать:", "",
              "- прийти на разбор недели;",
              "- помочь тому, у кого задача горит;",
              f"- посмотреть `weeks/week-{phase['n']:02d}-{SLUG[phase['n']]}/code/`, "
              "чтобы понимать, что сделали остальные.", ""]
        return o

    here = f"weeks/week-{phase['n']:02d}-{SLUG[phase['n']]}"
    prevdir = (f"weeks/week-{phase['n'] - 1:02d}-{SLUG[phase['n'] - 1]}"
               if phase["n"] else None)
    o += ["---", "", "## Где брать содержимое файлов", "",
          "Таблицы ниже говорят, **что** тронуть. Что именно написать:", "",
          "| Случай | Куда смотреть |", "| --- | --- |",
          f"| обычный файл | `{here}/code/<путь>` — как он выглядит в готовом проекте |",
          "| помечен **v1** | `plan/file-histories.md` — исходник первой версии; "
          "в `code/` лежит уже переписанная |"]
    if prevdir:
        o.append(f"| что было неделю назад | `{prevdir}/code/` |")
    o += ["| чей файл | `plan/ownership.map` |", "",
          "В `code/` лежит **финальное** содержимое — ориентир, а не то, что надо "
          "скопировать целиком. На этой неделе файл обычно пишется проще и дорастает "
          "позже.", "", "---", "",
          f"## {day_ru(start)} — ваши задачи уже в Jira", "",
          "Капитан завёл их в колонку `To Do`. **Ничего заводить не нужно** — "
          "проверьте только, что они там есть и вы в них исполнитель:", ""]
    for _, r in mine:
        o.append(f"- `{r['tid']} {r['title']}`")
    o += ["", "Какой-то нет — напишите капитану.", "", "---", ""]

    for i, (day, r) in enumerate(mine, 1):
        o += ticket_block(phase, start + datetime.timedelta(days=day - 1), r, i, len(mine))

    o += [f"## {day_ru(prd)} вечером — открыть PR", "",
          f"Все задачи закоммичены и запушены — теперь **{len(mine)} PR, по одному на "
          "задачу**. Для каждой строки таблицы:", "",
          "1. Откройте репозиторий на GitHub → вкладка **Pull requests** → "
          "**New pull request**.",
          "2. Слева **base** — выберите ветку недели. Справа **compare** — ветку задачи.",
          "3. Нажмите **Create pull request** — откроется форма. Заголовок "
          "впишите из таблицы.",
          "4. Справа **Reviewers** → найдите **ansarzeinulla** (это капитан).",
          "5. Ещё раз **Create pull request** — теперь PR отправлен.",
          "6. В Jira перетащите задачу в `In Review`.", "",
          "| Задача | base | compare | Заголовок PR |", "| --- | --- | --- | --- |"]
    for _, r in mine:
        o.append(f"| `{r['tid']}` | `{ph}` | `{branch(phase, r['tid'], who)}` | "
                 f"`{r['tid']} {r['title']}` |")
    o += ["", f"**base — всегда `{ph}`, не `main`.** Это самая частая ошибка.", "",
          "---", "", f"## {day_ru(end)} — капитан принимает работу", ""]
    if who == CAPTAIN:
        o += ["Вы капитан: в выходные вы принимаете PR всех пятерых, включая свои. "
              "Порядок и что проверять — в `captain.md` этой же папки.", ""]
    else:
        o += ["Больше ничего делать не нужно. Капитан посмотрит ваш PR, вольёт и "
              "переведёт задачу в `Done`.", "",
              "Если он вернул PR с замечаниями — поправьте в той же ветке, запушьте и "
              "напишите в PR «готово». **Новый PR не открывайте**: старый обновится сам.",
              ""]
    o += ["---", "", "## Если что-то пошло не так", "",
          "| Ситуация | Что делать |", "| --- | --- |",
          f"| `git checkout {ph}` пишет, что ветки нет | Капитан ещё не открыл неделю. "
          "Подождите, сами не создавайте. |",
          "| Проверка сборки красная | Чините, пока не станет зелёной. Не коммитьте "
          "красное. |",
          "| Забыли файл в коммите | Добавьте вторым коммитом в ту же ветку и запушьте. |",
          "| Нужен чужой файл | Напишите его владельцу. Сами не правите никогда. |",
          "| Не успеваете к пятнице | Скажите капитану заранее. Задача переедет на "
          "следующую неделю **со своим номером**, новую взамен не заводят. |",
          "| Открыли PR в `main` вместо недели | Закройте его и откройте заново с "
          f"base `{ph}`. |", ""]
    return o


def merge_order(phase):
    """Порядок слияния PR в выходные: схема, хендлеры, маршруты, типы, UI."""
    def num(t):
        m = re.match(r"BF-(B?)(\d+)", t)
        return (1 if m.group(1) else 0, int(m.group(2)))
    return sorted(phase["rows"], key=lambda r: (r["lane"], num(r["tid"])))


LANE = {0: "каркас папки — самым первым", 1: "схема базы", 2: "обычные задачи",
        3: "маршруты API", 4: "типы и клиент API", 5: "интерфейс — последним"}


def captain_doc(phase, plan, work, length, nxt):
    start, end = phase["span"]
    ph = week_branch(phase["n"])
    prev = week_branch(phase["n"] - 1)
    rows = sorted(phase["rows"], key=lambda r: re.sub(r"\D", "", r["tid"]).zfill(4))
    prd = pr_day(phase)
    sat = prd + datetime.timedelta(days=1)

    o = [f"# Капитан · {phase['key']} — {phase['title']}", "",
         f"**{day_ru(start)} — {day_ru(end)}** · задач на неделе: **{len(rows)}**", "",
         f"Капитан — {FULL[CAPTAIN]} ({CAPTAIN}). Свои задачи недели — в `S1.md`; "
         "здесь только то, что делаете за всю команду.", "", "---", "",
         f"## {day_ru(start)} утром — открыть неделю", ""]

    step = 1
    if phase["n"] >= 2:
        o += [f"**{step}. Влейте {prev} в `main`,** если ещё не сделали в воскресенье:",
              "", f"- GitHub → **Pull requests** → **New pull request**",
              f"- base: `main`, compare: `{prev}`, заголовок `Close W{phase['n'] - 1}`",
              "- апрув не нужен — вы администратор; вливаете, когда CI зелёный", "",
              "```bash", "git checkout main && git pull",
              f"git tag {prev}-done && git push origin {prev}-done", "```", ""]
        step += 1

    o += [f"**{step}. Создайте ветку недели** — без неё никто не начнёт:", "",
          "```bash", "git checkout main && git pull",
          f"git checkout -b {ph} && git push -u origin {ph}", "```", ""]
    step += 1

    o += [f"**{step}. Заведите в Jira все {len(rows)} "
          + plural(len(rows), "задачу", "задачи", "задач")
          + "** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по "
          "таблице:", "",
          "| Задача | Исполнитель | Тип | Эпик | Название |",
          "| --- | --- | --- | --- | --- |"]
    for r in rows:
        kind = "Bug" if r["tid"].startswith("BF-B") else "Story"
        epic = "—" if kind == "Bug" else f"`{epic_of(r['title'])}`"
        o.append(f"| `{r['tid']}` | {r['who']} · {FULL[r['who']]} | {kind} | {epic} | "
                 f"{r['title']} |")
    o += ["", "Задачи заводите **до** того, как кто-то начал работать: по датам в Jira "
          "это видно.", "", "---", "",
          f"## {day_ru(start)} — {day_ru(prd)} — следить", "",
          "Свои задачи делаете по `S1.md`. Каждый день пять минут:", "",
          "- нет ли карточки в `In Progress` без единого коммита больше суток;",
          "- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.", "",
          "| День | Кто | Задача |", "| --- | --- | --- |"]
    for day, r in sorted(plan, key=lambda x: (x[0], x[1]["who"])):
        date = start + datetime.timedelta(days=day - 1)
        o.append(f"| {day_ru(date)} | {r['who']} | `{r['tid']}` {r['title']} |")

    order = merge_order(phase)
    o += ["", "---", "",
          f"## {day_ru(prd)} вечером — приходят PR", "",
          f"Все пятеро открывают PR в `{ph}`, по одному на задачу. К утру у вас должно "
          f"быть **{len(rows)} PR**. Не хватает — напишите тому, чей нет.", "",
          "---", "", f"## {day_ru(sat)} — {day_ru(end)} — принять PR", "",
          "### Порядок — строго сверху вниз", "",
          "Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки "
          "(CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — "
          "без маршрутов. Поэтому вливаете по порядку:", "",
          "| № | Слой | Задача | Кто |", "| --- | --- | --- | --- |"]
    for i, r in enumerate(order, 1):
        o.append(f"| {i} | {LANE[r['lane']]} | `{r['tid']} {r['title']}` | {r['who']} |")
    o += ["", "### Как принять один PR", "",
          "1. Откройте PR → вкладка **Files changed**.",
          "2. Проверьте: файлы **только из зоны автора** (`plan/ownership.map`); "
          "коммит назван **дословно как задача**; код делает то, что в названии.",
          "3. Внизу PR жёлтая плашка **This branch is out-of-date** → нажмите "
          "**Update branch** и дождитесь зелёного CI. Так в ветку попадает всё, что "
          "вы влили выше по таблице.",
          "4. **Review changes** → **Approve** → **Submit review**.",
          "5. **Merge pull request** → **Confirm merge**. Ветка удалится сама.",
          "6. В Jira переведите задачу в `Done`.", "",
          "Не устраивает — **Request changes** с конкретным замечанием. Автор "
          "поправит в той же ветке, PR обновится сам.", "",
          "**Свои PR** апрувить нельзя — GitHub не даёт апрувить себя. Вливаете без "
          "апрува, но **только с зелёным CI**.", "", "---", "",
          f"## {day_ru(end)} вечером — закрыть неделю", "",
          f"**1. Все {len(rows)} задач в `Done`?** Если нет — не закрываете, выясняете.",
          "", f"**2. Влейте `{ph}` в `main`:** GitHub → **New pull request** → base "
          f"`main`, compare `{ph}` → **Create** → **Merge** (апрув не нужен).", "",
          "**3. Тег:**", "", "```bash", "git checkout main && git pull",
          f"git tag {ph}-done && git push origin {ph}-done", "```", ""]

    if phase["n"] == 8:
        o += ["**4. Контрольная точка.** После W8 продуктовый код должен совпадать с "
              "эталоном — это цель восьми недель разработки.", ""]
    if nxt:
        o += [f"Следующая неделя — `weeks/week-{nxt['n']:02d}-{SLUG[nxt['n']]}/captain.md`, "
              "раздел «открыть неделю».", ""]
    else:
        o += ["**Это последняя неделя.** Перед сдачей:", "", "```bash",
              "python3 plan/tools/check-plan.py", "bash plan/tools/check-history.sh",
              "```", ""]

    o += ["---", "", "## Чего капитан не делает", "",
          "- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.",
          "- **Не пушит в `main` напрямую.** Только через PR недели.",
          "- **Не чинит чужой код в своей ветке.** Возвращает PR автору.",
          "- **Не закрывает неделю с задачами вне `Done`.**", ""]
    return o


def open_path(phase, who, lines):
    d = f"weeks/week-{phase['n']:02d}-{SLUG[phase['n']]}"
    os.makedirs(d, exist_ok=True)
    open(f"{d}/{who}.md", "w").write("\n".join(lines) + "\n")


def main():
    phases = read()
    link_deps(phases)
    skip = frozen()
    total = 0
    for i, ph in enumerate(phases):
        if ph["n"] in skip:
            print(f"  week-{ph['n']:02d}-{SLUG[ph['n']]:<20} заморожена, не трогаю")
            continue
        plan, work, length = assign_days(ph)
        d = f"weeks/week-{ph['n']:02d}-{SLUG[ph['n']]}"
        os.makedirs(d, exist_ok=True)
        for old in ("ЗАДАЧИ.md", "JIRA.md"):
            if os.path.exists(f"{d}/{old}"):
                os.remove(f"{d}/{old}")
        for who in OWNERS:
            open_path(ph, who, student_doc(ph, who, plan, length))
        nxt = phases[i + 1] if i + 1 < len(phases) else None
        open(f"{d}/captain.md", "w").write(
            "\n".join(captain_doc(ph, plan, work, length, nxt)) + "\n")
        total += 6
        print(f"  {os.path.basename(d):26} S1…S5 + captain  ({len(ph['rows'])} задач)")
    print(f"\n  {total} файлов инструкций")


if __name__ == "__main__":
    main()
