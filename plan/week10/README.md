# Неделя 10 — Документация и релиз (16–22 ноября): готовые файлы и скрипты коммита

`code/<задача>/` — файлы задачи. `manifest.txt` — список по файлам,
`tickets.plan` — кто и поверх чего.

Код не меняется. Три документа доводятся до итогового вида, и после слияния
`week-10` репозиторий **совпадает с нынешним проектом файл в файл**.

## Проверено

Все три скрипта прогнаны в песочнице поверх слитой недели 9 — на GitHub ничего
не уходило (`tools/simweek.sh`): полный набор проверок зелёный, итоговое
дерево совпадает с нынешним кодом проекта.

## Перед началом недели — капитан

Когда все PR недели 9 влиты в `week-09`:

```bash
cd ~/Desktop/ticket-project
git fetch origin
git switch main && git pull
git merge --no-ff origin/week-09 -m "Week 9"
git push origin main
git switch -c week-10 && git push -u origin week-10
```

Завести в Jira `BF-93`, `BF-94`, `BF-95`, `BF-96` в **To Do**.

## Кто что запускает

Все задачи независимы, база `week-10`, порядок любой.

| Кто | Скрипт | Что внутри |
| --- | --- | --- |
| S2 Alibi | `bash ~/Desktop/bilet/plan/week10/S2-BF-93.sh` | `api/README.md` |
| S3 Abylay | `bash ~/Desktop/bilet/plan/week10/S3-BF-94.sh` | `web/README.md` |
| S5 Olzhas | `bash ~/Desktop/bilet/plan/week10/S5-BF-95.sh` | `README.md`, `Makefile` |

## Релиз — `BF-96`

Когда три PR влиты, капитан сливает неделю в `main` и ставит тег. Файлов в
этой задаче нет — это действие в git:

```bash
cd ~/Desktop/ticket-project
git fetch origin
git switch main && git pull
git merge --no-ff origin/week-10 -m "Week 10"
git push origin main
git tag -a v1.0.0 -m "BiletFlow 1.0.0"
git push origin v1.0.0
```

`BF-92` из плана не заводится: папка `plan/` в командном репозитории — только
шаблон отчёта, и он не меняется.

Задачи в Jira → **Done**.
