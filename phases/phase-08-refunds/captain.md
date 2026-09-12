# Капитан · Ф8 — Возвраты, поддержка, админ

**понедельник 2 ноября — воскресенье 8 ноября** · задач в фазе: 14

Капитан — это S1. Всё, что ниже, кроме своих задач, делает он.

---

## воскресенье 1 ноября — открыть фазу

**1. Закройте phase-07.** Убедитесь, что все её PR влиты, и переведите оставшиеся задачи в `Done`.

**2. Влейте phase-07 в `main`:**

- GitHub → Pull requests → **New pull request**
- base: `main`, compare: `phase-07`
- заголовок: `Close Ф7`
- любой другой человек апрувит, вы вливаете

**3. Поставьте тег:**

```bash
git checkout main && git pull
git tag phase-07-done && git push origin phase-07-done
```

**4. Создайте ветку этой фазы:**

```bash
git checkout -b phase-08
git push -u origin phase-08
```

Или одной командой — она же создаст рабочую папку и проверит, что прошлая фаза действительно закрыта:

```bash
bash plan/tools/new-phase.sh 8
```

**5. Проверьте, что заведены все 14 задач заведены.** Пока не все — фаза не начинается.

| Задача | Кто | Название |
| --- | --- | --- |
| `BF-60` | S1 | Add refund, support and report tables |
| `BF-61` | S1 | Add the report queue and settings |
| `BF-62` | S1 | Give every conflict its own error code |
| `BF-63` | S2 | Refund an order atomically |
| `BF-64` | S2 | Expose the rows the admin search needs |
| `BF-65` | S3 | Add the support widget to the order page |
| `BF-66` | S3 | Point the client at the proxy |
| `BF-67` | S4 | Open a support case with context |
| `BF-68` | S4 | Add the admin portal |
| `BF-69` | S5 | Refuse a refunded ticket at the gate |
| `BF-70` | S5 | Add the Phase 6 specification |
| `BF-B1` | S5 | Keep the check-in record when a ticket is voided |
| `BF-B2` | S4 | Move the session into an httpOnly cookie |
| `BF-B3` | S4 | Rename middleware to proxy for Next 16 |

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
| понедельник 2 ноября | S1 | `BF-60` Add refund, support and report tables |
| понедельник 2 ноября | S2 | `BF-63` Refund an order atomically |
| вторник 3 ноября | S2 | `BF-64` Expose the rows the admin search needs |
| вторник 3 ноября | S4 | `BF-67` Open a support case with context |
| вторник 3 ноября | S5 | `BF-69` Refuse a refunded ticket at the gate |
| среда 4 ноября | S4 | `BF-68` Add the admin portal |
| среда 4 ноября | S5 | `BF-70` Add the Phase 6 specification |
| четверг 5 ноября | S1 | `BF-61` Add the report queue and settings |
| четверг 5 ноября | S3 | `BF-66` Point the client at the proxy |
| четверг 5 ноября | S4 | `BF-B2` Move the session into an httpOnly cookie |
| четверг 5 ноября | S5 | `BF-B1` Keep the check-in record when a ticket is voided |
| пятница 6 ноября | S1 | `BF-62` Give every conflict its own error code |
| пятница 6 ноября | S3 | `BF-65` Add the support widget to the order page |
| пятница 6 ноября | S4 | `BF-B3` Rename middleware to proxy for Next 16 |

---

## суббота 7 ноября — добить ревью

Нового кода в этот день нет. Цель к вечеру:

- все 14 PR влиты в `phase-08`;
- CI на ветке фазы зелёный;
- ни одной задачи вне `Done` или `In Review`.

---

## воскресенье 8 ноября — закрыть фазу

**1. Соберите ветку фазы:**

```bash
git checkout phase-08 && git pull
make api-check && make web-check
```

**2. PR `phase-08` → `main`.** Апрувит любой другой; вливаете вы.

**3. Тег:**

```bash
git checkout main && git pull
git tag phase-08-done && git push origin phase-08-done
```

**4. Тут же откройте Ф9.** Каждый заводит свои задачи: `BF-71`…`BF-B4`. Инструкции — в `phases/phase-09-analytics/`.

---

## Чего капитан делать не должен

- **Вливать свой PR с красным CI.** Права позволяют, правило — нет. Это единственное место, где вас никто не остановит, кроме вас самих.
- **Заводить задачи за других.** Каждый заводит свои — иначе непонятно, кто что понял.
- **Пушить в `main`.** Никогда и ни при каких обстоятельствах.
- **Чинить чужой код в своей ветке.** Пишете владельцу.

