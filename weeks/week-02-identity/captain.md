# Капитан · W2 — Идентичность

**понедельник 21 сентября — воскресенье 27 сентября** · задач на неделе: **9**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 21 сентября утром — открыть неделю

**1. Влейте phase-01 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `phase-01`, заголовок `Close W1`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag phase-01-done && git push origin phase-01-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-02 && git push -u origin week-02
```

**3. Заведите в Jira все 9 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-14` | S1 · Ansar Zeinulla | Story | `BF-E2` | Hash passwords with bcrypt |
| `BF-15` | S1 · Ansar Zeinulla | Story | `BF-E1` | Reject duplicate emails with 409 |
| `BF-16` | S1 · Ansar Zeinulla | Story | `BF-E1` | Cover users, tokens and mail |
| `BF-17` | S2 · Alibi Takhtanov | Story | `BF-E1` | Document the authentication endpoints |
| `BF-18` | S3 · Abylay Otaubay | Story | `BF-E1` | Add the API client |
| `BF-19` | S3 · Abylay Otaubay | Story | `BF-E2` | Add reset and verify pages |
| `BF-20` | S4 · Alinur Burlybayev | Story | `BF-E2` | Keep the session in the browser |
| `BF-21` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Phase 1 specification |
| `BF-22` | S5 · Olzhas Nurseit | Story | `BF-E1` | Scaffold the Expo app and sign in on the device |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 21 сентября — пятница 25 сентября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 21 сентября | S1 | `BF-14` Hash passwords with bcrypt |
| понедельник 21 сентября | S5 | `BF-22` Scaffold the Expo app and sign in on the device |
| вторник 22 сентября | S1 | `BF-15` Reject duplicate emails with 409 |
| вторник 22 сентября | S2 | `BF-17` Document the authentication endpoints |
| среда 23 сентября | S4 | `BF-20` Keep the session in the browser |
| среда 23 сентября | S5 | `BF-21` Add the Phase 1 specification |
| четверг 24 сентября | S3 | `BF-18` Add the API client |
| пятница 25 сентября | S1 | `BF-16` Cover users, tokens and mail |
| пятница 25 сентября | S3 | `BF-19` Add reset and verify pages |

---

## пятница 25 сентября вечером — приходят PR

Все пятеро открывают PR в `week-02`, по одному на задачу. К утру у вас должно быть **9 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 26 сентября — воскресенье 27 сентября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | каркас папки — самым первым | `BF-22 Scaffold the Expo app and sign in on the device` | S5 |
| 2 | схема базы | `BF-14 Hash passwords with bcrypt` | S1 |
| 3 | схема базы | `BF-15 Reject duplicate emails with 409` | S1 |
| 4 | обычные задачи | `BF-17 Document the authentication endpoints` | S2 |
| 5 | обычные задачи | `BF-20 Keep the session in the browser` | S4 |
| 6 | обычные задачи | `BF-21 Add the Phase 1 specification` | S5 |
| 7 | типы и клиент API | `BF-18 Add the API client` | S3 |
| 8 | типы и клиент API | `BF-19 Add reset and verify pages` | S3 |
| 9 | интерфейс — последним | `BF-16 Cover users, tokens and mail` | S1 |

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

## воскресенье 27 сентября вечером — закрыть неделю

**1. Все 9 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-02` в `main`:** GitHub → **New pull request** → base `main`, compare `week-02` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-02-done && git push origin week-02-done
```

Следующая неделя — `weeks/week-03-events/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

