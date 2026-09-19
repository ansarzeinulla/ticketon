# Неделя 9 — Проверка (9–15 ноября): готовые файлы и скрипты коммита

`code/<задача>/` — файлы задачи в том виде, в каком они должны быть **на этой
неделе**. `manifest.txt` — список по файлам (собран `tools/gitpack.py` из
рабочего репозитория). `tickets.plan` — кто и поверх чего.

Продуктовый код не меняется — неделя про доказательства:
- тесты маршрутизации (404/405 в JSON, request id, CORS, размер тела);
- тест ленты событий при правке, публикации и отмене;
- unit-тесты веба на Vitest с jsdom (`vitest.config.mts`, тест поля формы);
- тесты по критериям приёмки фаз (`criteria_test.go`, `phase10_test.go`,
  `phase12_test.go`), результаты прогона спецификаций и скриншоты сканера в
  `docs/verification/`;
- Playwright e2e (`web/e2e/`), CI с e2e и сборкой всех Docker-образов,
  `docker-compose` с профилем `app`, полный `make api-smoke`;
- Postman-коллекция `docs/biletflow-api.postman_collection.json`.

Полный `make api-smoke` из `BF-90` прогнан вживую против API недели 9: все
проверки зелёные.

## Проверено

Все шесть скриптов прогнаны в песочнице поверх слитой недели 8 — на GitHub
ничего не уходило (`tools/simweek.sh`):

- каждая ветка задачи собирается сама: `gofmt`, `go vet`, `go test` на
  PostgreSQL 17; веб — `lint`, `typecheck`, `vitest` (с новым `field.test.tsx`);
- после слияния всех шести: то же плюс `next build` и SQL-тесты 01–06, дерево
  совпадает с эталоном файл в файл;
- у каждой ветки один коммит поверх своей базы, автор — владелец задачи, без
  соавторов, с настоящим временем.

**Не проверялось здесь:** Playwright e2e и Docker-джоба CI — им нужны браузеры
Playwright и Docker, которых в песочнице нет. Они проверятся в GitHub Actions.

## Перед началом недели — капитан

Когда все PR недели 8 влиты в `week-08`:

```bash
cd ~/Desktop/ticket-project
git fetch origin
git switch main && git pull
git merge --no-ff origin/week-08 -m "Week 8"
git push origin main
git switch -c week-09 && git push -u origin week-09
```

Потом завести в Jira задачи `BF-84`, `BF-85`, `BF-87`, `BF-89`, `BF-90`,
`BF-91` в **To Do**.

## Кто что запускает

Все задачи, кроме одной, независимы и строятся поверх `week-09`. `BF-90`
строится поверх `BF-87` (Abylay): без конфига Vitest из `BF-87` unit-тесты
веба подхватили бы Playwright-спеки из `web/e2e/` и упали. Скрипт `BF-90` сам
откажется, пока `BF-87` не запушена.

| Кто | Скрипт | Что внутри |
| --- | --- | --- |
| S1 Ansar | `bash ~/Desktop/bilet/plan/week09/S1-BF-84.sh` | `routing_test.go` |
| S2 Alibi | `bash ~/Desktop/bilet/plan/week09/S2-BF-85.sh` | тест ленты событий в `events_test.go` |
| S3 Abylay | `bash ~/Desktop/bilet/plan/week09/S3-BF-87.sh` | Vitest: конфиг, setup, `field.test.tsx` |
| S5 Olzhas | `bash ~/Desktop/bilet/plan/week09/S5-BF-89.sh` | criteria/phase-тесты, `docs/verification/` |
| S5 Olzhas | `bash ~/Desktop/bilet/plan/week09/S5-BF-90.sh` | Playwright e2e, CI, Docker, `docker-compose`, `.gitignore`, полный smoke (после `BF-87`) |
| S5 Olzhas | `bash ~/Desktop/bilet/plan/week09/S5-BF-91.sh` | Postman-коллекция |

`BF-86` и `BF-88` из плана не заводятся: словари и фильтр аналитики уже в
итоговом виде с недель 7–8. У S4 на этой неделе своих задач нет.

Капитан вливает PR способом **Create a merge commit**; `BF-87` — раньше `BF-90`.

После последнего слияния:

```bash
make reset && make seed && make test
make api-check
make api-run                           # в отдельном терминале, затем:
make api-smoke
make web-check
make scan-check
```
