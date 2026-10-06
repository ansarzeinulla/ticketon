# Неделя 4 — Платные продажи и билеты (5–11 октября): готовые файлы и скрипты коммита

`code/<задача>/` — файлы задачи в том виде, в каком они должны быть **на этой
неделе**. `code/<задача>.delete` — что задача удаляет. `manifest.txt` — список по
файлам (собран `tools/gitpack.py` из рабочего репозитория, где каждая задача —
отдельный коммит). `tickets.plan` — кто, в каком порядке и поверх чего.

Итог недели:
- профиль организатора и смена пароля;
- роли: платформенный админ может действовать на любом событии; аудит изменений;
- места в зале: карта мест, покупка по местам, одно место — один живой билет;
- платные продажи открываются только после чек-листа активации (бесплатные
  тарифы продаются всегда);
- у каждого билета QR и PDF формата A4 со встроенным шрифтом — казахские и
  русские имена печатаются как есть; событие выгружается в `.ics`;
- письмо «Ваши билеты» на каждый заказ, все письма пишутся в таблицу
  `notifications`;
- `make api-smoke` — проверка по HTTP, в CI отдельной джобой;
- `4.md` — спецификация выдачи билетов.

## Проверено

Все тринадцать скриптов прогнаны в песочнице поверх слитой недели 3 — на GitHub
ничего не уходило (`tools/simweek.sh`):

- каждая ветка задачи собирается сама: API — `gofmt`, `go vet`, `go test` на
  PostgreSQL 17; веб — `lint`, `typecheck`, `vitest`; `db/` — SQL-тесты с seed;
- после слияния всех тринадцати: то же плюс `next build`, SQL-тесты 01–05 и 07,
  и дерево совпадает с эталоном файл в файл;
- `make api-smoke` прогнан вживую против API недели 4: все проверки зелёные
  (регистрация, активация, покупка, отказ при перепродаже, PDF, QR, `.ics`);
- у каждой ветки один коммит поверх своей базы, автор — владелец задачи, без
  соавторов, с настоящим временем.

## Перед началом недели — капитан

Когда все PR недели 3 влиты в `week-03`:

```bash
cd ~/Desktop/ticket-project
git fetch origin
git switch main && git pull
git merge --no-ff origin/week-03 -m "Week 3"
git push origin main
git switch -c week-04 && git push -u origin week-04
```

Потом завести в Jira задачи `BF-37` … `BF-49` в **To Do**.

## Кто что запускает

Каждый запускает **только свои** скрипты за этим компьютером; скрипт спросит
«Вы — …?» и подпишет коммит его именем. Перед запуском — задачу в Jira в
**In Progress**.

Как и в неделе 3, это одна цепочка: сначала API (S1 и S2 вперемешку, потому что
письмо о заказе ссылается на PDF, а календарь — на функции письма), затем веб.
Скрипт сам откажется, если его база ещё не запушена, и скажет, кого ждать.

| Порядок | Кто | Скрипт | Строится поверх |
| --- | --- | --- | --- |
| 1 | S1 Ansar | `bash ~/Desktop/bilet/plan/week04/S1-BF-37.sh` | `week-04` |
| 2 | S1 Ansar | `bash ~/Desktop/bilet/plan/week04/S1-BF-38.sh` | BF-37 |
| 3 | S2 Alibi | `bash ~/Desktop/bilet/plan/week04/S2-BF-40.sh` | BF-38 (Ansar) |
| 4 | S2 Alibi | `bash ~/Desktop/bilet/plan/week04/S2-BF-41.sh` | BF-40 |
| 5 | S1 Ansar | `bash ~/Desktop/bilet/plan/week04/S1-BF-39.sh` | BF-41 (Alibi) |
| 6 | S2 Alibi | `bash ~/Desktop/bilet/plan/week04/S2-BF-42.sh` | BF-39 (Ansar) |
| 7 | S3 Abylay | `bash ~/Desktop/bilet/plan/week04/S3-BF-43.sh` | BF-42 (Alibi) |
| 8 | S3 Abylay | `bash ~/Desktop/bilet/plan/week04/S3-BF-44.sh` | BF-43 |
| 9 | S3 Abylay | `bash ~/Desktop/bilet/plan/week04/S3-BF-45.sh` | BF-44 |
| 10 | S4 Alinur | `bash ~/Desktop/bilet/plan/week04/S4-BF-46.sh` | BF-45 (Abylay) |
| 11 | S4 Alinur | `bash ~/Desktop/bilet/plan/week04/S4-BF-47.sh` | BF-46 |
| 12 | S5 Olzhas | `bash ~/Desktop/bilet/plan/week04/S5-BF-48.sh` | `week-04` |
| 13 | S5 Olzhas | `bash ~/Desktop/bilet/plan/week04/S5-BF-49.sh` | `week-04` |

`BF-48` и `BF-49` от цепочки не зависят — их можно запускать когда угодно.

Проверить без пуша: `DRY_RUN=1 bash …/S5-BF-49.sh` — коммит создаётся и сразу
удаляется.

## Конец недели — PR

Каждый открывает свои PR (скрипт печатает ссылку), **base: `week-04`**.
Капитан вливает **строго в порядке таблицы**, способом **Create a merge commit**.
Пока предыдущие PR цепочки не влиты, PR показывает и их коммиты; после их
слияния остаётся только свой.

После последнего слияния:

```bash
make reset && make seed && make test   # SQL: 01–05, 07
make api-check                         # gofmt, vet, go test
make api-run                           # в отдельном терминале, затем:
make api-smoke                         # проверка по HTTP
make web-check                         # lint, typecheck, next build
```

Задачи в Jira → **Done**, `week-04` → `main`.

## Что в этой неделе временное или первой версии

| Файл | Что с ним будет |
| --- | --- |
| `api/internal/ticketpdf/text.go` | в `BF-41` — транслитерация (встроенные шрифты PDF без кириллицы), в `BF-42` карта удаляется и приезжает шрифт DejaVu; оба шага в этой же неделе |
| `api/internal/store/checkout.go` | всё ещё одна транзакция; в W5 (`BF-51`) разбивается на резерв и подтверждение |
| `api/internal/store/activation.go` | комиссия активации — константа 5 000 ₸; в W6 её можно будет менять из админки |
| `db/tests/07_temp_checks.sql` | временные проверки триггеров; удаляются в W6 |
| `web/src/components/seat-*.tsx`, `seated-purchase.tsx` | место покупается сразу; в W5 — через 15-минутный резерв с таймером |
| `web/src/components/order-list.tsx` | больше не используется страницей заказа (её место занял `ticket-card`); удаляется в W6 |
| `api/internal/store/notifications.go` | держит у себя `nullableString`; в W5 функция переезжает в `checkin.go` |
