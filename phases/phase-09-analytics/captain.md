# Капитан · Ф9 — Аналитика, кампании, i18n

**понедельник 9 ноября — воскресенье 15 ноября** · задач в фазе: 10

Капитан — это S1. Всё, что ниже, кроме своих задач, делает он.

---

## воскресенье 8 ноября — открыть фазу

**1. Закройте phase-08.** Убедитесь, что все её PR влиты, и переведите оставшиеся задачи в `Done`.

**2. Влейте phase-08 в `main`:**

- GitHub → Pull requests → **New pull request**
- base: `main`, compare: `phase-08`
- заголовок: `Close Ф8`
- любой другой человек апрувит, вы вливаете

**3. Поставьте тег:**

```bash
git checkout main && git pull
git tag phase-08-done && git push origin phase-08-done
```

**4. Создайте ветку этой фазы:**

```bash
git checkout -b phase-09
git push -u origin phase-09
```

Или одной командой — она же создаст рабочую папку и проверит, что прошлая фаза действительно закрыта:

```bash
bash plan/tools/new-phase.sh 9
```

**5. Проверьте, что заведены все 10 задач заведены.** Пока не все — фаза не начинается.

| Задача | Кто | Название |
| --- | --- | --- |
| `BF-71` | S1 | Add campaign tables |
| `BF-72` | S2 | Create campaigns and promo codes |
| `BF-73` | S2 | Apply and cap discounts atomically |
| `BF-74` | S3 | Add the English dictionary |
| `BF-75` | S3 | Localise the attendee pages |
| `BF-76` | S3 | Add the language negotiation dependency |
| `BF-77` | S4 | Compute analytics from operational rows |
| `BF-78` | S4 | Localise the dashboard |
| `BF-79` | S5 | Add the Phase 7 specification |
| `BF-B4` | S4 | Read the GA4 id from a server-safe module |

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
| понедельник 9 ноября | S1 | `BF-71` Add campaign tables |
| понедельник 9 ноября | S2 | `BF-72` Create campaigns and promo codes |
| вторник 10 ноября | S2 | `BF-73` Apply and cap discounts atomically |
| вторник 10 ноября | S4 | `BF-77` Compute analytics from operational rows |
| среда 11 ноября | S3 | `BF-76` Add the language negotiation dependency |
| среда 11 ноября | S4 | `BF-78` Localise the dashboard |
| среда 11 ноября | S5 | `BF-79` Add the Phase 7 specification |
| четверг 12 ноября | S3 | `BF-75` Localise the attendee pages |
| четверг 12 ноября | S4 | `BF-B4` Read the GA4 id from a server-safe module |
| пятница 13 ноября | S3 | `BF-74` Add the English dictionary |

---

## суббота 14 ноября — добить ревью

Нового кода в этот день нет. Цель к вечеру:

- все 10 PR влиты в `phase-09`;
- CI на ветке фазы зелёный;
- ни одной задачи вне `Done` или `In Review`.

---

## воскресенье 15 ноября — закрыть фазу

**1. Соберите ветку фазы:**

```bash
git checkout phase-09 && git pull
make api-check && make web-check
```

**2. PR `phase-09` → `main`.** Апрувит любой другой; вливаете вы.

**3. Тег:**

```bash
git checkout main && git pull
git tag phase-09-done && git push origin phase-09-done
```

**4. Тут же откройте Ф10.** Каждый заводит свои задачи: `BF-80`…`BF-B9`. Инструкции — в `phases/phase-10-release-shape/`.

---

## Чего капитан делать не должен

- **Вливать свой PR с красным CI.** Права позволяют, правило — нет. Это единственное место, где вас никто не остановит, кроме вас самих.
- **Заводить задачи за других.** Каждый заводит свои — иначе непонятно, кто что понял.
- **Пушить в `main`.** Никогда и ни при каких обстоятельствах.
- **Чинить чужой код в своей ветке.** Пишете владельцу.

