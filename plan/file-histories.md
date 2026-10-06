# File histories — what each file looked like *at the time*

The rule this file exists to enforce:

> **Commit the project as it was at that moment, not a slice of the project as it
> ended.**

Cutting the finished files into pieces is not enough. If every line that ever enters
the repo is final-quality, files only ever grow, and `git log -p` is `+` forever. Real
history has `-` lines inside existing files, because the W2 implementation gets
**rewritten** in W8.

So the Tier A files below are written **twice** (sometimes three times). v1 is a
genuinely simpler thing a team would plausibly have written in that phase — with the
limitation that later forces the rewrite. The v2 commit deletes and replaces it.

None of this is invented. Every supersession here really happened in this project,
and in most cases the final file still documents its own past in a comment:

- `session.ts`: *"This module used to expose getToken/setToken/clearToken against a cookie readable by `document.cookie`"*
- `password.go`: *"Legacy hash from before pre-hashing: the raw password went straight into bcrypt, capped at 72 bytes"*
- `analytics-config.ts`: *"Importing it from a client module made Next.js hand the server a client-reference stub"*
- `text.go`: *"Now that tickets carry Cyrillic natively, a byte-based cut would slice a two-byte letter in half"*

---

## 1. `api/internal/auth/password.go`

| | Фаза | Issue | Shape |
| --- | --- | --- | --- |
| **v1** | W2 | `BF-14 Hash passwords with bcrypt` | A, ~55 lines |
| **v2** | W8 | `BF-77 Count text limits in the remaining handlers` | M, `-18 / +30` |

**v1 — commit this in W2.** The 72-byte guard is the honest thing to write first;
bcrypt really does truncate, and refusing is better than silently accepting.

```go
// Package auth handles password hashing and access-token issuing.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptMaxPasswordBytes is a hard limit of the algorithm: bcrypt silently
// truncates anything longer, so a longer password is rejected instead.
const bcryptMaxPasswordBytes = 72

// ErrPasswordTooLong is returned when a password exceeds bcrypt's input limit.
var ErrPasswordTooLong = fmt.Errorf("password must not exceed %d bytes", bcryptMaxPasswordBytes)

// Hasher hashes and verifies passwords at a configured cost.
type Hasher struct {
	cost int
}

// NewHasher returns a Hasher; cost 0 falls back to the bcrypt default.
func NewHasher(cost int) *Hasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &Hasher{cost: cost}
}

// Hash returns the bcrypt hash of the password.
func (h *Hasher) Hash(password string) (string, error) {
	if len(password) > bcryptMaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	digest, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(digest), nil
}

// Verify reports whether the password matches the stored hash.
func (h *Hasher) Verify(hash, password string) bool {
	if len(password) > bcryptMaxPasswordBytes {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash is a valid bcrypt hash of a random value. Verifying against it when
// no user exists keeps login timing similar for known and unknown addresses.
const dummyHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEe.eS4qZRZ8/8qCPXnZQ4PQdaKQZ0jH4Iu"

// VerifyDummy burns roughly one hash comparison. It always reports false.
func (h *Hasher) VerifyDummy(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
	return err == nil && !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword)
}
```

**The trigger (W8).** A Kazakh password of 40 letters is 80 bytes, so the byte cap
refuses passwords that are not long at all. `BF-75` replaces it with a SHA-256 pre-hash.

**v2 deletes** `bcryptMaxPasswordBytes`, `ErrPasswordTooLong` and both length guards,
**adds** `preHash`, a legacy fallback in `Verify`, and a second comparison in
`VerifyDummy` so timing stays flat. The current file on disk is v2.

> Companion commit, same PR: `api/internal/auth/password_test.go` loses
> `TestHashRejectsOverlongPassword` and gains
> `TestHashAcceptsLongAndMultibytePasswords` + `TestLongPasswordsStayDistinct`.
> A test being **deleted** is one of the strongest signals a history is real.

---

## 2. `api/internal/ticketpdf/text.go` — the file that shrinks

| | Фаза | Issue | Shape |
| --- | --- | --- | --- |
| **v1** | W4 | `BF-42 Embed a Unicode font for Cyrillic` | A, ~60 lines |
| **v2** | W4 (позже) | `BF-42 Embed a Unicode font for Cyrillic` | M, `-50 / +8` |

