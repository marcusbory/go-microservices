# Broker service handlers (notes)

This doc explains the code in `broker-service/cmd/api/handlers.go` and how it connects to routing in `broker-service/cmd/api/routes.go`.

## What is the broker service doing?

The broker acts as a **single entry point** for a client (often a frontend) to request work that is actually performed by other microservices. In this repo, the broker currently supports the `"auth"` action, which it fulfills by **calling the authentication service over HTTP**.

Think of the broker as:

- **HTTP API layer / gateway**: receives requests from the outside world
- **dispatcher**: looks at a request field (here: `action`) and routes the request to the right downstream service
- **translator**: forwards JSON to another service and returns a normalized response to the original caller

## `Config` in the broker

In `broker-service/cmd/api/main.go`, the broker defines:

- `type Config struct {}` (currently empty)
- `app := Config{}` and uses it as the receiver for methods like `app.routes()`

Even though `Config` is empty right now, it’s used as a **method namespace + dependency container**:

- As you add dependencies (db connection pool, logger, clients, config values), they typically become fields on `Config`.
- Handler/helper functions become methods on `*Config` so they can access those shared dependencies.

Important Go detail: other microservices can also have a `type Config struct { ... }` in their own `package main`. Those are **not the same type** as the broker’s `Config`—they just share the same name in different packages/directories.

## Routes: how requests reach handlers

`broker-service/cmd/api/routes.go` wires up the broker’s HTTP endpoints using `chi`:

- `POST /` → `app.Broker`
- `POST /handle` → `app.HandleSubmission`

So, `HandleSubmission` does not automatically receive “all submissions” to the broker—only requests sent to `POST /handle`.

## `HandleSubmission`: a single endpoint that dispatches by `action`

`HandleSubmission` reads a JSON body into `RequestPayload`:

- `action` determines what the broker should do
- `auth` holds auth-specific fields, but only when `action == "auth"`

Then it switches on `requestPayload.Action`:

- `"auth"` → `app.authenticate(...)`
- default → returns `"unknown action"`

This is a common “broker” pattern:

- **one endpoint** (`/handle`)
- **many actions** (auth/log/mail/…)
- each action fans out to an internal method that calls the corresponding downstream service

Example request body for auth:

```json
{
  "action": "auth",
  "auth": {
    "email": "you@example.com",
    "password": "your-password"
  }
}
```

## `authenticate`: broker → authentication-service (internal HTTP call)

`authenticate` is *not* the authentication service itself. It is the broker’s helper that:

1. marshals the `AuthPayload` to JSON
2. makes an HTTP `POST` request to `http://authentication-service/authenticate`
3. checks the downstream status code:
   - `401` → returns `invalid credentials`
   - anything other than `202 Accepted` → returns a generic error
4. decodes the downstream JSON into `JSONResponse`
5. returns its own `202 Accepted` response to the original caller with:
   - message `"Authenticated!"`
   - `Data` forwarded from the auth service’s `JSONResponse.Data`

The hostname `authentication-service` is typically resolvable **inside Docker Compose / the container network** (service name DNS). That’s why the URL looks like a bare service name instead of `localhost`.

## Why the “similar method names” feeling happens

In microservice codebases it’s normal to see repeated concepts:

- each service has its own router setup
- each service has “handler” functions
- each service may have a `Config`-like struct

They’re similar because they all solve the same *kind* of problem (serve HTTP), but they’re intentionally **separate** because each service is its own deployable unit and should not rely on in-process function calls across services.

## How to extend the broker

To add a new brokered action (e.g. logs), you typically:

- extend `RequestPayload` with a new field (e.g. `Log LogPayload \`json:"log,omitempty"\``)
- add a new `case` in `HandleSubmission` (e.g. `case "log": app.logItem(...)`)
- implement `app.logItem` to call the appropriate downstream service endpoint

