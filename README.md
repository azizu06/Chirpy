# Chirpy

A RESTful HTTP API for a Twitter-like microblogging service, written in Go with
no web framework — just the standard library's `net/http` and a PostgreSQL
backend. Users can register, authenticate, post short messages ("chirps"),
manage their accounts, and upgrade to a premium "Chirpy Red" membership.

> Built while working through [Boot.dev](https://boot.dev)'s "Learn HTTP Servers
> in Go" course as a hands-on way to learn Go's standard-library web stack,
> JWT/refresh-token auth, type-safe SQL, and database migrations.

## Why it's interesting

- **No framework.** Routing, middleware, and JSON handling are built directly on
  `net/http` and Go 1.22+ method-based pattern matching (`POST /api/chirps`,
  `DELETE /api/chirps/{chirpID}`), so the request lifecycle is fully visible.
- **Real authentication.** Password hashing with bcrypt, short-lived JWT access
  tokens (1 hour), and long-lived refresh tokens (60 days) with revocation.
- **Type-safe database access.** SQL queries are compiled into type-checked Go
  with [sqlc](https://sqlc.dev); schema changes are versioned with
  [goose](https://github.com/pressly/goose) up/down migrations.
- **Webhook security.** A Polka payment webhook upgrades users to Chirpy Red,
  gated by an API key so only the payment provider can trigger it.

## Tech stack

| Concern        | Choice                                   |
| -------------- | ---------------------------------------- |
| Language       | Go 1.26                                  |
| HTTP           | `net/http` (standard library)            |
| Database       | PostgreSQL                               |
| DB access      | [sqlc](https://sqlc.dev) (generated Go)  |
| Migrations     | [goose](https://github.com/pressly/goose)|
| Auth           | `golang-jwt`, `bcrypt`, `crypto/rand`    |

## Prerequisites

- [Go](https://go.dev/dl/) 1.22 or newer
- [PostgreSQL](https://www.postgresql.org/) running locally
- [goose](https://github.com/pressly/goose) — `go install github.com/pressly/goose/v3/cmd/goose@latest`
- [sqlc](https://sqlc.dev) (only needed if you change SQL) — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

## Setup

1. **Clone and install dependencies**

   ```bash
   git clone https://github.com/azizu06/Chirpy.git
   cd Chirpy
   go mod download
   ```

2. **Create the database**

   ```bash
   createdb chirpy
   ```

3. **Configure environment variables** — create a `.env` file in the project root:

   ```env
   DB_URL="postgres://username:password@localhost:5432/chirpy?sslmode=disable"
   JWT_SECRET="a-long-random-secret-string"
   POLKA_KEY=your-polka-api-key
   PLATFORM=dev
   ```

   - `DB_URL` — PostgreSQL connection string.
   - `JWT_SECRET` — secret used to sign access tokens. Generate one with
     `openssl rand -base64 64`.
   - `POLKA_KEY` — API key the payment webhook must present.
   - `PLATFORM` — set to `dev` to enable the admin reset endpoint.

4. **Run the migrations**

   ```bash
   goose -dir sql/schema postgres "$DB_URL" up
   ```

5. **Run the server**

   ```bash
   go run .
   ```

   The server listens on `http://localhost:8080`. Static files are served from
   `/app/`.

## API reference

### Health & admin

| Method | Endpoint          | Description                                   |
| ------ | ----------------- | --------------------------------------------- |
| GET    | `/api/healthz`    | Liveness check (returns `OK`).                |
| GET    | `/admin/metrics`  | HTML page with the file-server hit count.     |
| POST   | `/admin/reset`    | Deletes all users (only when `PLATFORM=dev`). |

### Users & auth

| Method | Endpoint        | Auth          | Description                               |
| ------ | --------------- | ------------- | ----------------------------------------- |
| POST   | `/api/users`    | —             | Create a user (email + password).         |
| PUT    | `/api/users`    | Access token  | Update the authenticated user's email/password. |
| POST   | `/api/login`    | —             | Log in; returns an access + refresh token.|
| POST   | `/api/refresh`  | Refresh token | Mint a new access token.                  |
| POST   | `/api/revoke`   | Refresh token | Revoke a refresh token.                   |

### Chirps

| Method | Endpoint                  | Auth         | Description                                 |
| ------ | ------------------------- | ------------ | ------------------------------------------- |
| POST   | `/api/chirps`             | Access token | Create a chirp (max 140 chars).             |
| GET    | `/api/chirps`             | —            | List chirps. Supports `?author_id=` and `?sort=asc\|desc`. |
| GET    | `/api/chirps/{chirpID}`   | —            | Get a single chirp.                         |
| DELETE | `/api/chirps/{chirpID}`   | Access token | Delete a chirp (author only).               |

### Webhooks

| Method | Endpoint                | Auth     | Description                                   |
| ------ | ----------------------- | -------- | --------------------------------------------- |
| POST   | `/api/polka/webhooks`   | API key  | Upgrade a user to Chirpy Red on `user.upgraded`. |

## Project layout

```
.
├── main.go                 # apiConfig, server setup, route wiring
├── handlers_chirps.go      # chirp endpoints
├── handlers_users.go       # user / auth / webhook endpoints
├── json.go                 # JSON response helpers
├── internal/
│   ├── auth/               # JWT, bcrypt, bearer/API-key parsing
│   └── database/           # sqlc-generated query code
├── sql/
│   ├── schema/             # goose migrations
│   └── queries/            # sqlc query definitions
└── sqlc.yaml
```

## Authentication flow

1. `POST /api/login` returns a **JWT access token** (1-hour expiry) and a
   **refresh token** (60-day expiry, stored in the database).
2. Send the access token as `Authorization: Bearer <token>` on protected routes.
3. When the access token expires, `POST /api/refresh` with the refresh token to
   get a new one — no need to log in again.
4. `POST /api/revoke` invalidates a refresh token (e.g. on logout).
