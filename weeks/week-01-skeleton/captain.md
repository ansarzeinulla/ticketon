# Капитан · W1 — Скелет

**понедельник 14 сентября — воскресенье 20 сентября** · задач на неделе: **8**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 14 сентября утром — открыть неделю

**1. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b phase-01 && git push -u origin phase-01
```

**2. Заведите в Jira все 8 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-6` | S1 · Ansar Zeinulla | Story | `BF-E1` | Add extensions and the users table |
| `BF-7` | S1 · Ansar Zeinulla | Story | `BF-E1` | Add config and the error envelope |
| `BF-8` | S1 · Ansar Zeinulla | Story | `BF-E1` | Add the pool and a health route |
| `BF-9` | S2 · Alibi Takhtanov | Story | `BF-E1` | Document the API layout |
| `BF-10` | S3 · Abylay Otaubay | Story | `BF-E11` | Scaffold Next.js |
| `BF-11` | S4 · Alinur Burlybayev | Story | `BF-E1` | Add the signed-in shell |
| `BF-12` | S5 · Olzhas Nurseit | Story | `BF-E1` | Add Postgres to compose |
| `BF-13` | S5 · Olzhas Nurseit | Story | `BF-E1` | Add CI |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 14 сентября — пятница 18 сентября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 14 сентября | S1 | `BF-7` Add config and the error envelope |
| понедельник 14 сентября | S3 | `BF-10` Scaffold Next.js |
| вторник 15 сентября | S1 | `BF-6` Add extensions and the users table |
| вторник 15 сентября | S2 | `BF-9` Document the API layout |
| среда 16 сентября | S4 | `BF-11` Add the signed-in shell |
| среда 16 сентября | S5 | `BF-12` Add Postgres to compose |
| четверг 17 сентября | S5 | `BF-13` Add CI |
| пятница 18 сентября | S1 | `BF-8` Add the pool and a health route |

---

## пятница 18 сентября вечером — приходят PR

Все пятеро открывают PR в `phase-01`, по одному на задачу. К утру у вас должно быть **8 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 19 сентября — воскресенье 20 сентября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | каркас папки — самым первым | `BF-7 Add config and the error envelope` | S1 |
| 2 | каркас папки — самым первым | `BF-10 Scaffold Next.js` | S3 |
| 3 | схема базы | `BF-6 Add extensions and the users table` | S1 |
| 4 | обычные задачи | `BF-9 Document the API layout` | S2 |
| 5 | обычные задачи | `BF-11 Add the signed-in shell` | S4 |
| 6 | обычные задачи | `BF-12 Add Postgres to compose` | S5 |
| 7 | обычные задачи | `BF-13 Add CI` | S5 |
| 8 | маршруты API | `BF-8 Add the pool and a health route` | S1 |

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

## воскресенье 20 сентября вечером — закрыть неделю

**1. Все 8 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `phase-01` в `main`:** GitHub → **New pull request** → base `main`, compare `phase-01` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag phase-01-done && git push origin phase-01-done
```

Следующая неделя — `weeks/week-02-identity/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

