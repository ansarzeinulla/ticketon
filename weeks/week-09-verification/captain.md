# Капитан · W9 — Проверка

**понедельник 9 ноября — воскресенье 15 ноября** · задач на неделе: **8**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 9 ноября утром — открыть неделю

**1. Влейте week-08 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `week-08`, заголовок `Close W8`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag week-08-done && git push origin week-08-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-09 && git push -u origin week-09
```

**3. Заведите в Jira все 8 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-84` | S1 · Ansar Zeinulla | Story | `BF-E1` | Say "required" instead of "too short" |
| `BF-85` | S2 · Alibi Takhtanov | Story | `BF-E3` | Refuse to publish an event that has ended |
| `BF-86` | S3 · Abylay Otaubay | Story | `BF-E11` | Close the localisation gaps |
| `BF-87` | S3 · Abylay Otaubay | Story | `BF-E1` | Add the test runner dependencies |
| `BF-88` | S4 · Alinur Burlybayev | Story | `BF-E9` | Make the money card answer the tier filter |
| `BF-89` | S5 · Olzhas Nurseit | Story | `BF-E12` | Run the phase specifications and record results |
| `BF-90` | S5 · Olzhas Nurseit | Story | `BF-E12` | Correct the specifications after the run |
| `BF-91` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Postman collection |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 9 ноября — пятница 13 ноября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 9 ноября | S2 | `BF-85` Refuse to publish an event that has ended |
| понедельник 9 ноября | S4 | `BF-88` Make the money card answer the tier filter |
| вторник 10 ноября | S5 | `BF-89` Run the phase specifications and record results |
| среда 11 ноября | S1 | `BF-84` Say "required" instead of "too short" |
| среда 11 ноября | S5 | `BF-90` Correct the specifications after the run |
| четверг 12 ноября | S3 | `BF-86` Close the localisation gaps |
| четверг 12 ноября | S5 | `BF-91` Add the Postman collection |
| пятница 13 ноября | S3 | `BF-87` Add the test runner dependencies |

---

## пятница 13 ноября вечером — приходят PR

Все пятеро открывают PR в `week-09`, по одному на задачу. К утру у вас должно быть **8 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 14 ноября — воскресенье 15 ноября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | обычные задачи | `BF-85 Refuse to publish an event that has ended` | S2 |
| 2 | обычные задачи | `BF-88 Make the money card answer the tier filter` | S4 |
| 3 | обычные задачи | `BF-89 Run the phase specifications and record results` | S5 |
| 4 | обычные задачи | `BF-90 Correct the specifications after the run` | S5 |
| 5 | обычные задачи | `BF-91 Add the Postman collection` | S5 |
| 6 | интерфейс — последним | `BF-84 Say "required" instead of "too short"` | S1 |
| 7 | интерфейс — последним | `BF-86 Close the localisation gaps` | S3 |
| 8 | интерфейс — последним | `BF-87 Add the test runner dependencies` | S3 |

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

## воскресенье 15 ноября вечером — закрыть неделю

**1. Все 8 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-09` в `main`:** GitHub → **New pull request** → base `main`, compare `week-09` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-09-done && git push origin week-09-done
```

Следующая неделя — `weeks/week-10-release/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

