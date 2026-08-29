# BiletFlow Web — Phases 3-5, 7-10 & 12

Next.js 16 (App Router) + React 19 + Tailwind CSS 4 frontend for the Go API.

Organizers register, sign in, manage events and configure ticket types.
Attendees browse a public event page and buy tickets through a simulated
KZT checkout.

---

## Run it

All three layers need to be running. From the **repository root**:

```bash
make up        # 1. PostgreSQL      (Phase 1)
make api-run   # 2. Go API :8080    (Phase 2) — leave running in its own shell
make web-dev   # 3. Dashboard :3000 (Phase 3)
```

Then open <http://localhost:3000>.

Or directly, from this directory:

```bash
npm install
npm run dev
```

| Command | What it does |
| --- | --- |
| `npm run dev` | Dev server on <http://localhost:3000> |
| `npm run build` | Production build |
| `npm run start` | Serve the production build |
| `npm run lint` | ESLint |

### Configuration

`.env.local` (copied from `.env.example`) holds one setting:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
```

`NEXT_PUBLIC_` is required because the browser calls the Go API directly, so the
value is inlined into the client bundle.

---

## Pages

| Route | Auth | What it does |
| --- | --- | --- |
| `/` | – | Redirects to `/dashboard` or `/login` |
| `/register` | – | Create an account, then straight to the dashboard |
| `/login` | – | Sign in, honouring `?next=` |
| `/dashboard` | required | Your events, with publish and cancel |
| `/dashboard/events/[id]` | required | One event, with its ticket types |
| `/events/new` | required | Create-event form |
| `/events/[slug]` | **public** | Attendee event page and ticket selector |
| `/orders/[id]` | **public** | Order confirmation, QR codes and PDF downloads |

`/events/[slug]` and `/orders/[id]` are deliberately outside the auth gate —
buying a ticket must work without an account. The proxy matcher lists
`/events/new` explicitly rather than `/events/:path*` for exactly that reason.

---

## How it is wired

```
src/
├── proxy.ts                      route gate (Next.js 16 renamed middleware → proxy)
├── lib/
│   ├── api.ts                    the only place that talks to the Go API
│   ├── session.ts                where the token lives
│   ├── auth-context.tsx          AuthProvider + useAuth
│   ├── datetime.ts               timezone-correct form ⇄ RFC 3339 conversion
│   └── types.ts                  mirrors of the Go response structs
│   └── money.ts                  KZT formatting, integer-tiyn arithmetic
├── components/
│   ├── site-header.tsx           nav + signed-in user + sign out
│   ├── event-card.tsx            one event row, with publish/cancel
│   ├── ticket-type-manager.tsx   organizer's ticket type CRUD
│   ├── checkout-dialog.tsx       the simulated payment modal
│   ├── ticket-card.tsx           QR preview + PDF download for one ticket
│   ├── campaign-manager.tsx      organizer promo codes + campaign QR
│   ├── promo-box.tsx             attendee promo entry and auto-apply
│   ├── support-thread.tsx        one conversation, shared by both sides
│   ├── order-support.tsx         attendee support panel on an order
│   ├── support-inbox.tsx         organizer support inbox for an event
│   └── ui/                       button, field, alert, status badge
└── app/
    ├── layout.tsx                fonts, metadata, AuthProvider
    ├── (auth)/                   centred card shell → login, register
    ├── (app)/                    header shell + auth gate → dashboard, events/new
    ├── events/[slug]/            public event page (Server Component)
    └── orders/[id]/              order confirmation (Server Component)
