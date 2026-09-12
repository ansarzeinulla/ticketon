# BiletFlow Scanner — Phases 6 & 12

React Native / Expo app for Event Admins to check attendees in at the entrance.

Expo SDK 57, React Native 0.86, expo-router, expo-camera.

---

## Run it

All three backend layers must be running first. From the **repository root**:

```bash
make up        # PostgreSQL
make api-run   # Go API on :8080
```

Then:

```bash
cd mobile
npm install
npx expo start          # press i for iOS, a for Android, or scan the QR
```

### On a real phone

Install **Expo Go**, put the phone on the same Wi-Fi as your machine, and scan
the QR that `expo start` prints.

No configuration is needed: the app derives the API host from Expo's own
`hostUri`, which is already the development machine's LAN address. The login
screen prints the API it resolved to, so a mis-pointed build is obvious
straight away.

To override it — a staging server, or a tunnel — set:

```bash
EXPO_PUBLIC_API_BASE_URL=http://192.168.1.24:8080/api/v1
```

### On a simulator

The **iOS Simulator has no camera**, so QR scanning cannot be exercised there.
Use the "Enter a code by hand" path, or run on a physical device.

---

## The three screens

| Screen | What it does |
| --- | --- |
| `/login` | Event Admin sign-in; shows the resolved API host |
| `/events` | The events this account may scan, with live check-in counts |
| `/scan/[eventId]` | Camera scanner, full-screen result, manual entry |

An Event Admin sees **only** the events an organizer assigned them to
(SRS 4.8). The list comes from `GET /events/scannable`, which returns events
you organize plus events you hold an unrevoked scanner assignment on.

To assign someone, the organizer calls:

```bash
POST /api/v1/events/{id}/staff   {"email": "scanner@biletflow.kz", "role": "event_admin"}
```

---

## Design decisions

### The result screen is the product

A scanner is read at arm's length by someone who is not looking closely, so a
result fills the entire screen. But **colour is never the only signal**: each
state also carries a large symbol (✓ / ✕) and a heading, because roughly one man
in twelve cannot reliably tell the green from the red.

**Green auto-dismisses after 2.6 s; red does not.** A queue has to keep moving,
and a valid admission needs no decision from staff. A refusal does — someone has
to read why, talk to the attendee, and decide what happens next — so the red
screen waits to be dismissed deliberately.

There is also haptic feedback: a short buzz for success, a longer double buzz
for a refusal, because staff often work with the screen at their side.

### Repeat frames are ignored

A camera decodes many frames a second. Without a cooldown, one physical ticket
would fire dozens of check-in requests and every one after the first would come
back "already used" — the app would accuse an attendee of double entry it caused
itself. The scanner ignores repeats of the same code for 3 seconds, and stops
handling frames entirely while a result is up or a request is in flight.

### Manual entry is a real feature, not a debug hatch

A scratched ticket or a dead camera still has to let someone in, and SRS 4.8
requires staff to be able to find an attendee without scanning. It is also the
only way to exercise the gate on a simulator.

### The token is in the keychain

A scanner is a shared device sitting on a table at a venue entrance, so the
access token goes in `expo-secure-store` (Keychain / Keystore), not plain
storage. On launch the stored token is confirmed with `GET /auth/me` before the
app trusts it — an Event Admin whose assignment was revoked should not get in.

### Promotional codes are refused, by name

A campaign QR encodes a link (`https://…/events/gala?c=CMP_…`), not a bare
token, so the API looks for the token inside the URL as well. The scanner then
says "promo code, not a ticket" instead of "unknown code" — staff can tell the
attendee they scanned the poster rather than turning away a valid holder.

### Failures are specific

The red screen says what actually went wrong, because "denied" gives staff
nothing to act on:

| Code | What the attendee is told |
| --- | --- |
| `already_checked_in` | Already used — with the name and the time of first entry |
| `campaign_token` | This is a promotional code, not an admission ticket |
| `wrong_event` | This ticket is for *(the other event)* |
| `ticket_not_valid` | The ticket is cancelled or refunded |
| `unknown_ticket` | The code matches no ticket |
| `network_error` | Offline — check the venue Wi-Fi |

---

## Layout

```
app/
├── _layout.tsx           AuthProvider + navigation stack
├── index.tsx             session gate → /login or /events
├── login.tsx             Event Admin sign-in
├── events.tsx            event selector with live counts
└── scan/[eventId].tsx    camera, manual entry, result handling
components/
└── ScanResultOverlay.tsx the full-screen green / red result
lib/
├── api.ts                the only place that talks to the Go API
├── config.ts             API host resolution
├── session.ts            keychain-backed token storage
├── auth-context.tsx      session restore and sign-in
├── theme.ts              high-contrast palette
└── types.ts              mirrors of the Go response structs
```

## Finding an attendee by name (Phase 12, SRS 4.8)

The screen staff open when a QR will not scan at all: a cracked screen, a dead
battery, a printout left at home. Reachable from the event list and from the
scanner's own footer, which is where somebody is standing when they discover
the code cannot be read.

Typing is debounced by 300 ms and every response carries a sequence number, so
a slow early request cannot overwrite the results of a newer one — the classic
way a search box ends up showing answers to a question the user has moved on
from.

**The list never contains QR tokens.** Staff searching by name are standing
next to the person; handing every door device a working admission credential
for every attendee would defeat the point of the QR code. Check-in goes by
ticket id, which only an already-authorised device could have obtained from the
search.

Admitting somebody here is a **real** check-in. The server runs the same
transaction as a scan, so the same duplicate protection applies: a second tap
is refused exactly as a second scan would be, and the row records
`device_label = "manual search"` so the history distinguishes a typed
admission from a scanned one. A ticket that is already in, cancelled or
refunded is shown with a disabled button rather than one that is certain to be
refused.

## Working offline (SRS 4.8)

A venue door often has no reliable network, so the scanner keeps working without
one. From the event list, **Offline mode** downloads the event's roster once,
then admits tickets against it with the connection off.

**The roster is hashes, never tokens.** The server returns the SHA-256 of each
admission token (`GET /events/{id}/roster`); the device stores those in SQLite
(`lib/offline-db.ts`) and, when a QR is scanned, hashes what it read with
`expo-crypto` and compares hash to hash. A phone left on a table at the entrance
therefore never holds the credential needed to forge a ticket — it can answer
"is this ticket real?" without holding the answer to "what are the real
tickets?".

Every offline admission is written to a local queue and, the moment connectivity
returns (`expo-network`, `lib/use-connectivity.ts`), synced to the server
(`POST /events/{id}/check-in/sync`). The server judges each queued admission
independently and reports back per ticket — `recorded`, `already_checked_in`,
`not_valid` or `unknown_ticket` — so one ticket refunded while the door was
offline cannot discard the good admissions queued behind it, and the winner of a
conflict is always whoever synced first rather than whichever phone's clock was
fastest. A queued admission not yet synced can still be undone at the door; once
the server has it, the reversal goes through the online endpoint.
