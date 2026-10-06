# Неделя 2 — Идентичность (21–27 сентября): готовые файлы и скрипты коммита

`code/<задача>/` — файлы задачи в том виде, в каком они должны быть **на этой
неделе**. `code/<задача>.delete` — что задача удаляет. `manifest.txt` — откуда
всё это взялось, по файлам.

Итог недели: регистрация, вход, сброс пароля и подтверждение почты в API и на
сайте; мобильное приложение Expo входит в аккаунт; `1.md` — спецификация проверки.

## Проверено

Все девять скриптов прогнаны в песочнице — локальной копии репозитория с
фейковым `origin`, на GitHub ничего не уходило (`tools/simweek.sh`):

- каждая ветка задачи **собирается сама**, поверх своей базы: для API — `gofmt`,
  `go vet`, `go test` на PostgreSQL 17; для веба — `lint` и `typecheck`; для
  мобильного — `tsc`; для `db/` — SQL-тесты;
- после слияния всех девяти в порядке ниже: те же проверки плюс `next build`,
  SQL-тесты 01–04 (96 проверок), и дерево совпадает с эталоном файл в файл;
- у каждой ветки ровно один коммит, автор — владелец задачи, без соавторов, с
  настоящим временем.

## Перед началом недели — капитан

Когда все PR недели 1 влиты в `phase-01`:

```bash
cd ~/Desktop/ticket-project
git fetch origin
git switch main && git pull
git merge --no-ff origin/phase-01 -m "Week 1"
git push origin main
git switch -c week-02 && git push -u origin week-02
```

Потом завести в Jira девять задач `BF-14` … `BF-22` в **To Do**.

## Кто что запускает

Каждый скрипт — одна задача одного человека. Запускает **сам этот человек**,
сидя за этим компьютером: скрипт спросит «Вы — …?» и подпишет коммит его
именем. Перед запуском — перевести свою задачу в Jira в **In Progress**.

| Порядок | Кто | Скрипт | Строится поверх |
| --- | --- | --- | --- |
| 1 | S1 Ansar | `bash ~/Desktop/bilet/plan/week02/S1-BF-14.sh` | `week-02` |
| 2 | S1 Ansar | `bash ~/Desktop/bilet/plan/week02/S1-BF-15.sh` | BF-14 |
| 3 | S1 Ansar | `bash ~/Desktop/bilet/plan/week02/S1-BF-16.sh` | BF-15 |
| 4 | S2 Alibi | `bash ~/Desktop/bilet/plan/week02/S2-BF-17.sh` | `week-02` |
| 5 | S3 Abylay | `bash ~/Desktop/bilet/plan/week02/S3-BF-18.sh` | `week-02` |
| 6 | S4 Alinur | `bash ~/Desktop/bilet/plan/week02/S4-BF-20.sh` | BF-18 (Abylay) |
| 7 | S3 Abylay | `bash ~/Desktop/bilet/plan/week02/S3-BF-19.sh` | BF-20 (Alinur) |
| 8 | S5 Olzhas | `bash ~/Desktop/bilet/plan/week02/S5-BF-21.sh` | `week-02` |
| 9 | S5 Olzhas | `bash ~/Desktop/bilet/plan/week02/S5-BF-22.sh` | `week-02` |

Порядок важен только внутри двух цепочек: **BF-14 → BF-15 → BF-16** и
**BF-18 → BF-20 → BF-19**. Скрипт сам откажется, если его база ещё не запушена,
и скажет, кого ждать. `BF-17`, `BF-21`, `BF-22` можно запускать когда угодно.

Почему цепочка S3 → S4 → S3: форме входа (`BF-19`) нужен `useAuth()` из
`BF-20`, а `BF-20` вызывает `api.login()` из `BF-18`.

Проверить без пуша: `DRY_RUN=1 bash …/S2-BF-17.sh` — коммит создаётся и сразу
удаляется.

## Конец недели — PR

Каждый открывает свой PR на GitHub: скрипт печатает готовую ссылку, **base:
`week-02`**. Капитан вливает строго в порядке таблицы, способом **Create a merge
commit** (не Squash и не Rebase: у цепочек общие коммиты, и squash развалит
следующий PR).

Пока `BF-18` не влит, PR `BF-20` показывает два коммита — свой и `BF-18`; после
слияния `BF-18` остаётся только свой. Так же `BF-19` после `BF-20`.

После последнего слияния:

```bash
make up && make test        # SQL: 01–04
make api-check              # gofmt, vet, go test (нужен make up)
make web-check              # lint, typecheck, next build
cd mobile && npm install && npx tsc --noEmit
```

Задачи в Jira → **Done**, `week-02` → `main`.

## Что в этой неделе временное

Эти файлы честно работают сейчас и будут заменены позже — так и задумано:

| Файл | Что с ним будет |
| --- | --- |
| `api/internal/auth/password.go` | лимит bcrypt в 72 байта; в W8 заменится пре-хешем SHA-256 |
| `api/internal/api/validate.go` | длины считаются в байтах; в W8 — в символах |
| `api/internal/email/console.go` | печать писем в консоль; удаляется в W4 (`BF-39`) |
| `api/internal/api/fixtures_test.go` | тестовая база; в W4 уступает `internal/testutil` |
| `web/src/lib/env.ts`, `format.ts` | удаляются в W3 (`BF-28`) |
| `web/src/lib/session.ts` | токен в cookie, видимой из JS; в W6 уезжает в httpOnly |
| `web/src/middleware.ts` | в W6 переименовывается в `proxy.ts` (Next 16 ругается при сборке — это ожидаемо) |
| `mobile/app/index.tsx` | экран «вы вошли»; в W3 его место займёт список событий |
