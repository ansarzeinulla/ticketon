# BiletFlow

BiletFlow is a Kazakhstan-focused self-service event and ticketing platform.
It allows individuals, student organizations, businesses, and community groups
to create events and distribute digital tickets.

## Product

BiletFlow provides tools for creating events, managing ticket inventory,
registering attendees, processing paid tickets, issuing digital QR tickets,
and verifying tickets at event entrances.

Creating an event and distributing free tickets is free. Organizers who want
to sell paid tickets must activate paid ticket sales and complete identity
and payout verification.

### Main Features

- Event creation and management
- Free and paid tickets
- Digital QR tickets
- Secure ticket verification
- Attendee management
- Event registration
- Calendar export
- Payment processing
- Mobile ticket verification

## Technology Stack

### Backend

- Go
- REST API

Go is used to build the backend services and API layer.

### Database

- PostgreSQL

PostgreSQL stores users, events, tickets, registrations, payments and other
persistent application data.

### Infrastructure

- Docker
- GitHub Actions
- Git / GitHub

Docker provides consistent application environments, while GitHub Actions can
be used for automated testing and continuous integration.

### Configuration

Application configuration is provided through environment variables.
Sensitive values such as database credentials are not committed to the
repository.

## Initial Architecture

The system follows a client-server architecture.

```text
Client Applications
        |
        v
    REST API
        |
        v
   Go Backend
        |
        v
   PostgreSQL
```

The web app (Next.js) and the scanner app (Expo) are both clients of the same
REST API; neither talks to the database.

## Run it locally

Requires Docker, Go 1.25 and Node 24. Three terminals, and a fourth for the
scanner:

```bash
make up && make seed   # PostgreSQL on localhost:5433, schema applied, demo data loaded
make api-run           # API on http://localhost:8080
make web-install && make web-dev   # web on http://localhost:3000
make scan-install && make scan-dev # scanner app in Expo Go (see mobile/README.md)
```

`make help` lists every target.

## Walk through it

With the three terminals running, the project can be used end to end:

1. **Organizer.** Sign in at <http://localhost:3000/login> as `dana@biletflow.kz`
   (password `biletflow-demo`), or register a new account. Create an event,
   add a free and a paid ticket type, and publish it from the dashboard.
2. **Attendee.** In a private window open <http://localhost:3000/events>, pick the
   event, choose tickets and pay. The payment is simulated: nothing is charged.
3. **The tickets.** The order page shows a QR code per ticket and a printable
   PDF; the confirmation email is printed in the API's terminal (there is no
   mail server - every email is printed there and recorded in the
   `notifications` table).
4. **Back to the organizer.** The event's page on the dashboard now lists the
   order and every ticket holder. Paid tickets only go on sale once the
   activation checklist on that page is complete. Under **Gate staff** the
   organizer names who may scan tickets at the entrance.
5. **At the door.** Sign in to the scanner app as `scanner@biletflow.kz` (same
   password), choose the event and scan the ticket's QR - or type its code, or
   find the attendee by name. A second scan of the same ticket is refused.

What each step must do, and what it must refuse, is written down in `1.md`
(accounts), `2.md` (events and ticket types), `3.md` (checkout), `4.md`
(ticket delivery) and `5.md` (check-in at the door).

## Checks

The same commands run in CI on every pull request:

| Command | What it checks |
| --- | --- |
| `make test` | database schema tests |
| `make api-check` | Go formatting, `go vet`, unit and integration tests (needs `make up`) |
| `make api-smoke` | cURL acceptance checks against a running API |
| `make web-check` | ESLint, TypeScript, production build |
| `make scan-check` | TypeScript for the scanner app |

