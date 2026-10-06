# Капитан · W5 — Контроль входа

**понедельник 12 октября — воскресенье 18 октября** · задач на неделе: **8**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 12 октября утром — открыть неделю

**1. Влейте week-04 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `week-04`, заголовок `Close W4`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag week-04-done && git push origin week-04-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-05 && git push -u origin week-05
```

**3. Заведите в Jira все 8 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-50` | S1 · Ansar Zeinulla | Story | `BF-E4` | Update the money assertions for the fee |
| `BF-51` | S2 · Alibi Takhtanov | Story | `BF-E4` | Reserve stock with 15-minute holds |
| `BF-52` | S2 · Alibi Takhtanov | Story | `BF-E4` | Charge a processing fee |
| `BF-53` | S3 · Abylay Otaubay | Story | `BF-E4` | Show the hold countdown at checkout |
| `BF-54` | S4 · Alinur Burlybayev | Story | `BF-E6` | Add the staff manager |
| `BF-55` | S5 · Olzhas Nurseit | Story | `BF-E6` | Prepare the gate backend and a fake scan payload |
| `BF-56` | S5 · Olzhas Nurseit | Story | `BF-E6` | Scan and admit a ticket |
| `BF-57` | S5 · Olzhas Nurseit | Story | `BF-E6` | Document how to run the scanner |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 12 октября — пятница 16 октября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 12 октября | S1 | `BF-50` Update the money assertions for the fee |
| понедельник 12 октября | S2 | `BF-51` Reserve stock with 15-minute holds |
| вторник 13 октября | S2 | `BF-52` Charge a processing fee |
| вторник 13 октября | S4 | `BF-54` Add the staff manager |
| среда 14 октября | S5 | `BF-55` Prepare the gate backend and a fake scan payload |
| четверг 15 октября | S5 | `BF-56` Scan and admit a ticket |
| пятница 16 октября | S3 | `BF-53` Show the hold countdown at checkout |
| пятница 16 октября | S5 | `BF-57` Document how to run the scanner |

---

## пятница 16 октября вечером — приходят PR

Все пятеро открывают PR в `week-05`, по одному на задачу. К утру у вас должно быть **8 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 17 октября — воскресенье 18 октября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | схема базы | `BF-50 Update the money assertions for the fee` | S1 |
| 2 | обычные задачи | `BF-51 Reserve stock with 15-minute holds` | S2 |
| 3 | обычные задачи | `BF-52 Charge a processing fee` | S2 |
| 4 | обычные задачи | `BF-54 Add the staff manager` | S4 |
| 5 | обычные задачи | `BF-55 Prepare the gate backend and a fake scan payload` | S5 |
| 6 | обычные задачи | `BF-56 Scan and admit a ticket` | S5 |
| 7 | обычные задачи | `BF-57 Document how to run the scanner` | S5 |
| 8 | типы и клиент API | `BF-53 Show the hold countdown at checkout` | S3 |

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

## воскресенье 18 октября вечером — закрыть неделю

**1. Все 8 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-05` в `main`:** GitHub → **New pull request** → base `main`, compare `week-05` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-05-done && git push origin week-05-done
```

Следующая неделя — `weeks/week-06-refunds/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

