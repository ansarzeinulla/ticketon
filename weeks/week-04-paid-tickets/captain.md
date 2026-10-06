# Капитан · W4 — Платные продажи и билеты

**понедельник 5 октября — воскресенье 11 октября** · задач на неделе: **13**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 5 октября утром — открыть неделю

**1. Влейте week-03 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `week-03`, заголовок `Close W3`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag week-03-done && git push origin week-03-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-04 && git push -u origin week-04
```

**3. Заведите в Jira все 13 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-37` | S1 · Ansar Zeinulla | Story | `BF-E1` | Add organizer profiles and money handling |
| `BF-38` | S1 · Ansar Zeinulla | Story | `BF-E2` | Enforce role permissions |
| `BF-39` | S1 · Ansar Zeinulla | Story | `BF-E5` | Send the order confirmation |
| `BF-40` | S2 · Alibi Takhtanov | Story | `BF-E4` | Add seat inventory |
| `BF-41` | S2 · Alibi Takhtanov | Story | `BF-E4` | Gate paid sales behind activation |
| `BF-42` | S2 · Alibi Takhtanov | Story | `BF-E5` | Embed a Unicode font for Cyrillic |
| `BF-43` | S3 · Abylay Otaubay | Story | `BF-E4` | Show the activation notice |
| `BF-44` | S3 · Abylay Otaubay | Story | `BF-E4` | Add the seat map and seated checkout |
| `BF-45` | S3 · Abylay Otaubay | Story | `BF-E5` | Link the printable PDF |
| `BF-46` | S4 · Alinur Burlybayev | Story | `BF-E4` | Add the activation checklist |
| `BF-47` | S4 · Alinur Burlybayev | Story | `BF-E5` | Show ticket status per order |
| `BF-48` | S5 · Olzhas Nurseit | Story | `BF-E4` | Prove checkout cannot oversell |
| `BF-49` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Phase 4 specification |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 5 октября — пятница 9 октября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 5 октября | S1 | `BF-37` Add organizer profiles and money handling |
| вторник 6 октября | S1 | `BF-38` Enforce role permissions |
| вторник 6 октября | S2 | `BF-40` Add seat inventory |
| среда 7 октября | S1 | `BF-39` Send the order confirmation |
| среда 7 октября | S2 | `BF-41` Gate paid sales behind activation |
| среда 7 октября | S3 | `BF-45` Link the printable PDF |
| среда 7 октября | S4 | `BF-46` Add the activation checklist |
| среда 7 октября | S5 | `BF-48` Prove checkout cannot oversell |
| четверг 8 октября | S2 | `BF-42` Embed a Unicode font for Cyrillic |
| четверг 8 октября | S3 | `BF-43` Show the activation notice |
| четверг 8 октября | S4 | `BF-47` Show ticket status per order |
| четверг 8 октября | S5 | `BF-49` Add the Phase 4 specification |
| пятница 9 октября | S3 | `BF-44` Add the seat map and seated checkout |

---

## пятница 9 октября вечером — приходят PR

Все пятеро открывают PR в `week-04`, по одному на задачу. К утру у вас должно быть **13 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 10 октября — воскресенье 11 октября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | схема базы | `BF-37 Add organizer profiles and money handling` | S1 |
| 2 | схема базы | `BF-38 Enforce role permissions` | S1 |
| 3 | схема базы | `BF-39 Send the order confirmation` | S1 |
| 4 | обычные задачи | `BF-40 Add seat inventory` | S2 |
| 5 | обычные задачи | `BF-41 Gate paid sales behind activation` | S2 |
| 6 | обычные задачи | `BF-42 Embed a Unicode font for Cyrillic` | S2 |
| 7 | обычные задачи | `BF-46 Add the activation checklist` | S4 |
| 8 | обычные задачи | `BF-47 Show ticket status per order` | S4 |
| 9 | обычные задачи | `BF-48 Prove checkout cannot oversell` | S5 |
| 10 | обычные задачи | `BF-49 Add the Phase 4 specification` | S5 |
| 11 | типы и клиент API | `BF-43 Show the activation notice` | S3 |
| 12 | типы и клиент API | `BF-44 Add the seat map and seated checkout` | S3 |
| 13 | типы и клиент API | `BF-45 Link the printable PDF` | S3 |

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

## воскресенье 11 октября вечером — закрыть неделю

**1. Все 13 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-04` в `main`:** GitHub → **New pull request** → base `main`, compare `week-04` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-04-done && git push origin week-04-done
```

Следующая неделя — `weeks/week-05-gate/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

