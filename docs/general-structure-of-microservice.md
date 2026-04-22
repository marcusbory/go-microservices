# General structure of an HTTP microservice (Go)

This repository follows a small, repeatable layout for each HTTP service under `cmd/api/`. The **mail-service** is a good reference implementation: a thin HTTP layer on top of a focused domain capability (sending mail).

## Request lifecycle

1. **`main`** constructs an application object (typically `Config`) and starts `http.Server` with `Handler` set to the router returned by `routes()`.
2. The **router** (`routes.go`) matches the path and method, runs shared middleware (CORS, heartbeat, logging, etc.), then dispatches to a **handler** function.
3. The **handler** (`handlers.go`) decodes the request, validates or maps input to domain types, calls services or helpers, and writes the HTTP response (often JSON).
4. **Domain / infrastructure code** (for example `mailer.go`) performs the real work (SMTP, templates, external APIs) behind a small, explicit API.
5. **Helper** (`helpers.go`) functions are written to define what HTTP clients send and receive

> Nothing in `main` “sends” requests to `routes`; the standard library server invokes the root `http.Handler` (your Chi `mux`) for each incoming connection.

## File roles

### `main.go` — process entry and wiring

- **`func main()`** is the executable entry point: build dependencies, configure the server address and handler, call `ListenAndServe`.
- **`Config` (or similar)** holds dependencies the handlers need (for example `Mailer` in mail-service). Handlers and helpers use a **receiver** on `*Config` (`app.readJSON`, `app.routes`) so dependencies travel explicitly instead of relying on package-level mutable globals.
- **Environment and constants** (ports, SMTP settings) are often read or defaulted here so the same binary can run in different environments (local, Docker, production).

### `routes.go` — HTTP surface area

- Constructs the **multiplexer** (here, **Chi**): `chi.NewRouter()`.
- Attaches **middleware** that applies to many routes: CORS, `/ping` heartbeat, optional logging or auth.
- Registers **routes** by mapping `(method, path)` to handler methods on `app`.

This file should stay mostly declarative: “what URLs exist and which middleware applies,” not business rules.

### `handlers.go` — HTTP adapters

- **First application-specific layer** after routing: parse query/body (often JSON), map to internal structs, pick status codes, call domain logic.
- Should remain **thin**: decode → call service → encode response or error. Heavy logic belongs in dedicated packages or types (for example `Mail` in `mailer.go`).

Multiple handler files (`handlers_mail.go`, etc.) are fine once a service grows.

### `mailer.go` (example domain module) — capability implementation

- Encapsulates one responsibility (here: build MIME parts from templates and send via SMTP).
- **`SendSMTPMessage`** is the main operation callers use. Other methods may exist as **unexported** helpers (`buildHTMLMessage`, `getEncryption`) or as part of the same type; the important idea is a **small public surface** for the rest of the service.
- Exported types such as `Message` describe the input contract for that capability.

In other services this file might be named after the domain (`payments.go`, `auth.go`) or split into a `internal/` package.

### `helpers.go` — shared HTTP utilities

- **Read/write JSON**, consistent error payloads, optional header helpers.
- These functions define much of what **API clients** see (status codes, JSON shape). They are not “frontend” code; they are still server-side, but they shape the **external HTTP contract**.

Keeping JSON and error formatting here avoids duplicating response structure in every handler.

## Patterns used across services

- **Same package (`main`)** for `cmd/api`: simple binaries often keep router, handlers, and helpers in `package main` for a flat layout. Larger services usually move non-`main` code under `internal/...` with proper package names.
- **Chi + CORS + heartbeat** appears in both broker-service and mail-service for a consistent operational baseline (`/ping` for health checks).

## When to split further

Consider extracting packages when you have:

- multiple binaries reusing the same logic;
- large test suites that benefit from smaller units;
- clear boundaries (e.g. `internal/mail`, `internal/httpapi`).

Until then, the five-file layout keeps the flow easy to follow: **main → routes → handlers → domain (mailer)**, with **helpers** supporting handlers throughout.
