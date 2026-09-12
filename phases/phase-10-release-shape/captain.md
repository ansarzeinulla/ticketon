# Капитан · Ф10 — Нынешняя версия

**понедельник 16 ноября — воскресенье 22 ноября** · задач в фазе: 15

Капитан — это S1. Всё, что ниже, кроме своих задач, делает он.

---

## воскресенье 15 ноября — открыть фазу

**1. Закройте phase-09.** Убедитесь, что все её PR влиты, и переведите оставшиеся задачи в `Done`.

**2. Влейте phase-09 в `main`:**

- GitHub → Pull requests → **New pull request**
- base: `main`, compare: `phase-09`
- заголовок: `Close Ф9`
- любой другой человек апрувит, вы вливаете

**3. Поставьте тег:**

```bash
git checkout main && git pull
git tag phase-09-done && git push origin phase-09-done
```

**4. Создайте ветку этой фазы:**

```bash
git checkout -b phase-10
git push -u origin phase-10
```

Или одной командой — она же создаст рабочую папку и проверит, что прошлая фаза действительно закрыта:

```bash
bash plan/tools/new-phase.sh 10
```

**5. Проверьте, что заведены все 15 задач заведены.** Пока не все — фаза не начинается.

| Задача | Кто | Название |
| --- | --- | --- |
| `BF-80` | S1 | Count text limits in characters, not bytes |
| `BF-81` | S1 | Count text limits in the remaining handlers |
| `BF-82` | S2 | Count text limits in commerce handlers |
| `BF-83` | S2 | Cut Cyrillic text by characters on the ticket |
| `BF-84` | S3 | Localise the remaining attendee screens |
| `BF-85` | S4 | Show reserved stock |
| `BF-86` | S4 | Localise the remaining dashboard screens |
| `BF-87` | S5 | Add the Phase 8, 9 and 10 specifications |
| `BF-88` | S5 | Point the scanner at the queue |
| `BF-B10` | S5 | Name the check-in reversal conflict |
| `BF-B5` | S1 | Stop sharing notification state on the server |
| `BF-B6` | S1 | Make the seed re-runnable after a purchase |
| `BF-B7` | S2 | Give a suspended event its own code |
| `BF-B8` | S2 | Name the refund conflict |
| `BF-B9` | S4 | Name the reversal conflict |

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
| понедельник 16 ноября | S2 | `BF-82` Count text limits in commerce handlers |
| понедельник 16 ноября | S4 | `BF-85` Show reserved stock |
| вторник 17 ноября | S1 | `BF-B6` Make the seed re-runnable after a purchase |
| вторник 17 ноября | S2 | `BF-83` Cut Cyrillic text by characters on the ticket |
| вторник 17 ноября | S4 | `BF-86` Localise the remaining dashboard screens |
| вторник 17 ноября | S5 | `BF-87` Add the Phase 8, 9 and 10 specifications |
| среда 18 ноября | S1 | `BF-80` Count text limits in characters, not bytes |
| среда 18 ноября | S2 | `BF-B7` Give a suspended event its own code |
| среда 18 ноября | S4 | `BF-B9` Name the reversal conflict |
| среда 18 ноября | S5 | `BF-88` Point the scanner at the queue |
| четверг 19 ноября | S1 | `BF-81` Count text limits in the remaining handlers |
| четверг 19 ноября | S2 | `BF-B8` Name the refund conflict |
| четверг 19 ноября | S5 | `BF-B10` Name the check-in reversal conflict |
| пятница 20 ноября | S1 | `BF-B5` Stop sharing notification state on the server |
| пятница 20 ноября | S3 | `BF-84` Localise the remaining attendee screens |

---

## суббота 21 ноября — добить ревью

Нового кода в этот день нет. Цель к вечеру:

- все 15 PR влиты в `phase-10`;
- CI на ветке фазы зелёный;
- ни одной задачи вне `Done` или `In Review`.

---

## воскресенье 22 ноября — закрыть фазу

**1. Соберите ветку фазы:**

```bash
git checkout phase-10 && git pull
make api-check && make web-check
```

**2. PR `phase-10` → `main`.** Апрувит любой другой; вливаете вы.

**3. Тег:**

```bash
git checkout main && git pull
git tag phase-10-done && git push origin phase-10-done
```

**4. Контрольная точка.** После этой фазы продуктовый код обязан совпадать с эталоном:

```bash
git diff phase-10-done main -- . ':!plan' ':!web/e2e' ':!docs' \
  ':!web/vitest.*' ':!web/playwright.config.ts' \
  ':!api/internal/api/criteria_test.go' ':!api/internal/api/phase1*_test.go'
```

Пусто — значит дошли. Не пусто — разбирайтесь до Ф11.

**5. Тут же откройте Ф11.** Каждый заводит свои задачи: `BF-89`…`BF-96`. Инструкции — в `phases/phase-11-verification/`.

---

## Чего капитан делать не должен

- **Вливать свой PR с красным CI.** Права позволяют, правило — нет. Это единственное место, где вас никто не остановит, кроме вас самих.
- **Заводить задачи за других.** Каждый заводит свои — иначе непонятно, кто что понял.
- **Пушить в `main`.** Никогда и ни при каких обстоятельствах.
- **Чинить чужой код в своей ветке.** Пишете владельцу.

