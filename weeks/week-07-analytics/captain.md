# Капитан · W7 — Аналитика, кампании, i18n

**понедельник 26 октября — воскресенье 1 ноября** · задач на неделе: **10**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 26 октября утром — открыть неделю

**1. Влейте week-06 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `week-06`, заголовок `Close W6`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag week-06-done && git push origin week-06-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-07 && git push -u origin week-07
```

**3. Заведите в Jira все 10 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-B4` | S4 · Alinur Burlybayev | Bug | — | Read the GA4 id from a server-safe module |
| `BF-68` | S1 · Ansar Zeinulla | Story | `BF-E1` | Add campaign tables |
| `BF-69` | S2 · Alibi Takhtanov | Story | `BF-E1` | Create campaigns and promo codes |
| `BF-70` | S2 · Alibi Takhtanov | Story | `BF-E1` | Apply and cap discounts atomically |
| `BF-71` | S3 · Abylay Otaubay | Story | `BF-E11` | Add the English dictionary |
| `BF-72` | S3 · Abylay Otaubay | Story | `BF-E11` | Localise the attendee pages |
| `BF-73` | S3 · Abylay Otaubay | Story | `BF-E11` | Add the language negotiation dependency |
| `BF-74` | S4 · Alinur Burlybayev | Story | `BF-E9` | Compute analytics from operational rows |
| `BF-75` | S4 · Alinur Burlybayev | Story | `BF-E11` | Localise the dashboard |
| `BF-76` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Phase 7 specification |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 26 октября — пятница 30 октября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 26 октября | S1 | `BF-68` Add campaign tables |
| понедельник 26 октября | S2 | `BF-69` Create campaigns and promo codes |
| вторник 27 октября | S2 | `BF-70` Apply and cap discounts atomically |
| вторник 27 октября | S4 | `BF-74` Compute analytics from operational rows |
| среда 28 октября | S3 | `BF-73` Add the language negotiation dependency |
| среда 28 октября | S4 | `BF-75` Localise the dashboard |
| среда 28 октября | S5 | `BF-76` Add the Phase 7 specification |
| четверг 29 октября | S3 | `BF-72` Localise the attendee pages |
| четверг 29 октября | S4 | `BF-B4` Read the GA4 id from a server-safe module |
| пятница 30 октября | S3 | `BF-71` Add the English dictionary |

---

## пятница 30 октября вечером — приходят PR

Все пятеро открывают PR в `week-07`, по одному на задачу. К утру у вас должно быть **10 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 31 октября — воскресенье 1 ноября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | схема базы | `BF-68 Add campaign tables` | S1 |
| 2 | обычные задачи | `BF-69 Create campaigns and promo codes` | S2 |
| 3 | обычные задачи | `BF-70 Apply and cap discounts atomically` | S2 |
| 4 | обычные задачи | `BF-74 Compute analytics from operational rows` | S4 |
| 5 | обычные задачи | `BF-75 Localise the dashboard` | S4 |
| 6 | обычные задачи | `BF-76 Add the Phase 7 specification` | S5 |
| 7 | обычные задачи | `BF-B4 Read the GA4 id from a server-safe module` | S4 |
| 8 | типы и клиент API | `BF-72 Localise the attendee pages` | S3 |
| 9 | интерфейс — последним | `BF-71 Add the English dictionary` | S3 |
| 10 | интерфейс — последним | `BF-73 Add the language negotiation dependency` | S3 |

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

## воскресенье 1 ноября вечером — закрыть неделю

**1. Все 10 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-07` в `main`:** GitHub → **New pull request** → base `main`, compare `week-07` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-07-done && git push origin week-07-done
```

Следующая неделя — `weeks/week-08-release-shape/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