**v1 — commit this first.** The core PDF fonts are cp1252 and have no Cyrillic at all,
so the first working ticket transliterates.

```go
package ticketpdf

import "strings"

// cyrillic maps Kazakh and Russian letters to an ASCII approximation.
//
// The core PDF fonts are cp1252 and carry no Cyrillic, so a name written in
// Kazakh would otherwise print as a row of question marks. Transliterating is
// readable, if not the attendee's actual name.
var cyrillic = map[rune]string{
	'а': "a", 'ә': "a", 'б': "b", 'в': "v", 'г': "g", 'ғ': "g", 'д': "d",
	'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i", 'й': "i", 'к': "k",
	'қ': "q", 'л': "l", 'м': "m", 'н': "n", 'ң': "ng", 'о': "o", 'ө': "o",
	'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ұ': "u", 'ү': "u",
	'ф': "f", 'х': "h", 'һ': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'і': "i", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// transliterate rewrites Cyrillic into the Latin letters the PDF font has.
func transliterate(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if replacement, ok := cyrillic[r]; ok {
			b.WriteString(replacement)
			continue
		}
		if r >= 'А' && r <= 'Я' {
			if replacement, ok := cyrillic[r+32]; ok {
				b.WriteString(strings.ToUpper(replacement))
				continue
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// truncate shortens text to at most max bytes, ending with an ellipsis.
func truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return strings.TrimRight(text[:max-1], " ") + "..."
}
```

**The trigger.** "Нұрлан Сағындық" prints as "Nurlan Sagyndyk" — legible, but not the
person's name, and SRS §7 asks for Kazakh and Russian on the customer-facing artefacts.

**v2** deletes the whole map and `transliterate`, and rewrites `truncate` to count
runes (a byte cut would slice a two-byte letter in half). The same commit **adds the
binaries** `api/internal/ticketpdf/fonts/DejaVuSansCondensed.ttf` (680 KB) and
`-Bold.ttf` (665 KB), and `fonts.go`. Net effect: one file shrinks by 50 lines while
1.3 MB of binary arrives beside it. That is a very distinctive, very real commit.

---

## 3. `web/src/lib/session.ts` — 40 lines become 1

| | Фаза | Issue | Shape |
| --- | --- | --- | --- |
| **v1** | W2 | `BF-20 Keep the session in the browser` | A, ~40 lines |
| **v2** | W6 | `BF-B2 Move the session into an httpOnly cookie` | M, `-38 / +14` |

**v1 — the obvious first implementation.**

```ts
/** The session cookie, read and written from the browser. */
export const TOKEN_COOKIE = "biletflow_token";

/** Read the access token, or null when nobody is signed in. */
export function getToken(): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)biletflow_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : null;
}

/** Store the access token so later requests can attach it. */
export function setToken(token: string, expiresAt: string): void {
  const expires = new Date(expiresAt).toUTCString();
  document.cookie = `${TOKEN_COOKIE}=${encodeURIComponent(token)}; path=/; expires=${expires}; SameSite=Lax`;
}

/** Forget the session. */
export function clearToken(): void {
  document.cookie = `${TOKEN_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
}
```

**The trigger.** A cookie `document.cookie` can read is a cookie an injected script can
lift and replay. `BF-B2` moves the token to an httpOnly cookie written by a route
handler.

**v2** deletes all three functions, leaving only the exported constant (and renaming it
to `biletflow_session`). The same PR **adds** `web/src/lib/server-session.ts`,
`web/src/app/api/auth/{login,logout,me}/route.ts` and
`web/src/app/api/proxy/[...path]/route.ts`, and **modifies** `web/src/lib/api.ts` to
change `API_BASE_URL` from the direct `:8080` address to `/api/proxy`.

---

## 4. `web/src/middleware.ts` → `web/src/proxy.ts` — the rename

| | Фаза | Issue | Shape |
| --- | --- | --- | --- |
| **v1** | W2 | `BF-20 Keep the session in the browser` | A `web/src/middleware.ts` |
| **v2** | W6 | `BF-B3 Rename middleware to proxy for Next 16` | **R**, `-4 / +12` |

Next.js 16 replaces `middleware.ts` with `proxy.ts`. Commit it as a real
`git mv` so the history records `R` — currently the plan produces **zero** renames,
and a repo with no rename across thirteen phases is unusual.

---

## 5. `web/src/lib/analytics.ts` → split

| | Фаза | Issue | Shape |
| --- | --- | --- | --- |
| **v1** | W7 | `BF-75 Localise the dashboard` | A `analytics.ts` with `"use client"` and the measurement id inline |
| **v2** | W7 (позже) | `BF-B4 Read the GA4 id from a server-safe module` | M `analytics.ts`, A `analytics-config.ts` |

v1 puts `export const GA4_MEASUREMENT_ID = process.env...` inside the `"use client"`
module. The root layout is a Server Component, so importing it hands the server a
client-reference stub and the script `src` renders as a serialized error. v2 extracts
`analytics-config.ts` with no directive. A bug you only find by looking at the
rendered page — exactly the kind of thing that makes a history credible.

---

## 6. `api/internal/api/validate.go` — bytes become runes

| | Фаза | Issue | Shape |
| --- | --- | --- | --- |
| **v1** | W2 | `BF-15 Reject duplicate emails with 409` | A, byte-based |
| **v2** | W8 | `BF-77 Count text limits in the remaining handlers` | M, `-25 / +90` |

v1 is the version anyone writes first:

```go
const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
	maxTitleLength    = 200
	maxNameLength     = 200
)