```

### The API client

`src/lib/api.ts` wraps native `fetch` — no Axios, because a base URL, a bearer
header, JSON encoding and typed errors are a few lines each and it keeps the
bundle smaller.

It attaches `Authorization: Bearer <token>` automatically, and turns the Go
API's error envelope into a typed `ApiError`:

```ts
try {
  await api.createEvent(payload);
} catch (error) {
  if (error instanceof ApiError) {
    error.status;   // 422
    error.code;     // "validation_failed"
    error.fields;   // { ends_at: "End time must be after the start time." }
  }
}
```

`error.fields` is fed straight back into the form, so a 422 from Go highlights
the exact input that caused it instead of showing a generic banner.

A 401 clears the stored token, so a rejected session cannot linger.

### Where the token lives

In a cookie (`biletflow_token`), not `localStorage` — for one concrete reason:
`proxy.ts` runs on the server and can only read cookies. Route protection
therefore happens *before* the page renders, rather than flashing a dashboard
and then bouncing to `/login`. The cookie's lifetime is set from the API's own
`expires_at`, so both expire together.

The cookie is deliberately readable by JavaScript, because the Go API
authenticates with a header rather than a cookie. **That carries the same XSS
exposure as `localStorage` would.** Moving to an httpOnly cookie needs a Next.js
route handler proxying every API call plus a refresh token to rotate against —
neither exists yet, so this is a documented Phase 3 trade-off, not an oversight.

### Two gates, not one

`proxy.ts` is an *optimistic* check: it can see that a token cookie exists, not
whether the token is still valid. `AuthProvider` does the real verification with
`GET /auth/me` and signs the user out when the API rejects it — which also
catches an account suspended after the token was issued. The `(app)` layout
waits for that answer before rendering.

### Timezones

The form's `datetime-local` inputs have no timezone. Passing `19:00` to
`new Date()` would interpret it in the **browser's** zone — wrong for anyone
planning an Almaty event from elsewhere.

`localInputToISO()` instead combines the wall-clock time with the event's chosen
IANA zone, deriving the offset from `Intl` (so DST is the platform's problem)
and refining once to settle DST boundaries. The dashboard renders timestamps
back in each event's own zone, so what you typed is what you see.

---

## Notes on the implementation

- **The dashboard reads `GET /api/v1/events/mine`, not `GET /api/v1/events`.**
  The latter is the public catalogue: it returns only *published, publicly
  visible* events, so a freshly created draft would be missing from the very
  dashboard that just created it. `/events/mine` is the organizer-scoped
  endpoint built for exactly this in Phase 2.
- **New events are drafts.** Publishing is a separate, explicit action from the
  dashboard, matching the API's lifecycle.
- **The create form asks for start and end times** even though the brief listed
  only Title, Slug, Capacity and Timezone — the API rejects an event without
  them (422). Everything else is behind an "Optional details" disclosure.
- **Server Components are used where they add something.** The auth pages,
  dashboard and form are interactive and read a browser cookie, so they are
  Client Components by necessity; `/` is a Server Component that reads the
  cookie and redirects without a client round-trip.
- `useSearchParams` in the login form sits behind a `Suspense` boundary, so the
  route is not forced into full client-side rendering.


---

## The attendee flow (Phase 4)

`/events/[slug]` is a **Server Component**. The public endpoint needs no token,
so the event and its live remaining counts are fetched on the server: real HTML
for sharing and search, and no loading flash. Only the ticket selector and its
checkout dialog are client-side.

The checkout is a native `<dialog>`, which brings the focus trap, Escape
handling and backdrop with it rather than reimplementing all three.

### Money

Prices arrive as decimal strings and stay that way. `lib/money.ts` converts to
integer **tiyn** (1/100 KZT) for the running total, so the selector never adds
floats together, and formats with `Intl.NumberFormat`.

### Two layers of inventory protection, one honest

The stepper clamps to `min(quantity_remaining, max_per_order)` and disables `+`
at the limit. That is a **convenience**, not a control — the page could be stale
by the time you press pay.

The server is the authority. When it answers `409 insufficient_inventory`, the
dialog shows what actually remains and calls `router.refresh()` to re-fetch the
Server Component, so the page corrects itself:

> These tickets sold out while you were choosing. The page has been refreshed.

This path is exercised by a real race in the browser, not just in theory: sell
the last tickets from another client while a checkout dialog is open, press pay,
and the message above is what appears.


---

## Ticket delivery (Phase 5)

`/orders/[id]` shows every issued ticket with its scannable QR and a
**Download PDF ticket** button.

**The download is a plain `<a href>`, not a fetch.** The API answers with
`Content-Disposition: attachment`, and that header is what makes the browser
save the file — the HTML `download` attribute is ignored cross-origin, the
header is not. No blob handling, no object URLs to revoke.

**The QR preview is a plain `<img>`, not `next/image`.** It is a per-ticket API
resource on another origin, served `no-store` because a ticket can be cancelled.
Routing it through the image optimizer would cache and re-encode it, and a
re-encoded QR is a QR that might not scan. This is the one deliberate exception
to the `next/image` rule, and it is marked as such in `ticket-card.tsx`.

Both the preview and the PDF are generated by the API from the same token with
the same encoder, so what a viewer sees on screen and what is printed can never
drift apart.


---

## Promo codes (Phase 7)

**Organizer** — `/dashboard/events/[id]` gains a campaign section: create a
promo code, see its campaign QR, copy the trackable link, download the PNG for a
poster, and watch orders, tickets and revenue attributed to it.

**Attendee** — `/events/[slug]` gains a promo box with two ways in that behave
identically:

- typing a code and pressing Apply, or
- arriving through a campaign QR, whose link carries `?c=CMP_…` and applies
  itself.

Every figure shown is priced by the server. The component sends the basket and
renders the discount it is handed back; it never computes one, so what appears
on screen is exactly what checkout charges.

### Details that matter

**A discount belongs to the basket it was priced against.** A percentage of two
tickets is not a percentage of three, so changing the selection re-prices the
code rather than showing a stale figure. The re-price is debounced by 250 ms, so
tapping "+" three times costs one request rather than three.

**Removing a code makes it stay removed.** Without that, arriving via a campaign
link would silently re-apply the discount the moment the attendee changed their
basket.

**A code can run out between applying it and paying.** The dialog catches
`promo_*` errors from checkout, drops the discount and shows the real price
rather than a total that can no longer be honoured.


---

## Support chat (Phase 8)

One `SupportThreadView` serves both sides. What differs is what the API returns:
`can_moderate` decides whether the status buttons and the internal-note checkbox
render, and the server has already filtered internal notes out of an attendee's
copy of the thread. The UI never decides who may see what.

**Attendee** — the order page gains a Support panel: pick a category, ask a
question, and the conversation appears inline. A guest buyer is told which
address to sign in with rather than being shown a form that would fail.

**Organizer** — the event page gains an inbox listing every case with its
status, requester and order, expanding into the thread in place.

**Polling, not sockets**, per SRS 4.13: an open thread refreshes every 10 s, the
inbox every 15 s, and a resolved thread stops polling altogether rather than
asking the server about a finished conversation forever.

## Suspended events

A suspended event still resolves publicly — people already hold links to it —
but the ticket selector is not rendered at all and a red banner explains that
sales are paused and existing tickets remain valid. The organizer sees the same
news on their own event page.


## Analytics and history (Phase 9)

The event page opens on its own numbers. `EventAnalytics` renders four stat
cards — tickets sold, remaining, gross revenue, checked in — each with the
figure that gives it meaning underneath (`of 40 capacity · 17.5% sold`, `5
absent · 28.6% attended`), then a per-ticket-type table and, when campaigns
exist, what each one actually brought in.

The sales-over-time chart is plain `div`s sized by percentage. A charting
library would have been the larger dependency in the project for one bar chart,
and this one inherits the theme, scales with the container and needs no client
bundle of its own.

**Nothing is computed in the browser.** Percentages, sums and rounding all
arrive from the API, so the page cannot disagree with the database — the failure
mode where a dashboard quietly rounds differently from the export simply does
not exist here.

**Filters** (from / to / ticket type) re-request the server rather than
narrowing an array on the client, so a filtered view is as authoritative as the
unfiltered one.

**Activity history** lists the append-only audit trail newest first, filtered by
type (Event, Tickets, Orders, Campaigns, Support) or by date.

## Duplicating an event

"Duplicate event" on the event page copies the configuration and navigates
straight to the new draft, where a banner explains it must be published before
anyone can find it. The dashboard's **Drafts** tab shows it alongside anything
else unpublished.

The dashboard tabs — All, Upcoming, Active, Completed, Cancelled, Drafts — pass
`lifecycle` to `GET /events/mine`. The badge on each card comes from the same
server-derived value as the filter, so a card can never be labelled differently
from the tab it appears under.


## Refunds and activation (Phase 10)

### The activation checklist

`ActivationChecklist` sits at the top of the organizer's event page, above the
numbers: an organizer whose paid tickets are not on sale needs to know that
before they wonder why nothing has sold. It renders **nothing at all** for a
free event — nagging about a checklist that gates nothing is noise.

Which boxes are already ticked comes from the server's `outstanding` list, so
the UI never keeps its own tally of what "complete" means. Steps can be saved
one at a time or all four together; the banner turns green the moment the last
one lands.

### Orders & attendees

`OrderManager` is the view the Refund button lives in: who bought what, how
many of their tickets still admit somebody, and how many were checked in.

Refunding is irreversible and voids QR codes, so the button opens an inline
confirmation naming the amount and the ticket count, with an optional reason
that reaches the attendee in their refund email. `refundable` comes from the
API and mirrors the rule the endpoint enforces, so a button is disabled rather
than offering an action certain to fail.

A completed refund bumps a counter on the page that re-keys the analytics and
ticket-type panels. A refund returns inventory, and leaving those panels showing
the pre-refund figures would have them contradict the row directly above.

### What the attendee sees

The public event page reads `paid_sales_active` and `paid_sales_required`. When
activation is outstanding it shows a banner explaining that paid tickets are not
on sale yet, and the paid tiers are not offered — the checkout would refuse them
anyway, and a selector that lets somebody build a basket it will reject is worse
than one showing only what can be bought. If the event has a free tier, that
stays bookable and the banner says so.


## The admin portal, recovery pages and uploads (Phase 12)

### /admin

Restricted to `platform_admin`. The route is in `proxy.ts`'s matcher, but only
for the token cookie: a route matcher cannot read a role out of a cookie, and
pretending otherwise would be security theatre. The portal itself checks the
role to decide what to *render*, and the API enforces it on every request. An
organizer who navigates there is told plainly that it is not for them.

Searching is debounced by 300 ms, so typing an email is one request rather than
eight, and every in-flight request is aborted when the next keystroke lands.

The CSV download cannot be a plain `<a href>`: the report needs a bearer token,
and a navigation cannot carry one. The file is fetched with the header
attached, turned into a blob, clicked, and the object URL revoked - rather than
leaking a reference to the whole file for the life of the page.

The portal carries its own status pill. `StatusBadge` is typed to event
statuses, and widening it to `string` to serve four different kinds of status
here would throw away the type safety it provides everywhere else.

### Forgot password, reset password, confirm email

Three pages under `(auth)`. `/reset-password` reads `?token=` from the link but
leaves the field editable and visible, because in this MVP the code is printed
to the API console and somebody will be pasting it by hand. `/verify-email`
redeems a token on arrival when the link carries one, which is what clicking it
is meant to do.

The confirmation after "forgot password" never says whether the address had an
account - the API is careful about that, and a page that helpfully said "no
account with that email" would undo it.

The two-password check lives here rather than at the API, which has no business
knowing the user typed it twice.

### Uploading a banner

`ImageUpload` owns the upload and hands back a URL; the form owns the field.
That split means the same control works on the create form and on an edit form
later without either knowing how storage works. The 5 MB cap is checked here
too, so an oversized file is refused before it is sent rather than after a slow
upload - the server enforces it regardless.

Both the banner and the refund policy render on the public event page. They use
a plain `<img>` rather than `next/image`: the URL points at the API's upload
route, which is not a configured image domain, and optimising a banner an
organizer just uploaded buys nothing here.
