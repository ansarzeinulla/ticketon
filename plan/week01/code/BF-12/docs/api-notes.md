# API notes

Scratch notes while the API takes shape. The agreed contract lives in
`api/README.md`; anything here that survives moves there.

## Conventions

- JSON in, JSON out. Errors always use one envelope:
  `{"error": {"code": "...", "message": "..."}}`.
- Clients switch on `code`, never on `message`.
- Versioned under `/api/v1` once real endpoints exist. `/health` stays unversioned
  so load balancers and CI can call it.

## Local database

`make up` starts PostgreSQL on port **5433** (5432 is often taken by a local
install). The API reads `DATABASE_URL` from `.env`.

## To decide

- Access tokens: JWT in a header, or an httpOnly cookie for the web app?
- Pagination: offset/limit or cursor?
