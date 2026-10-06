# Фазы Ф0…Ф12

Тринадцать фаз от пустого репозитория до нынешней версии. Ф0 — до 14 сентября,
остальные равномерно ложатся на 11 недель до 28 ноября.

## Как читать

Каждая фаза — таблица **коммитов**, а не список файлов. Операции в последней колонке:

| Знак | Что значит |
| --- | --- |
| **A** | файл создан — и создан **маленьким**, в том виде, какой был в ту фазу |
| **M** | существующий файл изменён — таких строк должно быть больше всего |
| **D** | файл удалён |
| **R** | файл переименован (`git mv`) |

Если в фазе одни **A** — фаза спланирована неправильно. Найдите, какой существующий
файл должна была затронуть работа.

Владелец в колонке «кто» обязан совпадать с `plan/ownership.map`. Проверяется
скриптом `plan/tools/check-plan.py`.

Исходники v1 для переписываемых файлов — в `plan/file-histories.md`.

## Ритм фазы

```
Jira до фазы  →  ветки и коммиты  →  PR + ревью  →  merge в main  →  Jira после фазы
```

Порядок merge внутри фазы фиксирован, иначе ветка интегратора не соберётся:

```
1. S1 схема → 2. S2/S4/S5 store+handlers → 3. S1 маршруты → 4. S3 types+api.ts → 5. UI
```

---

## W0 — Discovery (11–13 сентября)

