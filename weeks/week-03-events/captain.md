# Капитан · W3 — События, каталог и регистрация

**понедельник 28 сентября — воскресенье 4 октября** · задач на неделе: **14**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 28 сентября утром — открыть неделю

**1. Влейте week-02 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `week-02`, заголовок `Close W2`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag week-02-done && git push origin week-02-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-03 && git push -u origin week-03
```

**3. Заведите в Jira все 14 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-23` | S1 · Ansar Zeinulla | Story | `BF-E3` | Seed a demo organizer and event |
| `BF-24` | S1 · Ansar Zeinulla | Story | `BF-E1` | Add order, ticket and attendee tables |
| `BF-25` | S2 · Alibi Takhtanov | Story | `BF-E3` | Derive a slug from the title |
| `BF-26` | S2 · Alibi Takhtanov | Story | `BF-E3` | Manage ticket types |
| `BF-27` | S2 · Alibi Takhtanov | Story | `BF-E4` | Add the checkout transaction |
| `BF-28` | S3 · Abylay Otaubay | Story | `BF-E1` | Add the formatting dependencies |
| `BF-29` | S3 · Abylay Otaubay | Story | `BF-E11` | Add the public catalogue |
| `BF-30` | S3 · Abylay Otaubay | Story | `BF-E11` | Add the order page |
| `BF-31` | S4 · Alinur Burlybayev | Story | `BF-E3` | Move the create form to the client |
| `BF-32` | S4 · Alinur Burlybayev | Story | `BF-E3` | Add the organizer dashboard |
| `BF-33` | S4 · Alinur Burlybayev | Story | `BF-E1` | List orders and attendees |
| `BF-34` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Phase 2 specification |
| `BF-35` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Phase 3 specification |
| `BF-36` | S5 · Olzhas Nurseit | Story | `BF-E3` | List assigned events on the device |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 28 сентября — пятница 2 октября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 28 сентября | S1 | `BF-23` Seed a demo organizer and event |
| понедельник 28 сентября | S2 | `BF-25` Derive a slug from the title |
| вторник 29 сентября | S1 | `BF-24` Add order, ticket and attendee tables |
| вторник 29 сентября | S2 | `BF-26` Manage ticket types |
| вторник 29 сентября | S4 | `BF-31` Move the create form to the client |
| среда 30 сентября | S2 | `BF-27` Add the checkout transaction |
| среда 30 сентября | S3 | `BF-30` Add the order page |
| среда 30 сентября | S4 | `BF-32` Add the organizer dashboard |
| среда 30 сентября | S5 | `BF-34` Add the Phase 2 specification |
| четверг 1 октября | S3 | `BF-28` Add the formatting dependencies |
| четверг 1 октября | S4 | `BF-33` List orders and attendees |
| четверг 1 октября | S5 | `BF-35` Add the Phase 3 specification |
| пятница 2 октября | S3 | `BF-29` Add the public catalogue |
| пятница 2 октября | S5 | `BF-36` List assigned events on the device |

---

## пятница 2 октября вечером — приходят PR

Все пятеро открывают PR в `week-03`, по одному на задачу. К утру у вас должно быть **14 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 3 октября — воскресенье 4 октября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | схема базы | `BF-23 Seed a demo organizer and event` | S1 |
| 2 | схема базы | `BF-24 Add order, ticket and attendee tables` | S1 |
| 3 | обычные задачи | `BF-25 Derive a slug from the title` | S2 |
| 4 | обычные задачи | `BF-26 Manage ticket types` | S2 |
| 5 | обычные задачи | `BF-27 Add the checkout transaction` | S2 |
| 6 | обычные задачи | `BF-31 Move the create form to the client` | S4 |
| 7 | обычные задачи | `BF-32 Add the organizer dashboard` | S4 |
| 8 | обычные задачи | `BF-33 List orders and attendees` | S4 |
| 9 | обычные задачи | `BF-34 Add the Phase 2 specification` | S5 |
| 10 | обычные задачи | `BF-35 Add the Phase 3 specification` | S5 |
| 11 | обычные задачи | `BF-36 List assigned events on the device` | S5 |
| 12 | типы и клиент API | `BF-28 Add the formatting dependencies` | S3 |
| 13 | типы и клиент API | `BF-29 Add the public catalogue` | S3 |
| 14 | типы и клиент API | `BF-30 Add the order page` | S3 |

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

## воскресенье 4 октября вечером — закрыть неделю

**1. Все 14 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-03` в `main`:** GitHub → **New pull request** → base `main`, compare `week-03` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-03-done && git push origin week-03-done
```

Следующая неделя — `weeks/week-04-paid-tickets/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

