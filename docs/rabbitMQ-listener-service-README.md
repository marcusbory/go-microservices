# RabbitMQ logging flow (Broker → Listener → Logger)

This document explains how `broker-service` publishes log events to RabbitMQ and how `listener-service` consumes those events and forwards them to `logger-service`.

## Architecture at a glance

- **Broker service (`broker-service/`)**: receives HTTP requests, and for `"log"` actions it **publishes** an event to RabbitMQ (instead of calling `logger-service` synchronously).
- **RabbitMQ**: routes published events via a **topic exchange** named `logs_topic`.
- **Listener service (`listener-service/`)**: creates a consumer that **binds** to specific routing keys (topics) and **consumes** messages; for each message it calls `logger-service`.
- **Logger service (`logger-service/`)**: persists logs (the listener calls `POST /log` on it).

## RabbitMQ primitives used in this repo

### Exchange: `logs_topic` (type: topic)

Declared in both broker and listener event packages:

- `broker-service/event/event.go`: `declareExchange(ch)`
- `listener-service/event/event.go`: `declareExchange(ch)`

Key properties (from `declareExchange`):

- **name**: `logs_topic`
- **type**: `topic` (routing key pattern matching)
- **durable**: `true` (survives broker restarts)

### Queues: random, exclusive (created by Listener)

The listener does **not** consume from a pre-named queue. Instead it declares a random queue and binds it to topics:

- `listener-service/event/event.go`: `declareRandomQueue(ch)` declares:
  - **name**: `""` (RabbitMQ auto-generates a name)
  - **exclusive**: `true` (only this connection can use it)
  - **durable**: `false`

This pattern is common for “fan-out-ish” consumption: each listener instance gets its own queue bound to the exchange topics.

## End-to-end flow (logging)

### 1) Broker connects to RabbitMQ

Broker startup connects to RabbitMQ with backoff:

- `broker-service/cmd/api/main.go`: `connect()` dials `amqp://guest:guest@rabbitmq` and backs off up to 5 retries.

Listener uses the same connection pattern in `listener-service/main.go`.

### 2) Broker HTTP `"log"` action publishes an event

On `POST` to Broker’s submission endpoint, `HandleSubmission` routes `"log"` to RabbitMQ publishing:

- `broker-service/cmd/api/handlers.go`: `HandleSubmission` → `app.logEventViaRabbit(w, requestPayload.Log)`
- `broker-service/cmd/api/rabbitHandlers.go`: `logEventViaRabbit` → `pushToQueue`

`pushToQueue`:

- Creates an `event.Emitter` with `event.NewEventEmitter(app.Rabbit)`
- Marshals a JSON payload of shape:

```json
{
  "name": "<name>",
  "data": "<data>"
}
```

- Publishes to exchange `logs_topic` with routing key (severity) `log.INFO`:
  - `broker-service/event/emitter.go`: `Emitter.Push(event, severity)`

### 3) Listener binds to routing keys and consumes

Listener startup:

- `listener-service/main.go`: `consumer.Listen([]string{"log.INFO", "log.WARNING", "log.ERROR"})`

Consumption flow:

- `listener-service/event/consumer.go`:
  - declares a random exclusive queue
  - binds it to the exchange `logs_topic` for each routing key provided
  - consumes messages from that queue
  - unmarshals message body into:

```go
type Payload struct {
  Name string `json:"name"`
  Data string `json:"data"`
}
```

### 4) Listener forwards payload to logger-service

For payloads with `Name` `"log"` or `"event"` (and also default case), listener calls:

- `listener-service/event/consumer.go`: `logEvent(payload)` → `POST http://logger-service/log`

This preserves your “logger-service owns persistence” design, but moves the call off the broker’s critical path.

## Answers to your 5 questions

### 1) Broker connects to RabbitMQ using `connect()` (backoff)

**Confirmed.** Broker’s `main()` connects once at startup, stores the connection in `Config{Rabbit: rabbitConn}`, and reuses it for emitting events.

Relevant code:

- `broker-service/cmd/api/main.go`: connect loop + exponential backoff.

### 2) What does `broker-service/event/event.go` do?

**It’s a small RabbitMQ “infrastructure” helper module** that defines how your system declares:

- the shared **exchange** (`declareExchange`) named `logs_topic` of type `topic`, durable
- a **random exclusive queue** (`declareRandomQueue`) used by consumers (mainly the listener)

It centralizes these declarations so both emitter/consumer code uses the same exchange name/type and the same queue-declare policy.

### 3) Deleting the consumer from Broker because Broker only emits

**Confirmed.** In this architecture:

- `broker-service` is a **publisher** (emits events)
- `listener-service` is the **consumer** (receives events and triggers side effects)

So a consumer in `broker-service` would be redundant unless you explicitly wanted broker to also react to events.

### 4) `broker-service/event/emitter.go` is Broker-specific (queues work)

**Confirmed.** `Emitter` is a broker-side abstraction that publishes messages to RabbitMQ (`logs_topic`) with a routing key (severity/topic). That’s exactly the broker’s role in your design: queue work/events for asynchronous processing.

### 5) `rabbitHandlers.go` replaces direct logger calls by emitting events

**Confirmed.** Previously, `handlers.go:logItem` directly called `logger-service` synchronously.

Now:

- Broker `"log"` action calls `logEventViaRabbit` (RabbitMQ publish)
- Listener consumes and calls `logger-service`

Net effect: logging is **asynchronous** from the broker’s perspective and decoupled by RabbitMQ.

## Notes / small gotchas to be aware of

- **Routing keys must match**: Listener binds to `log.INFO|log.WARNING|log.ERROR`. Broker currently publishes only `log.INFO` (hardcoded in `pushToQueue`). If you want warnings/errors, publish with the corresponding routing key.
- **Emitter creation per request**: `pushToQueue` creates a new `Emitter` per call. That’s fine for learning and correctness; later you might keep one emitter around if you want to avoid re-declaring exchanges repeatedly.
- **Auto-ack**: listener consumes with auto-ack (`ch.Consume(..., autoAck=true, ...)`). If `logger-service` is down, you’ll still ack the message. If you need reliability, switch to manual ack + retry/dead-lettering.

