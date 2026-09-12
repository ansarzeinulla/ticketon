# Капитан · Ф11 — Проверка

**понедельник 23 ноября — четверг 26 ноября** · задач в фазе: 8

Капитан — это S1. Всё, что ниже, кроме своих задач, делает он.

---

## воскресенье 22 ноября — открыть фазу

**1. Закройте phase-10.** Убедитесь, что все её PR влиты, и переведите оставшиеся задачи в `Done`.

**2. Влейте phase-10 в `main`:**

- GitHub → Pull requests → **New pull request**
- base: `main`, compare: `phase-10`
- заголовок: `Close Ф10`
- любой другой человек апрувит, вы вливаете

**3. Поставьте тег:**

```bash
git checkout main && git pull
git tag phase-10-done && git push origin phase-10-done
```

**4. Создайте ветку этой фазы:**

```bash
git checkout -b phase-11
git push -u origin phase-11
```

Или одной командой — она же создаст рабочую папку и проверит, что прошлая фаза действительно закрыта:

```bash
bash plan/tools/new-phase.sh 11
```

**5. Проверьте, что заведены все 8 задач заведены.** Пока не все — фаза не начинается.

| Задача | Кто | Название |
| --- | --- | --- |
| `BF-89` | S1 | Say "required" instead of "too short" |
| `BF-90` | S2 | Refuse to publish an event that has ended |
| `BF-91` | S3 | Close the localisation gaps |
| `BF-92` | S3 | Add the test runner dependencies |
| `BF-93` | S4 | Make the money card answer the tier filter |
| `BF-94` | S5 | Run the phase specifications and record results |
| `BF-95` | S5 | Correct the specifications after the run |
| `BF-96` | S5 | Add the Postman collection |

---

## Рабочие дни — смотреть PR

**Все PR в команде смотрите и вливаете вы.** Больше никто. Поэтому открытый PR не должен висеть дольше суток — иначе встанет вся фаза.

### Как смотреть один PR

Открываете PR, вкладку **«Files changed»**, и проверяете четыре вещи:

1. **Файлы только из зоны автора.** Сверяйтесь с `plan/ownership.map`. Тронул чужой файл — возвращаете, это причина отклонить сразу.
2. **Коммит один**, и его сообщение дословно совпадает с названием задачи.
3. **CI зелёный** — галочка внизу PR.
4. Код делает то, что написано в названии, и не больше.

Дальше **«Review changes»** справа сверху → **Approve** (или **Request changes** с замечаниями) → **Submit review**.

Если одобрили — сразу вливаете: **«Merge pull request»** → **«Confirm merge»**. Ветку GitHub удалит сам. После слияния **переведите задачу автора в `Done`** — автор этого не делает.

### Свои собственные PR

Свой PR вы апрувить не сможете — GitHub не даёт апрувить себя. Поэтому в настройках ветки **не включена** галочка *Do not allow bypassing the above settings*: как администратор репозитория вы вливаете свой PR без апрува, но **только когда CI зелёный**. Остальным эта лазейка недоступна.

### Каждый день пять минут

- влить всё, что пришло за сутки;
- посмотреть, нет ли карточки в `In Progress` без единого коммита;
- посмотреть, нет ли задачи, которая должна была уйти вчера, но не ушла.

Кто что должен сдать по дням:

| День | Кто | Задача |
| --- | --- | --- |
| понедельник 23 ноября | S2 | `BF-90` Refuse to publish an event that has ended |
| понедельник 23 ноября | S4 | `BF-93` Make the money card answer the tier filter |
| понедельник 23 ноября | S5 | `BF-94` Run the phase specifications and record results |
| вторник 24 ноября | S1 | `BF-89` Say "required" instead of "too short" |
| вторник 24 ноября | S3 | `BF-92` Add the test runner dependencies |
| вторник 24 ноября | S5 | `BF-95` Correct the specifications after the run |
| среда 25 ноября | S3 | `BF-91` Close the localisation gaps |
| среда 25 ноября | S5 | `BF-96` Add the Postman collection |
---

## четверг 26 ноября — закрыть фазу

**1. Соберите ветку фазы:**

```bash
git checkout phase-11 && git pull
make api-check && make web-check
```

**2. PR `phase-11` → `main`.** Апрувит любой другой; вливаете вы.

**3. Тег:**

```bash
git checkout main && git pull
git tag phase-11-done && git push origin phase-11-done
```

**4. Тут же откройте Ф12.** Каждый заводит свои задачи: `BF-100`…`BF-99`. Инструкции — в `phases/phase-12-release/`.

---

## Чего капитан делать не должен

- **Вливать свой PR с красным CI.** Права позволяют, правило — нет. Это единственное место, где вас никто не остановит, кроме вас самих.
- **Заводить задачи за других.** Каждый заводит свои — иначе непонятно, кто что понял.
- **Пушить в `main`.** Никогда и ни при каких обстоятельствах.
- **Чинить чужой код в своей ветке.** Пишете владельцу.