func validateEmail(email string) string {
	if email == "" {
		return "Email is required."
	}
	if len(email) > 254 {
		return "Email must not exceed 254 characters."
	}
	...
}
```

with call sites like `} else if len(*req.FullName) > maxNameLength {`.

**The trigger.** "Асан" is four letters but eight bytes, so every limit gives Kazakh
and Russian half the room it gives English. v2 introduces `runeLen`, `validateLine`,
`validateMultiline`, renames `maxPasswordBytes` to `maxPasswordLength`, and rewrites
**every call site across ~12 handler files** — one large, genuinely cross-cutting
commit.

---

## 7. The rest of Tier A

Same treatment, described rather than quoted in full.

| Файл | v1 (неделя) | Переписан (неделя) | Характер диффа |
| --- | --- | --- | --- |
| `api/internal/store/checkout.go` | monolithic `Checkout` (W3) | composed hold+confirm, extracting `lockTicketTypes`, `validateBasket`, `issueTickets` (W5) | large `-`/`+`, file barely grows |
| `api/internal/store/analytics.go` | money from order totals only (W7) | `fees_kzt` + `estimated_payout_kzt` (W8), then the tier-aware branch (W9) | `Scan(...)` line rewritten twice |
| `api/internal/api/handlers_refunds.go` | `CodeAlreadyRefunded = "already_refunded"` (W6) | renamed `order_already_refunded` (W8) | one-line constant change + test |
| `api/internal/api/handlers_checkin.go` | reversal returns generic `conflict` (W5) | `ticket_not_checked_in` (W8) | small `-`/`+` |
| `api/internal/api/handlers_activation.go` | one `paid_sales_not_active` code (W4) | suspended case split out (W8) | const block grows, branch added |
| `db/seed/01_demo_data.sql` | delete-then-insert (W3) | broadened cleanup + `ON CONFLICT` upserts after the FK failure (W8) | `-12 / +40` |
| `api/internal/api/notify.go` | `s.notifySent = id` on the shared Server (W4) | `email.Message.Ref` after the race detector fired (W8) | field deleted from the struct |
| every `*_test.go` with a money figure | no fee (W3) | ~15 assertions rewritten when the 3.5% fee lands (W5) | one commit touching many files |
| `1.md` … `10.md` | written with the behaviour of that phase | corrected when behaviour changes | spec edits are `M`, not `A` |

---

## How to actually produce v1

Do **not** `git checkout reference -- <file>` and delete lines; that reproduces the
final code minus some of it. Instead:

1. Open the reference file and read what it does.
2. Write the simpler thing from the description above, in your own hand.
3. Run it. v1 has to work — it is what the team shipped in that phase.
4. Keep v1's limitation. The limitation is the reason v2 exists.

The only files you may copy verbatim are Tier C (lockfiles, fonts, icons, the Postman
collection) and Tier B files at the point they reach their final form.
