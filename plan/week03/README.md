# Неделя 3 — События, каталог и регистрация (28 сентября – 4 октября): готовые файлы и скрипты коммита

`code/<задача>/` — файлы задачи в том виде, в каком они должны быть **на этой
неделе**. `code/<задача>.delete` — что задача удаляет. `manifest.txt` — откуда
всё это взялось, по файлам.

Итог недели: организатор создаёт событие с баннером, добавляет платные и
бесплатные типы билетов, публикует; зритель находит событие в каталоге
`/events`, покупает билет (платёж симулирован) и получает страницу заказа;
организатор видит заказы и список гостей. В мобильном — список опубликованных
событий. `make seed` загружает демо-данные, `2.md` и `3.md` — спецификации.

## Проверено

Все четырнадцать скриптов прогнаны в песочнице поверх слитой недели 2 — на
GitHub ничего не уходило (`tools/simweek.sh`):

- каждая ветка задачи **собирается сама**: API — `gofmt`, `go vet`, `go test` на
  PostgreSQL 17; веб — `lint`, `typecheck`, `vitest`; мобильное — `tsc`;
  `db/` — SQL-тесты вместе с seed;
- после слияния всех четырнадцати в порядке ниже: то же плюс `next build`,
  SQL-тесты 01–05;
- у каждой ветки ровно один коммит поверх своей базы, автор — владелец задачи,
  без соавторов, с настоящим временем.

## Перед началом недели — капитан

Когда все PR недели 2 влиты в `week-02`:

```bash
cd ~/Desktop/ticket-project
git fetch origin
git switch main && git pull
git merge --no-ff origin/week-02 -m "Week 2"
git push origin main
git switch -c week-03 && git push -u origin week-03
```

Потом завести в Jira задачи `BF-23` … `BF-36` в **To Do**.

## Кто что запускает

Каждый запускает **только свои** скрипты, сидя за этим компьютером; скрипт
спросит «Вы — …?» и подпишет коммит его именем. Перед запуском — перевести
задачу в Jira в **In Progress**.

Неделя — это **одна цепочка**: API, затем веб зрителя и организатора. Так
вышло потому, что `BF-33` (S4) трогает и API, и дашборд, а дашборд стоит на
формах, а формы — на клиенте API. Скрипт сам откажется, если его база ещё не
запушена, и скажет, кого ждать.

| Порядок | Кто | Скрипт | Строится поверх |
| --- | --- | --- | --- |
| 1 | S1 Ansar | `bash ~/Desktop/bilet/plan/week03/S1-BF-23.sh` | `week-03` |
| 2 | S1 Ansar | `bash ~/Desktop/bilet/plan/week03/S1-BF-24.sh` | `week-03` |
| 3 | S2 Alibi | `bash ~/Desktop/bilet/plan/week03/S2-BF-25.sh` | BF-23 (Ansar) |
| 4 | S2 Alibi | `bash ~/Desktop/bilet/plan/week03/S2-BF-26.sh` | BF-25 |
| 5 | S2 Alibi | `bash ~/Desktop/bilet/plan/week03/S2-BF-27.sh` | BF-26 |
| 6 | S3 Abylay | `bash ~/Desktop/bilet/plan/week03/S3-BF-28.sh` | BF-27 (Alibi) |
| 7 | S4 Alinur | `bash ~/Desktop/bilet/plan/week03/S4-BF-31.sh` | BF-28 (Abylay) |
| 8 | S4 Alinur | `bash ~/Desktop/bilet/plan/week03/S4-BF-32.sh` | BF-31 |
| 9 | S3 Abylay | `bash ~/Desktop/bilet/plan/week03/S3-BF-29.sh` | BF-32 (Alinur) |
| 10 | S3 Abylay | `bash ~/Desktop/bilet/plan/week03/S3-BF-30.sh` | BF-29 |
| 11 | S4 Alinur | `bash ~/Desktop/bilet/plan/week03/S4-BF-33.sh` | BF-30 (Abylay) |
| 12 | S5 Olzhas | `bash ~/Desktop/bilet/plan/week03/S5-BF-34.sh` | `week-03` |
| 13 | S5 Olzhas | `bash ~/Desktop/bilet/plan/week03/S5-BF-35.sh` | `week-03` |
| 14 | S5 Olzhas | `bash ~/Desktop/bilet/plan/week03/S5-BF-36.sh` | `week-03` |

`BF-24`, `BF-34`, `BF-35`, `BF-36` от цепочки не зависят — их можно запускать
когда угодно. Почему S3 и S4 чередуются: `BF-29` делает главную страницу
развилкой «вошёл → `/dashboard`», поэтому ей нужен уже существующий дашборд из
`BF-32`.

Проверить без пуша: `DRY_RUN=1 bash …/S1-BF-24.sh` — коммит создаётся и сразу
удаляется.

## Конец недели — PR

Каждый открывает свои PR на GitHub: скрипт печатает ссылку, **base: `week-03`**.
Капитан вливает **строго в порядке таблицы**, способом **Create a merge commit**
(не Squash и не Rebase: у цепочки общие коммиты).

Пока предыдущие PR цепочки не влиты, PR показывает и их коммиты — например, PR
`BF-28` показывает коммиты `BF-23`, `BF-25`, `BF-26`, `BF-27` и свой. После
слияния предыдущих остаётся только свой. Поэтому порядок слияния важен.

После последнего слияния:

```bash
make reset && make seed && make test   # SQL: 01–05 вместе с демо-данными
make api-check                         # gofmt, vet, go test
make web-check                         # lint, typecheck, next build
cd web && npx vitest run               # money и datetime
cd mobile && npx tsc --noEmit
```

Задачи в Jira → **Done**, `week-03` → `main`.

## Что в этой неделе временное

| Файл | Что с ним будет |
| --- | --- |
| `api/internal/store/checkout.go` | одна монолитная транзакция; в W5 (`BF-51`) разбивается на резерв и подтверждение |
| `api/internal/store/inventory.go`, `api/internal/api/handlers_inventory.go` | остатки отдельным стором; в W4 (`BF-40`) вливаются в оформление заказа |
| `api/internal/store/guests.go` | в W6 переименовывается в `attendees.go` («attendee», как в SRS) |
| `db/seed/01_demo_data.sql` | «удалить и вставить заново»; в W8 переписывается на upsert |
| `web/src/components/ui/toast.tsx` | всплывающие уведомления; удаляется в W6 |
| `web/src/components/order-list.tsx` | коды билетов на странице заказа; в W4 его место займёт `ticket-card` с QR |
| `web/src/app/events/[slug]/page.tsx`, `ticket-selector.tsx`, `checkout-dialog.tsx` | без промокодов, мест и перевода; они придут в W4–W7 |
| `mobile/app/events.tsx` | события организатора; в W5 — события, назначенные на вход |
