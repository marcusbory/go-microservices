# Logger Service: MongoDB Models (`logger-service/data/models.go`)

This doc explains the `LogEntry` model and its MongoDB CRUD helpers, with extra context if you’re coming from Firebase Firestore.

## Quick orientation (Firestore → MongoDB)

- **Database**
  - **Firestore**: one project DB (logical), documents live inside collections.
  - **MongoDB**: multiple databases per server; this service uses database **`logs`**.
- **Collection**
  - **Firestore**: collection.
  - **MongoDB**: collection; this service uses collection **`logs`** inside database `logs`.
- **Document**
  - **Firestore**: JSON-like document with a string ID.
  - **MongoDB**: BSON document; the default primary key is **`_id`**, commonly an **ObjectID** (12-byte value usually shown as a 24-char hex string).
- **Queries**
  - **Firestore**: chained query builder.
  - **MongoDB (Go driver)**: functions like `Find`, `FindOne`, `UpdateOne`, with filters/updates expressed as `bson.*` values.

## File overview

`logger-service/data/models.go` contains:

- A package-level MongoDB client (`client *mongo.Client`) that must be set once via `New`.
- A `Models` struct that groups model helpers (currently only `LogEntry`).
- A `LogEntry` struct with BSON + JSON tags so the same struct can be stored in MongoDB and encoded/decoded in HTTP JSON.
- CRUD-like methods on `*LogEntry` that read/write from the `logs.logs` collection.

## The `Models` container

### `New(mongo *mongo.Client) Models`

- Sets the package-global `client`.
- Returns a `Models` value with `LogEntry` initialized (as an empty value).

This is a simple “service locator” pattern: other packages can hold one `Models` value and access `models.LogEntry.Insert(...)`, etc.

## The `LogEntry` model

### Fields

`LogEntry` represents one log record.

- **`ID`**: stored in MongoDB as `_id` (usually an ObjectID), but exposed to JSON clients as `id`.
- **`Name`**: a short label (e.g. service name).
- **`Data`**: the log payload (often JSON-as-string or message text).
- **`CreatedAt`, `UpdatedAt`**: timestamps stored in MongoDB (and optionally returned to clients).

## Collection used by all methods

Every method uses:

- Database: `client.Database("logs")`
- Collection: `.Collection("logs")`

So the “full path” is **database `logs` → collection `logs`**.

## Methods

### `(*LogEntry) Insert(entry LogEntry) error`

What it does:

- Inserts a new document into the collection.
- Ignores `entry.ID`, and sets `CreatedAt`/`UpdatedAt` to `time.Now()` in the inserted struct.

Notes:

- MongoDB will generate `_id` automatically when omitted (because of `bson:"_id,omitempty"` on `ID`).
- This method currently uses `context.TODO()` (see FAQ for why that’s worth changing).

### `(*LogEntry) All() ([]*LogEntry, error)`

What it does:

- Queries all documents (`bson.D{}` empty filter).
- Sorts by `created_at` descending.
- Iterates the cursor and decodes each document into a `LogEntry`.

Important pieces:

- `opts := options.Find().SetSort(...)` controls sorting.
- `cursor.Next(ctx)` drives iteration.
- `cursor.Decode(&item)` converts BSON → Go struct using the `bson:"..."` tags.

### `(*LogEntry) GetOne(id string) (*LogEntry, error)`

What it does:

- Converts a hex string id into a MongoDB ObjectID via `primitive.ObjectIDFromHex(id)`.
- Runs a `FindOne` filtered by `_id`.
- Decodes into a `LogEntry`.

Why conversion is needed:

- In MongoDB, `_id` is an ObjectID type (binary), not a plain string. The driver needs the typed `ObjectID` in the filter for an `_id` match.

### `(*LogEntry) DropCollection() error`

What it does:

- Drops the entire `logs` collection.

This is destructive and is usually only used for local dev/test.

### `(*LogEntry) Update() (*mongo.UpdateResult, error)`

What it does:

- Converts `l.ID` (string) to an ObjectID.
- Runs `UpdateOne` with a filter `_id = docID`.
- Uses `$set` to update `name`, `data`, and `updated_at`.

Notes:

- `$set` updates only the provided fields; it doesn’t replace the whole document.
- The method returns MongoDB’s `UpdateResult` so callers can inspect matched/modified counts.

## FAQ

### 1) Why are there “two contexts”: `context.TODO()` and `ctx`?

There is only **one package** involved: Go’s standard library `context` package (`import "context"`). The difference you’re seeing is:

- **`context.TODO()` / `context.Background()`**: *functions* that create a `context.Context` value.
- **`ctx`**: just a *variable name* holding a `context.Context` value.

In `All()` you create a timeout context:

- `ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)`

…but the query itself uses a different context:

- `collection.Find(context.TODO(), ...)`

