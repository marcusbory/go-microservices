# Broker service helpers: JSON utilities

This doc explains the three `Config` helper methods in `broker-service/cmd/api/helpers.go`:

- `(*Config).readJSON`
- `(*Config).writeJSON`
- `(*Config).errorJSON` (wraps `writeJSON`)

These helpers centralize JSON request/response handling so your handlers can stay focused on business logic.

---

## The `JSONResponse` type (shared response shape)

Many endpoints want a consistent “envelope” for responses (especially errors). This service uses:

- **`Error`** (`bool`): whether the response represents an error
- **`Message`** (`string`): human-readable message (often `err.Error()`)
- **`Data`** (`any`, `omitempty`): optional payload; omitted when empty

`errorJSON` uses this type, but `writeJSON` can write *any* JSON-serializable value.

---

## `(*Config).readJSON(w, r, data) error`

### What it does

Reads the request body, **decodes JSON into `data`**, and validates that the body contains **exactly one JSON value**.

### How it works (key steps)

- **Limits body size to 1MB**
  - Uses `http.MaxBytesReader` to cap the body at `1_048_576` bytes.
  - **Why**: prevents very large bodies from tying up memory/CPU (a common API hardening step).

- **Decodes the JSON**
  - Creates a `json.Decoder` and calls `Decode(data)`.
  - `data` is typically a pointer to a struct, e.g. `&requestPayload`.

- **Ensures “single JSON value”**
  - Calls `Decode(&struct{}{})` a second time.
  - The *only* acceptable error at this point is `io.EOF`, meaning there was nothing left to decode.
  - If it’s not `io.EOF`, the body had extra JSON (or trailing non-whitespace), so it returns:
    - `body must have only a single JSON value`

### FAQ (the “why/how” questions)

#### 1) Why is `w http.ResponseWriter` passed in, if we’re “just reading” JSON?

It’s used here:

- `http.MaxBytesReader(w, r.Body, maxBytes)`

This wraps `r.Body` with a size limit (1MB). `w` is part of that limiter’s API, so the server can handle “body too large” situations correctly.


#### 2) What’s the difference between `r *http.Request` and `data any`?

- **`r`**: the incoming HTTP request. The JSON payload you care about lives in **`r.Body`** (a stream of bytes).
- **`data`**: the destination value to fill with decoded JSON. In practice you pass a **pointer** (e.g. `&myStruct`).

So: **`r.Body` is the source**, **`data` is the destination**.

#### 3) Why do we “read `r` then decode JSON into `data`”?

Because `dec.Decode(data)`:

- reads bytes from `r.Body`
- parses one JSON value
- **stores the result into the value at the address you pass** (typically a pointer like `&payload`)

That’s the key mental model: **decode reads from the request body and writes into your Go value**.

#### 4) How does the decoder work? Why decode into an empty struct, and what does `io.EOF` tell us?

- `json.NewDecoder(r.Body)` creates a streaming JSON decoder that reads from the request body.
- The first `Decode(data)` reads **one** JSON value (e.g. one object `{...}`) and fills your Go value.
- The second `Decode(&struct{}{})` is a *probe* to see if there is **another** JSON value after the first.
  - If there is nothing left (besides whitespace), the decoder hits end-of-input and returns **`io.EOF`** — that’s the “good” case.
  - If it returns anything other than `io.EOF`, there was extra content (another JSON value or junk), so we reject the body with `"body must have only a single JSON value"`.

#### 5) Why “single JSON value” matters

- It prevents ambiguous/abusive payloads like:
  - `{"a":1}{"b":2}` (two JSON objects back-to-back)
  - `{"a":1}   {"b":2}`
- With this rule, the API enforces “one request, one payload”.

### Common errors you might see

- **Invalid JSON**: malformed JSON, type mismatches, etc. (`Decode` returns an error)
- **Too large**: if the body exceeds the max bytes, the decoder will error (often surfaced as “request body too large”)
- **Multiple values**: triggers the explicit `"body must have only a single JSON value"` error

---

## `(*Config).writeJSON(w, status, data, headers...) error`

### What it does

Marshals `data` to JSON and writes it to the response with:

- HTTP status code you provide
- `Content-Type: application/json`
- optional additional headers (variadic)

### How it works (key steps)

- **Marshals to JSON**
  - Calls `json.Marshal(data)` to produce the response body bytes.
  - If marshaling fails (e.g., `data` contains unsupported values like functions or circular references), it returns the error and writes nothing.

- **Applies optional headers**
  - The function accepts a variadic `headers ...http.Header`.
  - If at least one header map is provided, it copies the *first* one (`headers[0]`) into `w.Header()`.
  - **Why variadic**: makes the call site convenient when you need headers sometimes, but not always.
  - **Important detail**: only the first header argument is used; any additional `headers[1:]` are ignored by this implementation.

- **Sets content type + status code**
  - Sets `Content-Type` to `application/json`.
  - Calls `w.WriteHeader(status)` to send the HTTP status code.

- **Writes the JSON bytes**
  - Calls `w.Write(out)` and returns any write error (rare, but possible if the client disconnects mid-response).

### Typical usage

- **Success response**: write a payload struct/map or a `JSONResponse` with `Error=false`.
- **Error response**: most handlers should call `errorJSON`, which uses `writeJSON` under the hood.

---

## `(*Config).errorJSON(w, err, status...) error`

### What it does

Builds a consistent JSON error response and writes it using `writeJSON`.

### How it works (key steps)

- **Chooses a status code**
  - Defaults to `http.StatusInternalServerError` (500).
  - If a status code is passed (e.g., `http.StatusBadRequest`), it uses the first one.
  - Like `writeJSON`, it uses only the first variadic value (`status[0]`).

- **Builds a `JSONResponse`**
  - Sets `payload.Error = true`
  - Sets `payload.Message = err.Error()`
  - Leaves `payload.Data` empty (so it will be omitted in JSON due to `omitempty`)

- **Delegates output to `writeJSON`**
  - Calls `app.writeJSON(w, statusCode, payload)`
  - This ensures the same `Content-Type` and JSON encoding path is used everywhere.

### Typical usage

In a handler, when something goes wrong:

- You construct/receive an `error`
- You call `app.errorJSON(w, err)` or `app.errorJSON(w, err, http.StatusBadRequest)`
- The client always receives a predictable JSON shape:
  - `{ "error": true, "message": "..." }`


