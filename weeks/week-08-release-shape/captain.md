# Капитан · W8 — Релизная форма

**понедельник 2 ноября — воскресенье 8 ноября** · задач на неделе: **13**

Капитан — Ansar Zeinulla (S1). Свои задачи недели — в `S1.md`; здесь только то, что делаете за всю команду.

---

## понедельник 2 ноября утром — открыть неделю

**1. Влейте week-07 в `main`,** если ещё не сделали в воскресенье:

- GitHub → **Pull requests** → **New pull request**
- base: `main`, compare: `week-07`, заголовок `Close W7`
- апрув не нужен — вы администратор; вливаете, когда CI зелёный

```bash
git checkout main && git pull
git tag week-07-done && git push origin week-07-done
```

**2. Создайте ветку недели** — без неё никто не начнёт:

```bash
git checkout main && git pull
git checkout -b week-08 && git push -u origin week-08
```

**3. Заведите в Jira все 13 задач** в колонку `To Do`. Название копируйте дословно, исполнителя ставьте по таблице:

| Задача | Исполнитель | Тип | Эпик | Название |
| --- | --- | --- | --- | --- |
| `BF-B5` | S1 · Ansar Zeinulla | Bug | — | Stop sharing notification state on the server |
| `BF-B6` | S1 · Ansar Zeinulla | Bug | — | Make the seed re-runnable after a purchase |
| `BF-B7` | S2 · Alibi Takhtanov | Bug | — | Give a suspended event its own code |
| `BF-B8` | S2 · Alibi Takhtanov | Bug | — | Name the refund conflict |
| `BF-B9` | S4 · Alinur Burlybayev | Bug | — | Name the reversal conflict |
| `BF-B10` | S5 · Olzhas Nurseit | Bug | — | Name the check-in reversal conflict |
| `BF-77` | S1 · Ansar Zeinulla | Story | `BF-E1` | Count text limits in the remaining handlers |
| `BF-78` | S2 · Alibi Takhtanov | Story | `BF-E5` | Cut Cyrillic text by characters on the ticket |
| `BF-79` | S3 · Abylay Otaubay | Story | `BF-E11` | Localise the remaining attendee screens |
| `BF-80` | S4 · Alinur Burlybayev | Story | `BF-E9` | Show reserved stock |
| `BF-81` | S4 · Alinur Burlybayev | Story | `BF-E11` | Localise the remaining dashboard screens |
| `BF-82` | S5 · Olzhas Nurseit | Story | `BF-E12` | Add the Phase 8, 9 and 10 specifications |
| `BF-83` | S5 · Olzhas Nurseit | Story | `BF-E6` | Point the scanner at the queue |

Задачи заводите **до** того, как кто-то начал работать: по датам в Jira это видно.

---

## понедельник 2 ноября — пятница 6 ноября — следить

Свои задачи делаете по `S1.md`. Каждый день пять минут:

- нет ли карточки в `In Progress` без единого коммита больше суток;
- не пишет ли кто-то, что ему нужен чужой файл — разрулите сразу.

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 2 ноября | S2 | `BF-78` Cut Cyrillic text by characters on the ticket |
| понедельник 2 ноября | S4 | `BF-80` Show reserved stock |
| вторник 3 ноября | S2 | `BF-B7` Give a suspended event its own code |
| вторник 3 ноября | S4 | `BF-81` Localise the remaining dashboard screens |
| вторник 3 ноября | S5 | `BF-82` Add the Phase 8, 9 and 10 specifications |
| среда 4 ноября | S1 | `BF-B6` Make the seed re-runnable after a purchase |
| среда 4 ноября | S2 | `BF-B8` Name the refund conflict |
| среда 4 ноября | S4 | `BF-B9` Name the reversal conflict |
| среда 4 ноября | S5 | `BF-83` Point the scanner at the queue |
| четверг 5 ноября | S1 | `BF-77` Count text limits in the remaining handlers |
| четверг 5 ноября | S5 | `BF-B10` Name the check-in reversal conflict |
| пятница 6 ноября | S1 | `BF-B5` Stop sharing notification state on the server |
| пятница 6 ноября | S3 | `BF-79` Localise the remaining attendee screens |

---

## пятница 6 ноября вечером — приходят PR

Все пятеро открывают PR в `week-08`, по одному на задачу. К утру у вас должно быть **13 PR**. Не хватает — напишите тому, чей нет.

---

## суббота 7 ноября — воскресенье 8 ноября — принять PR

### Порядок — строго сверху вниз

Файлы у всех разные, поэтому конфликтов слияния не будет. Но проверки (CI) зависят друг от друга: хендлеры не пройдут тесты без схемы, клиент — без маршрутов. Поэтому вливаете по порядку:

| № | Слой | Задача | Кто |
| --- | --- | --- | --- |
| 1 | обычные задачи | `BF-78 Cut Cyrillic text by characters on the ticket` | S2 |
| 2 | обычные задачи | `BF-80 Show reserved stock` | S4 |
| 3 | обычные задачи | `BF-81 Localise the remaining dashboard screens` | S4 |
| 4 | обычные задачи | `BF-82 Add the Phase 8, 9 and 10 specifications` | S5 |
| 5 | обычные задачи | `BF-83 Point the scanner at the queue` | S5 |
| 6 | обычные задачи | `BF-B7 Give a suspended event its own code` | S2 |
| 7 | обычные задачи | `BF-B8 Name the refund conflict` | S2 |
| 8 | обычные задачи | `BF-B9 Name the reversal conflict` | S4 |
| 9 | обычные задачи | `BF-B10 Name the check-in reversal conflict` | S5 |
| 10 | маршруты API | `BF-77 Count text limits in the remaining handlers` | S1 |
| 11 | маршруты API | `BF-B5 Stop sharing notification state on the server` | S1 |
| 12 | типы и клиент API | `BF-79 Localise the remaining attendee screens` | S3 |
| 13 | интерфейс — последним | `BF-B6 Make the seed re-runnable after a purchase` | S1 |

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

## воскресенье 8 ноября вечером — закрыть неделю

**1. Все 13 задач в `Done`?** Если нет — не закрываете, выясняете.

**2. Влейте `week-08` в `main`:** GitHub → **New pull request** → base `main`, compare `week-08` → **Create** → **Merge** (апрув не нужен).

**3. Тег:**

```bash
git checkout main && git pull
git tag week-08-done && git push origin week-08-done
```

**4. Контрольная точка.** После W8 продуктовый код должен совпадать с эталоном — это цель восьми недель разработки.

Следующая неделя — `weeks/week-09-verification/captain.md`, раздел «открыть неделю».

---

## Чего капитан не делает

- **Не вливает свой PR с красным CI.** Права позволяют, правило — нет.
- **Не пушит в `main` напрямую.** Только через PR недели.
- **Не чинит чужой код в своей ветке.** Возвращает PR автору.
- **Не закрывает неделю с задачами вне `Done`.**