That’s confusing because it means the `Find` call itself does **not** use your timeout/cancellation. Meanwhile, cursor iteration *does* use `ctx`:

- `cursor.Next(ctx)`
- `cursor.Close(ctx)`

**Rule of thumb**: create one request/operation context (`ctx`) and pass it everywhere for that operation (`Find`, `FindOne`, `UpdateOne`, cursor iteration, etc.). Using `context.TODO()` is typically a placeholder when you haven’t wired a “real” context through yet.

### 1a) What’s the difference between `context.Background()` and `context.TODO()`? Why use `context.Background()` here?

- **`context.Background()`**: use this when you need a “root” context and you *don’t* have a better one to derive from (common in `main()`, initialization, tests, or in libraries that must start a new tree of work).
- **`context.TODO()`**: a placeholder root context that signals “we haven’t decided what context should be used here yet”.

They behave the same at runtime (both are empty root contexts with no deadline and no cancellation), but they communicate **different intent** to humans and tooling.

In this logger service, we create:

- `ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)`

We use `context.Background()` because we’re deliberately creating a new operation-scoped context with a deadline. Once you have that `ctx`, the “rule of thumb” is to pass **that** `ctx` into all the MongoDB calls for the operation (instead of mixing in `context.TODO()`).

### 2) Why specify both BSON and JSON tags if MongoDB uses BSON?

You’re using the same struct (`LogEntry`) for **two different encodings**:

- **BSON**: how the MongoDB driver maps struct fields to document fields when reading/writing MongoDB.
  - Controlled by `bson:"..."`
- **JSON**: how Go’s `encoding/json` maps struct fields when you send/receive HTTP payloads.
  - Controlled by `json:"..."`

Even though MongoDB stores BSON, your service almost certainly talks to clients over HTTP using **JSON**, so both tag sets are useful.

Also note the deliberate mapping difference for the ID:

- Mongo: `_id` (`bson:"_id,omitempty"`)
- JSON: `id` (`json:"id,omitempty"`)

That avoids exposing MongoDB’s internal `_id` name to API clients.

### 3) What’s the difference between `bson.M` and `bson.D`?

They represent BSON documents in different Go shapes:

- **`bson.M`**: `map[string]any` (unordered)
  - Great for simple filters like `bson.M{"_id": docID}`
  - You don’t write `{Key: ..., Value: ...}` because it’s just a Go map literal.
- **`bson.D`**: ordered list of elements (`[]bson.E`)
  - Used when **order matters** or when you want to allow duplicate keys (MongoDB commands sometimes rely on order).
  - Elements are `bson.E{Key: "...", Value: ...}`, which is why you see `Key`/`Value`.

In your file:

- Filter uses `bson.M{"_id": docID}` (order doesn’t matter).
- Sort and update use `bson.D`:
  - Sort: `bson.D{{Key: "created_at", Value: -1}}`
  - Update: `bson.D{{Key: "$set", Value: bson.D{...}}}`

**Practical guidance**:

- Use **`bson.M`** for most simple filters.
- Use **`bson.D`** for updates/operators and command-like documents, and when you’re following driver examples that use `bson.D`.

### 4) How does `context.WithTimeout` work? Will the query fail after 15 seconds?

`context.WithTimeout(parent, 15*time.Second)` returns a derived context (`ctx`) that has a **deadline**: “now + 15 seconds”.

What that means in practice:

- If the MongoDB operation finishes before the deadline, everything proceeds normally.
- If the deadline is exceeded, `ctx.Done()` is closed and `ctx.Err()` becomes `context.DeadlineExceeded`.
- The MongoDB Go driver uses that context to **cancel in-flight work** and return an error from the call that was using the context (e.g. `Find`, `FindOne`, `UpdateOne`, or even `cursor.Next` during iteration).

So yes: if the query (or cursor iteration) takes longer than ~15 seconds, you should expect the call to return an error consistent with a timeout/cancellation (commonly `context deadline exceeded`, sometimes wrapped by the driver).

### 5) I see `context.TODO()` in `cmd/api/main.go` for `Connect`, `Ping`, and `Disconnect`. Should those use `ctx` too?

Generally, yes.

In `main()` / startup code, `context.TODO()` often appears early in a project as a placeholder. But for operations like:

- `mongo.Connect(...)`
- `client.Ping(...)`
- `client.Disconnect(...)`

it’s usually better to create an explicit timeout context and pass that same `ctx` into the whole operation so:

- connection attempts don’t hang indefinitely
- ping has a clear deadline
- disconnect has a bounded time to clean up resources

In this repo we updated `connectToMongo()` to create a `ctx` with `context.WithTimeout(context.Background(), 15*time.Second)` and use that `ctx` for both `Connect` and `Ping`, and we removed the extra `Disconnect(context.TODO())` in `main()` in favor of a single deferred disconnect with a timeout context.