**Цель:** решили, что делаем и на чём. Кода почти нет.

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-1 Record the delivery plan` | **A** `plan/**` |
| S2 | `BF-2 Sketch the API contract` | **A** `api/README.md` *(только описание, кода нет)* |
| S3 | `BF-3 Sketch the web routes` | **A** `web/README.md` |
| S5 | `BF-4 Add the SRS and repository skeleton` | **A** `bilet.md`, **A** `README.md` *(пока одна строка)*, **A** `.gitignore`, **A** `.env.example`, **A** `docs/decisions.md` *(журнал решений)* |
| S5 | `BF-5 Describe the product and the stack` | **M** `README.md` *(`-1/+48`)* |

В Ф0 коммитят четверо: за три дня discovery появляется пять файлов, и делить их
на пятерых — выдумка. S4 в этой фазе участвует в обсуждении и подключается с Ф1.
То же в Ф12: релиз занимает два дня.

---

## W1 — Скелет (14–20 сентября)

**Цель:** проект поднимается на каждой машине. БД настоящая, приложения пустые.

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-6 Add extensions and the users table` | **A** `db/init/01_extensions.sql`, **A** `db/init/02_schema.sql` *(1 таблица, ~40 строк)*, **A** `db/init/03_sample.sql` *(пара строк, чтобы было что выбрать)*, **A** `db/tests/{run_tests.sh,_helpers.sql,_fixture.sql,01_environment.sql,02_tables.sql}`, **A** `db/tests/00_smoke.sql` *(проверка «база вообще поднялась»)* |
| S1 | `BF-7 Add config and the error envelope` | **A** `api/go.mod`, **A** `api/go.sum`, **A** `api/cmd/api/main.go`, **A** `api/internal/config/config.go`, **A** `api/internal/httpx/httpx.go`, **A** `api/internal/config/env.go` *(чтение переменных окружения врозь)* |
| S1 | `BF-8 Add the pool and a health route` | **A** `api/internal/database/database.go`, **A** `api/internal/store/store.go`, **A** `api/internal/api/server.go` *(1 маршрут, ~40 строк)*, **A** `api/internal/api/handlers_dev.go` *(дев-роут с дампом конфига)*, **A** `api/internal/api/handlers_health.go` *(одна ручка /health отдельным файлом)*, **A** `api/internal/config/config_test.go`, **M** `api/internal/config/config.go` |
| S2 | `BF-9 Document the API layout` | **M** `api/README.md` |
| S3 | `BF-10 Scaffold Next.js` | **A** `web/package.json`, **A** `web/package-lock.json` *(8 879 строк, целиком)*, **A** `web/{next.config.ts,tsconfig.json,eslint.config.mjs,postcss.config.mjs}`, **A** `web/src/lib/mock-data.ts` *(каталог рисуется на заглушках, пока нет API)*, **A** `web/src/app/{layout.tsx,page.tsx,globals.css}`, **A** `web/src/components/ui/{button,alert,field}.tsx`, **A** `web/src/components/ui/spinner.tsx` *(ручной спиннер загрузки)*, **A** `web/src/app/{error,not-found}.tsx`, **A** `web/src/app/favicon.ico`, **A** `web/{.gitignore,AGENTS.md,CLAUDE.md}` |
| S4 | `BF-11 Add the signed-in shell` | **A** `web/src/app/(app)/layout.tsx`, **A** `web/src/components/site-header.tsx`, **A** `web/src/app/api/health/route.ts` *(пинг бэкенда из браузера)* |
| S5 | `BF-12 Add Postgres to compose` | **A** `docker-compose.yml`, **M** `.env.example`, **A** `docs/schema-draft.md` *(черновик схемы в markdown)*, **A** `Makefile` *(~30 строк: up, down, psql)*, **A** `docs/api-notes.md` *(черновые заметки по эндпоинтам)* |
| S5 | `BF-13 Add CI` | **A** `.github/workflows/ci.yml`, **M** `Makefile` *(+цели `api-check`, `web-check`, `test` — CI и люди запускают одно и то же)*, **M** `README.md`, **A** `.github/workflows/lint.yml` *(отдельный workflow для линтера)*, **A** `api/Dockerfile`, **A** `api/.dockerignore`, **A** `web/Dockerfile`, **A** `web/.dockerignore` |

---

## W2 — Идентичность (21–27 сентября)

**Цель:** человек регистрируется, входит, восстанавливает пароль.

Здесь выходят **v1** двух файлов, которые перепишут только в Ф10: `auth/password.go`
с ограничением bcrypt в 72 байта и `api/validate.go` со счётом длин в байтах.
Это не ошибка, которую надо обойти, — это причина, по которой существует Ф10.

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-14 Hash passwords with bcrypt` | **A** `api/internal/auth/password.go`, **A** `auth/password_test.go`, **A** `api/internal/auth/token.go`, **A** `auth/token_test.go`, **A** `api/internal/httpx/context.go`, **A** `api/internal/api/middleware.go`, **M** `db/init/02_schema.sql`, **A** `api/internal/store/users.go` *(2 функции)*, **D** `api/internal/config/env.go` *(слито в `config.go`)* |
| S1 | `BF-15 Reject duplicate emails with 409` | **A** `api/internal/api/handlers_auth.go`, **A** `api/internal/api/validate.go`, **M** `server.go`, **D** `api/internal/api/handlers_health.go` *(ручка переехала в `server.go`)*, **M** `handlers_auth.go`, **M** `store/users.go`, **A** `api/internal/api/{main_test,auth_test}.go`, **A** `api/internal/api/fixtures_test.go` *(фикстуры тестов лежали отдельно)*, **M** `db/init/02_schema.sql`, **A** `store/tokens.go`, **A** `handlers_account.go`, **A** `api/internal/email/{email,templates}.go`, **A** `api/internal/email/console.go` *(печать письма в консоль)* |
| S1 | `BF-16 Cover users, tokens and mail` | **R** `db/tests/02_tables.sql` → `db/tests/02_schema_objects.sql`, **M** `store/tokens.go`, **A** `api/internal/api/account_test.go`, **A** `db/tests/{03_user_crud.sql,04_constraints.sql}`, **A** `api/internal/store/store_test.go`, **A** `api/internal/email/email_test.go`, **D** `db/tests/00_smoke.sql` *(вытеснен полноценным набором)*, **M** `db/tests/02_schema_objects.sql` *(набор проверяет уже не только таблицы, но и enum'ы со связями)*, **M** `api/internal/httpx/httpx.go`, **M** `api/internal/database/database.go` *(конверт ошибок и пул подросли под аутентификацию)* |
| S2 | `BF-17 Document the authentication endpoints` | **M** `api/README.md` *(+раздел `/auth`, заглушка из Ф1 **удалена**)* |
| S3 | `BF-18 Add the API client` | **A** `web/src/lib/api.ts` *(~80 строк, прямой `:8080`)*, **A** `web/src/lib/types.ts`, **A** `web/src/lib/format.ts` *(самодельные форматтеры денег и дат)*, **A** `web/src/lib/env.ts` *(адрес API читался вручную)* |
| S3 | `BF-19 Add reset and verify pages` | **A** `web/src/app/(auth)/{layout,login/page,register/page}.tsx`, **M** `web/src/lib/api.ts`, **A** `web/src/app/(auth)/{forgot-password,reset-password,verify-email}/page.tsx`, **M** `web/src/lib/types.ts`, **A** `web/src/app/(auth)/{login/login-form,register/register-form,forgot-password/forgot-password-form,reset-password/reset-password-form,verify-email/verify-email-view}.tsx`, **M** `web/package.json`, **M** `web/package-lock.json` *(формы входа потребовали валидации)*, **M** `web/README.md` *(появились страницы входа)* |
| S4 | `BF-20 Keep the session in the browser` | **A** `web/src/lib/session.ts`, **A** `web/src/lib/auth-context.tsx`, **A** `web/src/middleware.ts` |
| S5 | `BF-21 Add the Phase 1 specification` | **A** `1.md`, **M** `Makefile` *(`make test` и `make db-test`)* |
| S5 | `BF-22 Scaffold the Expo app and sign in on the device` | **A** `mobile/package.json`, **A** `mobile/package-lock.json` *(7 124 строки, целиком)*, **A** `mobile/app.json`, **A** `mobile/assets/**`, **A** `mobile/{.gitignore,.env.example,tsconfig.json,README.md,LICENSE,AGENTS.md,CLAUDE.md}`, **A** `mobile/lib/{config,theme,types}.ts`, **A** `mobile/lib/auth-context.tsx`, **A** `mobile/lib/{api,session}.ts`, **A** `mobile/lib/storage.ts` *(обёртка над AsyncStorage)*, **A** `mobile/app/{_layout,login}.tsx` |

> `db/tests/**` принадлежит S1 по карте, но прогон и спецификации — зона S5. Чтобы не
> нарушать правило одного владельца, **тесты схемы пишет S1**, а S5 владеет `1.md`…`10.md`
> и запускает их. Если удобнее иначе — поменяйте `ownership.map`, а не договаривайтесь
> устно.

---

## W3 — События, каталог и регистрация (28 сентября – 4 октября)

**Цель:** организатор создаёт событие, добавляет тарифы, публикует.

коммит с тем же названием.

**Цель:** зритель видит событие и бронирует бесплатный билет.

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-23 Seed a demo organizer and event` | **M** `db/init/02_schema.sql` *(+2 таблицы)*, **A** `db/seed/01_demo_data.sql`, **D** `db/init/03_sample.sql` *(вытеснен `db/seed/01_demo_data.sql`)*, **M** `api/internal/api/server.go` *(интеграция, ветка от main после S2)* |
| S2 | `BF-25 Derive a slug from the title` | **A** `store/events.go`, **A** `handlers_events.go`, **A** `store/slug.go`, **A** `store/slug_test.go`, **A** `handlers_public.go` |
| S2 | `BF-26 Manage ticket types` | **A** `store/ticket_types.go`, **A** `handlers_ticket_types.go`, **A** `api/internal/api/ticket_types_test.go`, **A** `api/internal/api/handlers_uploads.go`, **M** `store/events.go`, **M** `handlers_events.go`, **A** `api/internal/api/events_test.go`, **M** `api/README.md` *(события и типы билетов)* |
| S3 | `BF-28 Add the formatting dependencies` | **M** `web/src/lib/types.ts`, **M** `web/src/lib/api.ts`, **A** `web/src/components/ui/toast.tsx` *(всплывающие уведомления)*, **A** `web/src/components/ui/status-badge.tsx`, **D** `web/src/lib/env.ts` *(вытеснен переменными `NEXT_PUBLIC_*`)*, **D** `web/src/lib/format.ts` *(вытеснен `money.ts` и `datetime.ts`)*, **M** `web/package.json`, **M** `web/package-lock.json`, **M** `web/tsconfig.json` *(деньги и даты на витрине)*, **M** `web/.gitignore` *(появился `.next`)* |
| S4 | `BF-31 Move the create form to the client` | **A** `web/src/app/(app)/events/new/page.tsx`, **A** `web/src/components/event-form-fields.tsx`, **A** `web/src/app/(app)/events/new/create-event-form.tsx`, **A** `web/src/components/image-upload.tsx`, **M** `(app)/events/new/page.tsx` |
| S4 | `BF-32 Add the organizer dashboard` | **A** `web/src/app/(app)/dashboard/page.tsx`, **A** `web/src/app/(app)/dashboard/events/[id]/page.tsx`, **M** `dashboard/events/[id]/page.tsx`, **A** `web/src/components/ticket-type-manager.tsx`, **A** `web/src/app/(app)/dashboard/events/[id]/event-detail.tsx`, **A** `(app)/dashboard/events/[id]/edit/{page,edit-event-form}.tsx` |
| S5 | `BF-34 Add the Phase 2 specification` | **A** `2.md`, **M** `1.md` *(поправка по 409)*, **D** `docs/schema-draft.md` *(вытеснен самой схемой в `db/init`)*, **M** `Makefile` *(`make seed` поднимает демо-данные)* |
| S1 | `BF-24 Add order, ticket and attendee tables` | **M** `db/init/02_schema.sql` *(+5 таблиц)*, **M** `server.go`, **A** `db/tests/05_business_rules.sql` |
| S2 | `BF-27 Add the checkout transaction` | **A** `store/checkout.go`, **A** `handlers_checkout.go`, **A** `api/internal/store/inventory.go` *(остатки считались отдельным стором)*, **A** `api/internal/api/handlers_inventory.go` *(ручка остатков врозь)*, **A** `api/internal/api/checkout_test.go`, **M** `api/README.md` *(оформление и бесплатная регистрация)* |
| S3 | `BF-29 Add the public catalogue` | **A** `web/src/app/events/page.tsx`, **A** `web/src/components/event-card.tsx`, **M** `web/src/lib/api.ts`, **M** `web/src/lib/types.ts`, **D** `web/src/lib/mock-data.ts` *(вытеснен реальными вызовами `api.ts`)*, **A** `web/src/app/events/[slug]/page.tsx`, **A** `web/src/app/events/layout.tsx` |
| S3 | `BF-30 Add the order page` | **A** `web/src/components/checkout-dialog.tsx`, **A** `web/src/lib/money.ts`, **A** `web/src/lib/money.test.ts`, **A** `web/src/app/orders/[id]/page.tsx`, **M** `web/src/lib/api.ts`, **M** `web/src/lib/types.ts`, **A** `web/src/components/order-list.tsx` *(первый список заказов)*, **A** `web/src/app/events/[slug]/ticket-selector.tsx`, **A** `web/src/lib/{datetime.ts,datetime.test.ts}`, **M** `web/src/app/events/[slug]/page.tsx` *(`-52/+9`)* |
| S4 | `BF-33 List orders and attendees` | **A** `api/internal/store/guests.go`, **A** `handlers_attendees.go`, **A** `web/src/components/{order-manager,attendee-list}.tsx`, а не «guest» — приводим код к тексту)* |
| S5 | `BF-35 Add the Phase 3 specification` | **A** `3.md`, **M** `README.md` *(проект впервые проходится насквозь)* |
| S5 | `BF-36 List assigned events on the device` | **A** `mobile/app/index.tsx`, **A** `mobile/app/events.tsx`, **M** `mobile/lib/{api.ts,types.ts}` |

---

## W4 — Платные продажи и билеты (5–11 октября)

**Цель:** деньги. И перепродажи одного места быть не должно.

коммит с тем же названием.

**Цель:** билет становится вещью, которую можно распечатать.

**Первая крупная замена.** `ticketpdf/text.go` выходит во вторник с картой
транслитерации на 60 строк, а в четверг карта **удаляется** — приходит встроенный
шрифт. Два коммита, два дня, именно в этом порядке: удаление и есть смысл.

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-37 Add organizer profiles and money handling` | **M** `db/init/02_schema.sql` *(+3 таблицы)*, **A** `api/internal/store/profiles.go`, **A** `api/internal/api/handlers_profile.go`, **A** `api/internal/api/money.go`, **A** `api/internal/api/profile_test.go`, **M** `api/go.mod`, **M** `api/go.sum`, **M** `server.go` |
| S1 | `BF-38 Enforce role permissions` | **M** `api/internal/api/middleware.go`, **M** `server.go`, **M** `handlers_auth.go`, **D** `api/internal/api/handlers_dev.go` *(закрыт, когда появились роли)*, **M** `db/init/02_schema.sql` *(триггер)*, **A** `store/audit.go`, **A** `db/tests/07_temp_checks.sql` *(временные проверки триггеров)*, **M** `api/internal/api/main_test.go`, **A** `api/internal/testutil/testdb.go`, **D** `api/internal/api/fixtures_test.go` *(вытеснен `main_test.go` и `testutil`)*, **M** `db/init/01_extensions.sql` *(для блокировок понадобился `pgcrypto`)* |
| S2 | `BF-40 Add seat inventory` | **M** `store/checkout.go`, **D** `api/internal/api/handlers_inventory.go` *(ручка вложена в checkout)*, **D** `api/internal/store/inventory.go` *(логика вложена в `checkout.go` вместе с блокировкой строк)*, **M** `handlers_checkout.go`, **M** `checkout_test.go`, **A** `api/internal/store/seating.go`, **A** `api/internal/api/handlers_seating.go`, **A** `api/internal/api/seating_test.go`, **M** `api/internal/api/handlers_public.go` *(проверка роли добавляется в каждый хендлер организатора)*, **M** `api/README.md` *(платные продажи под чек-листом)* |
| S2 | `BF-41 Gate paid sales behind activation` | **A** `store/activation.go`, **A** `handlers_activation.go`, **M** `store/checkout.go`, **M** `handlers_checkout.go`, **M** `checkout_test.go`, **A** `api/internal/api/activation_test.go`, **A** `store/tickets.go`, **A** `handlers_tickets.go`, **A** `api/internal/api/tickets_test.go`, **A** `ticketpdf/ticketpdf_test.go`, **A** `ticketpdf/extract_test.go` |
| S3 | `BF-43 Show the activation notice` | **M** `web/src/app/events/[slug]/page.tsx`, **M** `web/src/lib/api.ts`, **M** `web/src/lib/types.ts` |
| S3 | `BF-44 Add the seat map and seated checkout` | **A** `web/src/components/{seat-map,seat-checkout,seated-purchase}.tsx`, **M** `web/src/lib/{api.ts,types.ts}` |
| S4 | `BF-46 Add the activation checklist` | **A** `web/src/components/activation-checklist.tsx`, **M** `dashboard/events/[id]/page.tsx`, **A** `web/src/app/(app)/dashboard/profile/{page,profile-form}.tsx` |
| S5 | `BF-48 Prove checkout cannot oversell` | **M** `3.md`, **A** `api/scripts/smoke_test.sh`, **M** `.github/workflows/ci.yml`, **M** `docker-compose.yml` *(SQL-набор гоняется на каждом пуше)*, **M** `Makefile` |
| S1 | `BF-39 Send the order confirmation` | **A** `store/notifications.go`, **A** `api/internal/api/notify.go`, **M** `email/templates.go`, **D** `api/internal/email/console.go` *(вытеснен `email.go` с транспортом)*, **M** `db/init/02_schema.sql`, **A** `api/internal/api/notifications_test.go` |
| S2 | `BF-42 Embed a Unicode font for Cyrillic` | **A** `api/internal/ticketpdf/{render,qr}.go`, **A** `ticketpdf/text.go`, **A** `ticketpdf/fonts.go`, **A** `api/internal/ticketpdf/fonts/{DejaVuSansCondensed.ttf,DejaVuSansCondensed-Bold.ttf,LICENSE-fpdf.txt}` *(1.3 МБ)*, **M** `render.go`, **A** `api/internal/calendar/{ics.go,ics_test.go}`, **A** `api/internal/api/handlers_calendar.go`, **M** `api/README.md` *(QR и PDF)* |
| S3 | `BF-45 Link the printable PDF` | **A** `web/src/components/ticket-card.tsx`, **M** `web/src/app/orders/[id]/page.tsx`, **D** `web/src/components/ui/spinner.tsx` *(вытеснен состояниями Suspense)*, **M** `web/src/lib/api.ts`, **M** `web/src/lib/types.ts`, **M** `web/package.json`, **M** `web/package-lock.json` *(QR на странице заказа)* |
| S4 | `BF-47 Show ticket status per order` | **M** `web/src/components/order-manager.tsx` |
| S5 | `BF-49 Add the Phase 4 specification` | **A** `4.md`, **M** `README.md` *(письма печатаются в консоль)* |

---

## W5 — Контроль входа (12–18 октября)

**Цель:** дверь работает. Один билет — один вход.

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-50 Update the money assertions for the fee` | **M** `db/init/02_schema.sql` *(+2 таблицы, частичный уникальный индекс)*, **M** `server.go`, **M** `api/internal/api/account_test.go`, **M** `api/internal/api/notifications_test.go`, **M** `api/internal/api/profile_test.go` *(комиссия 3.5% сдвинула каждую сумму в тестах)* |
| S2 | `BF-51 Reserve stock with 15-minute holds` | **A** `store/holds.go`, **A** `handlers_holds.go`, **A** `api/internal/api/holds_test.go`, **M** `store/checkout.go` *(**рефакторинг**, `-180/+120`)* |
| S2 | `BF-52 Charge a processing fee` | **M** `store/checkout.go`, **M** `store/holds.go`, **M** `checkout_test.go`, **M** `api/internal/api/seating_test.go`, **M** `api/internal/api/ticket_types_test.go`, **M** `api/internal/api/tickets_test.go` *(комиссия 3.5% сдвинула каждую сумму в тестах)*, **M** `api/README.md` *(резервы и комиссия)* |
| S3 | `BF-53 Show the hold countdown at checkout` | **M** `web/src/components/checkout-dialog.tsx` *(`-9/+34`, мгновенная покупка **заменена** на резерв)*, **M** `web/src/lib/types.ts` |
| S4 | `BF-54 Add the staff manager` | **A** `web/src/components/staff-manager.tsx` |
| S5 | `BF-55 Prepare the gate backend and a fake scan payload` | **A** `store/staff.go`, **A** `handlers_staff.go`, **A** `mobile/lib/mock-tickets.ts` *(сканер собирают на фейковых payload'ах, пока гейта нет)* |
| S5 | `BF-56 Scan and admit a ticket` | **A** `mobile/app/scanner.tsx`, **A** `store/checkin.go`, **A** `handlers_checkin.go`, **D** `mobile/lib/mock-tickets.ts` *(вытеснен реальным ответом гейта)*, а не одним экраном)*, **A** `mobile/components/ScanResultOverlay.tsx`, **A** `api/internal/api/checkin_test.go` |
| S5 | `BF-57 Document how to run the scanner` | **A** `mobile/app/attendees/[eventId].tsx`, **M** `handlers_checkin.go`, **A** `5.md`, **M** `api/scripts/smoke_test.sh`, **M** `.github/workflows/ci.yml` *(дымовой прогон после сборки)*, **M** `README.md`, **M** `mobile/README.md`, **M** `Makefile` *(Expo требует отдельных шагов)*, **M** `3.md`, **M** `4.md` *(суммы в ожиданиях сдвинулись на 3.5%)* |

---

## W6 — Возвраты, поддержка, админ (19–25 октября)

**Цель:** деньги возвращаются, зритель может попросить помощи, платформа модерируется.

**Фаза безопасности.** `session.ts` теряет три функции и превращается в одну
константу; `middleware.ts` **переименовывается** в `proxy.ts`; `api.ts` меняет адрес
на прокси. Подробности — `file-histories.md` §3 и §4.

коммит с тем же названием. Дефекты `BF-B1` · `BF-B2` · `BF-B3` заводятся по ходу
фазы, когда их находят.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-58 Add refund, support and report tables` | **M** `db/init/02_schema.sql` *(+5 таблиц)*, **M** `server.go` |
| S1 | `BF-59 Add the report queue and settings` | **A** `store/admin.go`, **A** `handlers_admin_portal.go`, **A** `handlers_admin_users.go`, **A** `store/moderation.go`, **A** `handlers_moderation.go`, **A** `handlers_admin.go`, **A** `api/internal/store/reports.go` *(отдельный стор очереди жалоб)*, **A** `api/internal/api/{portal_test,moderation_test,admin_users_test}.go`, **A** `db/tests/06_end_to_end_flow.sql` |
| S1 | `BF-60 Give every conflict its own error code` | **D** `db/tests/07_temp_checks.sql` *(вытеснен `05_business_rules.sql`)*, **M** `api/internal/store/audit.go`, **M** `api/internal/store/notifications.go`, **M** `api/internal/store/profiles.go`, **M** `api/internal/store/store.go`, **M** `api/internal/store/store_test.go` *(поиск по всем сущностям потребовал новых выборок)*, **M** `db/tests/01_environment.sql`, **M** `db/tests/02_schema_objects.sql`, **M** `db/tests/03_user_crud.sql`, **M** `db/tests/04_constraints.sql`, **M** `db/tests/05_business_rules.sql`, **M** `db/tests/_fixture.sql`, **M** `db/tests/_helpers.sql`, **M** `db/tests/run_tests.sh` *(общий `conflict` разошёлся на именованные коды)*, **M** `api/internal/testutil/testdb.go` *(наборы тестов разрослись, фикстура вынесена)* |
| S2 | `BF-61 Refund an order atomically` | **A** `store/refunds.go`, **A** `handlers_refunds.go`, **A** `api/internal/api/refunds_test.go`, **M** `refunds_test.go` |
| S2 | `BF-62 Expose the rows the admin search needs` | **A** `store/cancellations.go`, **A** `handlers_cancellations.go`, **A** `cancellations_test.go`, **M** `api/internal/store/activation.go`, **M** `api/internal/store/seating.go`, **M** `api/internal/store/ticket_types.go`, **M** `api/internal/store/tickets.go` *(поиск по всем сущностям потребовал новых выборок)*, **M** `api/internal/api/handlers_calendar.go`, **M** `api/internal/api/handlers_holds.go`, **M** `api/internal/api/handlers_seating.go`, **M** `api/internal/api/handlers_tickets.go` *(общий `conflict` разошёлся на именованные коды)*, **M** `api/README.md` *(возвраты и отмены)* |
| S3 | `BF-63 Add the support widget to the order page` | **A** `web/src/components/order-support.tsx`, **M** `web/src/app/orders/[id]/page.tsx`, **D** `web/src/components/order-list.tsx` *(вытеснен `order-manager.tsx` у организатора)*, **A** `web/src/lib/support.ts`, **M** `web/src/components/ticket-card.tsx`, **D** `web/src/components/ui/toast.tsx` *(вытеснен инлайновыми состояниями)*, **M** `web/README.md` *(браузер больше не ходит в API напрямую)* |
| S3 | `BF-64 Point the client at the proxy` | **M** `web/src/lib/api.ts` *(адрес `:8080` → `/api/proxy`)* |
| S4 | `BF-65 Add the admin portal` | **R** `api/internal/store/guests.go` → `api/internal/store/attendees.go` *(в SRS термин «attendee», а не «guest»)*, **A** `store/support.go`, **A** `handlers_support.go`, **A** `web/src/components/{support-inbox,support-thread}.tsx`, **A** `api/internal/api/support_test.go`, **A** `web/src/app/(app)/admin/{page,admin-portal}.tsx`, **A** `web/src/components/moderation-action.tsx`, **A** `web/src/app/(app)/admin/reports/{page,reports-queue}.tsx`, **M** `api/internal/store/attendees.go` *(поиск по всем сущностям потребовал новых выборок)*, **M** `api/internal/api/handlers_attendees.go` *(общий `conflict` разошёлся на именованные коды)* |
| S4 | `BF-B2 Move the session into an httpOnly cookie` | **M** `web/src/lib/session.ts` *(`-38/+14`, три функции **удалены**)*, **A** `web/src/lib/server-session.ts`, **A** `web/src/app/api/auth/{login,logout,me,register}/route.ts`, **A** `web/src/app/api/proxy/[...path]/route.ts`, **M** `web/src/lib/auth-context.tsx`, **D** `web/src/app/api/health/route.ts` *(вытеснен прокси-маршрутом)* |
| S4 | `BF-B3 Rename middleware to proxy for Next 16` | **R** `web/src/middleware.ts` → `web/src/proxy.ts`, **M** `web/src/proxy.ts` |
| S5 | `BF-66 Refuse a refunded ticket at the gate` | **M** `store/checkin.go`, **M** `checkin_test.go` |
| S5 | `BF-B1 Keep the check-in record when a ticket is voided` | **M** `store/checkin.go`, **M** `checkin_test.go` |
| S5 | `BF-67 Add the Phase 6 specification` | **A** `6.md`, **M** `5.md`, **M** `api/internal/store/staff.go` *(поиск по всем сущностям потребовал новых выборок)*, **M** `api/internal/api/handlers_staff.go` *(общий `conflict` разошёлся на именованные коды)*, **M** `README.md` *(появился портал)* |

---

## W7 — Аналитика, кампании, i18n (26 октября – 1 ноября)

**Цель:** организатор видит цифры, запускает кампании; сайт говорит по-казахски.

`dictionaries.ts` выходит **только с английским** и получает `ru` и `kk` отдельными
коммитами. `analytics.ts` разделяется на два модуля (`file-histories.md` §5).

коммит с тем же названием. Дефекты `BF-B4` заводятся по ходу фазы, когда их находят.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-68 Add campaign tables` | **M** `db/init/02_schema.sql` *(+3 таблицы)*, **M** `server.go` |
| S2 | `BF-69 Create campaigns and promo codes` | **A** `store/campaigns.go`, **A** `handlers_campaigns.go`, **A** `handlers_promo.go` |
| S2 | `BF-70 Apply and cap discounts atomically` | **M** `store/campaigns.go`, **M** `store/checkout.go`, **A** `api/internal/api/campaigns_test.go`, **A** `store/duplicate.go`, **M** `handlers_events.go`, **M** `api/README.md` *(кампании и промокоды)* |
| S3 | `BF-71 Add the English dictionary` | **A** `web/src/lib/i18n/{config.ts,context.tsx,dictionaries.ts,translate.ts,server.ts}` *(**только en**)*, **M** `web/src/app/layout.tsx`, **M** `web/src/lib/i18n/dictionaries.ts` *(+`ru`)*, **M** `i18n/config.ts`, **A** `web/src/components/language-switcher.tsx`, **A** `web/src/app/api/locale/route.ts` |
| S3 | `BF-72 Localise the attendee pages` | **A** `web/src/components/promo-box.tsx`, **M** `web/src/lib/api.ts`, **M** `web/src/lib/types.ts`, **M** `web/src/app/(auth)/forgot-password/forgot-password-form.tsx`, **M** `web/src/app/(auth)/forgot-password/page.tsx`, **M** `web/src/app/(auth)/layout.tsx`, **M** `web/src/app/(auth)/login/login-form.tsx`, **M** `web/src/app/(auth)/login/page.tsx`, **M** `web/src/app/(auth)/register/page.tsx`, **M** `web/src/app/(auth)/register/register-form.tsx`, **M** `web/src/app/(auth)/reset-password/page.tsx`, **M** `web/src/app/(auth)/reset-password/reset-password-form.tsx`, **M** `web/src/app/(auth)/verify-email/page.tsx`, **M** `web/src/app/(auth)/verify-email/verify-email-view.tsx`, **M** `web/src/app/error.tsx`, **M** `web/src/app/events/[slug]/ticket-selector.tsx`, **M** `web/src/app/events/layout.tsx`, **M** `web/src/app/events/page.tsx`, **M** `web/src/app/favicon.ico`, **M** `web/src/app/globals.css`, **M** `web/src/app/not-found.tsx`, **M** `web/src/app/page.tsx`, **M** `web/src/components/event-card.tsx`, **M** `web/src/components/order-support.tsx`, **M** `web/src/components/seat-checkout.tsx`, **M** `web/src/components/seat-map.tsx`, **M** `web/src/components/seated-purchase.tsx`, **M** `web/src/components/ui/alert.tsx`, **M** `web/src/components/ui/button.tsx`, **M** `web/src/components/ui/field.tsx`, **M** `web/src/components/ui/status-badge.tsx`, **M** `web/src/lib/datetime.test.ts`, **M** `web/src/lib/datetime.ts`, **M** `web/src/lib/money.test.ts`, **M** `web/src/lib/money.ts`, **M** `web/src/lib/support.ts` |
| S3 | `BF-73 Add the language negotiation dependency` | **M** `web/package.json`, **M** `web/package-lock.json`, **M** `web/next.config.ts` *(разбор `Accept-Language`)*, **M** `web/README.md` *(kk, ru, en)* |
| S4 | `BF-74 Compute analytics from operational rows` | **A** `store/analytics.go`, **A** `handlers_analytics.go`, **A** `web/src/app/(app)/dashboard/analytics/page.tsx` *(аналитика жила отдельной страницей)*, **A** `web/src/components/stats-panel.tsx`, **A** `api/internal/api/analytics_test.go` |
| S4 | `BF-75 Localise the dashboard` | **M** `web/src/components/stats-panel.tsx`, **M** `dashboard/events/[id]/page.tsx`, **M** `handlers_analytics.go`, **A** `web/src/components/event-timeline.tsx`, **A** `web/src/components/campaign-manager.tsx`, **M** `(app)/dashboard/events/[id]/page.tsx`, **A** `web/src/lib/analytics.ts`, **A** `web/src/components/analytics-scripts.tsx`, **A** `web/src/components/event-analytics-tracker.tsx`, **M** `web/src/app/(app)/admin/admin-portal.tsx`, **M** `web/src/app/(app)/admin/page.tsx`, **M** `web/src/app/(app)/admin/reports/page.tsx`, **M** `web/src/app/(app)/admin/reports/reports-queue.tsx`, **M** `web/src/app/(app)/dashboard/events/[id]/edit/edit-event-form.tsx`, **M** `web/src/app/(app)/dashboard/events/[id]/edit/page.tsx`, **M** `web/src/app/(app)/dashboard/events/[id]/event-detail.tsx`, **M** `web/src/app/(app)/dashboard/page.tsx`, **M** `web/src/app/(app)/dashboard/profile/page.tsx`, **M** `web/src/app/(app)/dashboard/profile/profile-form.tsx`, **M** `web/src/app/(app)/events/new/create-event-form.tsx`, **M** `web/src/app/(app)/layout.tsx`, **M** `web/src/app/api/auth/login/route.ts`, **M** `web/src/app/api/auth/logout/route.ts`, **M** `web/src/app/api/auth/me/route.ts`, **M** `web/src/app/api/auth/register/route.ts`, **M** `web/src/app/api/proxy/[...path]/route.ts`, **M** `web/src/components/activation-checklist.tsx`, **M** `web/src/components/attendee-list.tsx`, **M** `web/src/components/event-form-fields.tsx`, **M** `web/src/components/image-upload.tsx`, **M** `web/src/components/moderation-action.tsx`, **M** `web/src/components/staff-manager.tsx`, **M** `web/src/components/support-inbox.tsx`, **M** `web/src/components/support-thread.tsx`, **M** `web/src/components/ticket-type-manager.tsx` *(то же самое для экранов организатора)*, **M** `web/src/lib/server-session.ts` |
| S4 | `BF-B4 Read the GA4 id from a server-safe module` | **A** `web/src/lib/analytics-config.ts`, **M** `web/src/lib/analytics.ts` *(`-21/+6`, чтение id **вырезано**)*, **M** `web/src/components/analytics-scripts.tsx` |
| S5 | `BF-76 Add the Phase 7 specification` | **A** `7.md`, **D** `.github/workflows/lint.yml` *(слит в `ci.yml` одним job)*, **M** `Makefile` *(`make i18n-check` ищет пропущенные ключи)* |

---

## W8 — Релизная форма (2–8 ноября)

**Цель:** проект выходит ровно в том виде, в каком он есть сегодня.

Это фаза замен. Всё, что в Ф2 было написано «как пишут сначала», переписывается.
Здесь будет больше всего удалённых строк за весь семестр — так и выглядит настоящий
hardening-спринт.

коммит с тем же названием. Дефекты `BF-B5` · `BF-B6` · `BF-B7` · `BF-B8` · `BF-B9` ·
`BF-B10` заводятся по ходу фазы, когда их находят.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-77 Count text limits in the remaining handlers` | **M** `api/internal/api/validate.go` *(`-25/+90`)*, **M** `handlers_auth.go`, **M** `handlers_account.go`, **M** `handlers_profile.go`, **M** `api/internal/auth/password.go` *(`-18/+30`, `ErrPasswordTooLong` **удалён**)*, **M** `auth/password_test.go` *(тест **удалён**)*, **M** `server.go`, **M** `api/internal/api/admin_users_test.go`, **M** `api/internal/api/moderation_test.go`, **M** `api/internal/api/portal_test.go` *(тесты считали байты вслед за кодом)*, **D** `api/internal/store/reports.go` *(слит в `admin.go`)*, **M** `api/internal/store/admin.go`, **M** `api/internal/store/moderation.go` *(общий `conflict` дочищен до именованных кодов)*, **M** `api/internal/api/handlers_admin.go`, **M** `api/internal/api/handlers_admin_portal.go`, **M** `api/internal/api/handlers_admin_users.go`, **M** `api/internal/api/handlers_moderation.go`, **M** `api/internal/api/money.go`, **M** `api/internal/auth/token.go`, **M** `api/internal/auth/token_test.go`, **M** `api/internal/config/config_test.go`, **M** `api/internal/email/email_test.go`, **M** `api/internal/httpx/context.go` *(лимиты текста считаются символами по всему API)* |
| S1 | `BF-B5 Stop sharing notification state on the server` | **M** `api/internal/api/notify.go`, **M** `server.go`, **M** `email/email.go` *(гонка; общее поле **удалено**)* |
| S1 | `BF-B6 Make the seed re-runnable after a purchase` | **M** `db/seed/01_demo_data.sql` *(`-12/+40`, порядок FK и upsert)* |
| S2 | `BF-78 Cut Cyrillic text by characters on the ticket` | **M** `handlers_events.go`, **M** `handlers_ticket_types.go`, **M** `handlers_checkout.go`, **M** `handlers_uploads.go`, **A** `store/my_orders.go`, **A** `handlers_my_orders.go`, **A** `api/internal/api/my_orders_test.go`, **M** `api/internal/api/campaigns_test.go`, **M** `api/internal/api/cancellations_test.go`, **M** `api/internal/api/holds_test.go` *(то же для коммерческих хендлеров)*, **M** `api/internal/store/cancellations.go`, **M** `api/internal/store/duplicate.go` *(общий `conflict` дочищен до именованных кодов)*, **M** `api/internal/calendar/ics.go`, **M** `api/internal/calendar/ics_test.go`, **M** `api/internal/ticketpdf/extract_test.go`, **M** `api/internal/ticketpdf/fonts.go`, **M** `api/internal/ticketpdf/fonts/LICENSE-fpdf.txt`, **M** `api/internal/ticketpdf/qr.go`, **M** `api/internal/ticketpdf/text.go`, **M** `api/internal/ticketpdf/ticketpdf_test.go` *(байтовая обрезка резала двухбайтовую букву пополам)*, **M** `api/internal/api/handlers_cancellations.go`, **M** `api/internal/api/handlers_promo.go` *(лимиты текста считаются символами по всему API)*, **M** `api/README.md` *(зритель ищет свои заказы)* |
| S2 | `BF-B7 Give a suspended event its own code` | **M** `handlers_activation.go`, **M** `handlers_checkout.go`, **M** `activation_test.go` |
| S2 | `BF-B8 Name the refund conflict` | **M** `handlers_refunds.go`, **M** `refunds_test.go` |
| S3 | `BF-79 Localise the remaining attendee screens` | **M** `web/src/lib/i18n/dictionaries.ts`, **M** `web/src/lib/api.ts`, **M** `web/src/lib/types.ts`, **M** `web/src/app/api/locale/route.ts`, **M** `web/src/components/language-switcher.tsx`, **M** `web/src/components/promo-box.tsx`, **M** `web/src/lib/i18n/context.tsx`, **M** `web/src/lib/i18n/server.ts`, **M** `web/src/lib/i18n/translate.ts` *(последние строки уехали в словарь)* |
| S4 | `BF-80 Show reserved stock` | **R** `web/src/components/stats-panel.tsx` → `web/src/components/event-analytics.tsx` *(панель выросла в аналитику события)*, **M** `store/analytics.go`, **A** `api/internal/api/payout_figures_test.go`, **M** `web/src/components/event-analytics.tsx`, **A** `reserved_counts_test.go`, **D** `web/src/app/(app)/dashboard/analytics/page.tsx` *(страница вложена в карточку события)* |
| S4 | `BF-B9 Name the reversal conflict` | **M** `handlers_support.go`, **M** `support_test.go` |
| S4 | `BF-81 Localise the remaining dashboard screens` | **A** `web/src/app/(app)/orders/page.tsx`, **M** `web/src/components/site-header.tsx`, **M** `api/internal/store/support.go` *(общий `conflict` дочищен до именованных кодов)*, **M** `web/src/components/campaign-manager.tsx`, **M** `web/src/components/event-analytics-tracker.tsx`, **M** `web/src/components/event-timeline.tsx`, **M** `web/src/lib/analytics-config.ts` *(последние строки уехали в словарь)* |
| S5 | `BF-B10 Name the check-in reversal conflict` | **M** `handlers_checkin.go`, **M** `checkin_test.go` |
| S5 | `BF-82 Add the Phase 8, 9 and 10 specifications` | **A** `8.md`, **A** `9.md`, **A** `10.md`, **D** `docs/api-notes.md` *(вытеснен разделом в `api/README.md`)*, **D** `docs/decisions.md` *(вытеснен разделом в `README.md`)* |
| S5 | `BF-83 Point the scanner at the queue` | **R** `mobile/app/scanner.tsx` → `mobile/app/scan/[eventId].tsx` *(сканер стал маршрутом на событие)*, **A** `api/internal/store/offline.go`, **A** `api/internal/api/handlers_offline.go`, **A** `api/internal/api/offline_test.go`, **A** `mobile/lib/{offline,offline-db,use-connectivity}.ts`, **A** `mobile/app/offline/[eventId].tsx`, **M** `mobile/app/scan/[eventId].tsx`, **M** `mobile/.env.example`, **M** `mobile/.gitignore`, **M** `mobile/AGENTS.md`, **M** `mobile/CLAUDE.md`, **M** `mobile/LICENSE`, **M** `mobile/README.md`, **M** `mobile/app.json`, **M** `mobile/app/_layout.tsx`, **M** `mobile/app/attendees/[eventId].tsx`, **M** `mobile/app/events.tsx`, **M** `mobile/app/index.tsx`, **M** `mobile/app/login.tsx`, **M** `mobile/components/ScanResultOverlay.tsx`, **M** `mobile/lib/api.ts`, **M** `mobile/lib/auth-context.tsx`, **M** `mobile/lib/config.ts`, **M** `mobile/lib/session.ts`, **M** `mobile/lib/theme.ts`, **M** `mobile/lib/types.ts`, **M** `mobile/package-lock.json`, **M** `mobile/package.json`, **M** `mobile/tsconfig.json` *(каждый экран теперь умеет работать без сети)*, **D** `mobile/lib/storage.ts` *(вытеснен expo-sqlite)*, **M** `bilet.md` *(формулировки лимитов разошлись с кодом)*, **M** `1.md`, **M** `2.md` *(лимиты в символах, а не в байтах)* |

**Контрольная точка Ф10:** после merge всех веток **продуктовый код** совпадает с
нынешним. Не хватает только того, что Ф11 добавляет сверху: e2e-набора, Postman,
criteria-тестов. Ни один переходный файл до Ф11 не доживает — это проверяет
`plan/tools/check-plan.py`.

```bash
git diff phase-10-done main -- . ':!plan' ':!web/e2e' ':!web/playwright.config.ts' \
  ':!web/vitest.*' ':!docs' ':!api/internal/api/criteria_test.go' \
  ':!api/internal/api/phase1*_test.go'   # должно быть пусто
```

---

## W9 — Проверка (9–15 ноября)

**Цель:** прогнать все десять спецификаций и починить найденное.

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-84 Say "required" instead of "too short"` | **M** `api/internal/api/validate.go`, **M** `auth_test.go`, **A** `api/internal/api/routing_test.go` |
| S2 | `BF-85 Refuse to publish an event that has ended` | **M** `handlers_events.go`, **M** `events_test.go` |
| S3 | `BF-86 Close the localisation gaps` | **M** `web/src/lib/i18n/dictionaries.ts` |
| S3 | `BF-87 Add the test runner dependencies` | **A** `web/{vitest.config.mts,vitest.setup.ts}`, **A** `web/src/components/ui/field.test.tsx`, **M** `web/package.json`, **M** `web/package-lock.json`, **M** `web/eslint.config.mjs`, **M** `web/postcss.config.mjs` *(vitest и playwright в наборе)* |
| S4 | `BF-88 Make the money card answer the tier filter` | **M** `store/analytics.go`, **M** `analytics_test.go` |
| S5 | `BF-89 Run the phase specifications and record results` | **M** `1.md`, **M** `6.md`, **A** `docs/verification/**` *(скриншоты)*, **A** `api/internal/api/{criteria_test,phase10_test,phase12_test}.go` |
| S5 | `BF-90 Correct the specifications after the run` | **A** `web/playwright.config.ts`, **A** `web/e2e/{fixtures.ts,auth.spec.ts,free-event.spec.ts,paid-checkout.spec.ts,analytics-and-gate.spec.ts,support-and-moderation.spec.ts}`, **M** `10.md`, **M** `2.md`, **M** `4.md`, **M** `7.md`, **M** `8.md`, **M** `9.md` *(спецификации правятся под фактическое поведение)*, **M** `.github/workflows/ci.yml`, **M** `web/Dockerfile`, **M** `web/.dockerignore`, **M** `api/Dockerfile`, **M** `api/.dockerignore` *(e2e требует собранных образов)* |
| S5 | `BF-91 Add the Postman collection` | **A** `docs/biletflow-api.postman_collection.json` *(2 177 строк, целиком)* |

**Контрольная точка Ф11:** здесь `git diff phase-11-done main -- . ':!plan'` обязан быть
пустым целиком — это и есть сегодняшний репозиторий. Ф12 только дописывает
документацию и ставит тег.

Если после Ф11 всё зелёное — Ф12 можно не делать. Номер ничего не решает.

---

## W10 — Документация и релиз (16–22 ноября)

коммит с тем же названием.

| Кто | Коммит | Файлы |
| --- | --- | --- |
| S1 | `BF-92 Record the final plan state` | **M** `plan/**` |
| S2 | `BF-93 Update the API documentation` | **M** `api/README.md` |
| S3 | `BF-94 Update the web documentation` | **M** `web/README.md` |
| S5 | `BF-95 Update the README and run instructions` | **M** `README.md`, **M** `Makefile` |
| S5 | `BF-96 Tag v1.0.0` |  |

- **ср 25, 18:00 — feature freeze.** Дальше только починки и репетиция.
- **сб 28 — Репорт 7 и демо.**

---

## Перед каждым репортом

```bash
python3 plan/tools/check-plan.py      # зоны не пересеклись
bash plan/tools/check-history.sh      # история похожа на настоящую
```

## Если не успеваем

Режем в этом порядке — первые четыре в SRS §8 помечены как бонус:

1. Офлайн-синхронизация сканера
2. Экспорт `.ics`
3. Интерактивная карта мест — откат на счётчик количества
4. GA4

Не режем никогда: корректность checkout, возвраты, защиту от повторного входа и
казахский с русским на страницах для зрителя. Это требования.

---
