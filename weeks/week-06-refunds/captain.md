# Капитан · W6 — Возвраты, поддержка, админ

**понедельник 19 октября — воскресенье 25 октября** · задач на неделе: **13**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 19 октября утром — открыть неделю

**1. Влейте week-05 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `week-05`, заголовок `Close W5`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag week-05-done && git push origin week-05-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-06 && git push -u origin week-06
```

**3. Заведите в Jira все 13 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-B1` | S5 · Olzhas Nurseit | Bug | — | Keep the check-in record when a ticket is voided |
| `BF-B2` | S4 · Alinur Burlybayev | Bug | — | Move the session into an httpOnly cookie |
| `BF-B3` | S4 · Alinur Burlybayev | Bug | — | Rename middleware to proxy for Next 16 |
| `BF-58` | S1 · Ansar Zeinulla | Story | `BF-E8` | Add refund, support and report tables |
| `BF-59` | S1 · Ansar Zeinulla | Story | `BF-E10` | Add the report queue and settings |
| `BF-60` | S1 · Ansar Zeinulla | Story | `BF-E1` | Give every conflict its own error code |
| `BF-61` | S2 · Alibi Takhtanov | Story | `BF-E1` | Refund an order atomically |
| `BF-62` | S2 · Alibi Takhtanov | Story | `BF-E10` | Expose the rows the admin search needs |
| `BF-63` | S3 · Abylay Otaubay | Story | `BF-E8` | Add the support widget to the order page |
| `BF-64` | S3 · Abylay Otaubay | Story | `BF-E11` | Point the client at the proxy |
| `BF-65` | S4 · Alinur Burlybayev | Story | `BF-E10` | Add the admin portal |
| `BF-66` | S5 · Olzhas Nurseit | Story | `BF-E7` | Refuse a refunded ticket at the gate |
| `BF-67` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Phase 6 specification |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 19 октября — пятница 23 октября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 19 октября | S1 | `BF-58` Add refund, support and report tables |
| понедельник 19 октября | S2 | `BF-61` Refund an order atomically |
| вторник 20 октября | S2 | `BF-62` Expose the rows the admin search needs |
| вторник 20 октября | S4 | `BF-65` Add the admin portal |
| вторник 20 октября | S5 | `BF-66` Refuse a refunded ticket at the gate |
| среда 21 октября | S4 | `BF-B2` Move the session into an httpOnly cookie |
| среда 21 октября | S5 | `BF-67` Add the Phase 6 specification |
| четверг 22 октября | S1 | `BF-59` Add the report queue and settings |
| четверг 22 октября | S3 | `BF-64` Point the client at the proxy |
| четверг 22 октября | S4 | `BF-B3` Rename middleware to proxy for Next 16 |
| четверг 22 октября | S5 | `BF-B1` Keep the check-in record when a ticket is voided |
| пятница 23 октября | S1 | `BF-60` Give every conflict its own error code |
| пятница 23 октября | S3 | `BF-63` Add the support widget to the order page |

---

## пятница 23 октября вечером — приходят PR

Все пятеро открывают PR в `week-06`, по одному на задачу. К утру у вас должно быть **13 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 24 октября — воскресенье 25 октября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | схема базы | `BF-58 Add refund, support and report tables` | S1 |
| 2 | обычные задачи | `BF-61 Refund an order atomically` | S2 |
| 3 | обычные задачи | `BF-62 Expose the rows the admin search needs` | S2 |
| 4 | обычные задачи | `BF-65 Add the admin portal` | S4 |
| 5 | обычные задачи | `BF-66 Refuse a refunded ticket at the gate` | S5 |
| 6 | обычные задачи | `BF-67 Add the Phase 6 specification` | S5 |
| 7 | обычные задачи | `BF-B1 Keep the check-in record when a ticket is voided` | S5 |
| 8 | обычные задачи | `BF-B2 Move the session into an httpOnly cookie` | S4 |
| 9 | обычные задачи | `BF-B3 Rename middleware to proxy for Next 16` | S4 |
| 10 | типы и клиент API | `BF-64 Point the client at the proxy` | S3 |
| 11 | интерфейс — последним | `BF-59 Add the report queue and settings` | S1 |
| 12 | интерфейс — последним | `BF-60 Give every conflict its own error code` | S1 |
| 13 | интерфейс — последним | `BF-63 Add the support widget to the order page` | S3 |

### Как принять один PR

1. Откройте PR → вкладка **Files changed**.
2. Проверьте: файлы **только из зоны автора** (`plan/ownership.map`); коммит назван **дословно как задача**; код делает то, что в названии.
3. Внизу PR жёлтая плашка **This branch is out-of-date** → нажмите **Update branch** и дождитесь зелёного CI. Так в ветку попадает всё, что вы влили выше по таблице.
4. **Review changes** → **Approve** → **Submit review**.
5. **Merge pull request** → **Confirm merge**. Ветка удалится сама.
6. В Jira переведите задачу в `Done`.

Не устраивает — **Request changes** с конкретным замечанием. Автор поправит в той же ветке, PR обновится сам.

**Свои PR** апрувить нельзя — GitHub не даёт апрувить себя. Вливаете без апрува, но **только с зелёным CI**.

---

## воскресенье 25 октября вечером — закрыть неделю

**1. Все 13 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-06` в `main`:** GitHub → **New pull request** → base `main`, compare `week-06` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-06-done && git push origin week-06-done
```

Следующая неделя — `weeks/week-07-analytics/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

