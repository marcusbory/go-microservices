# RPC vs RabbitMQ in this project

This write-up clarifies why you have two “logging paths” in `broker-service` (RPC vs RabbitMQ), what each mechanism is *for*, and a few common misunderstandings.

**Note**: RPC in Golang requires both microservices communicating to be **written in Golang**. This can be resolved by switching to gRPC (`protoc` generates code for other languages' backends).

## The one-sentence difference

- **RPC**: *call a specific service and wait for a result* (synchronous request/response).
- **RabbitMQ**: *publish a message and let consumers process it later* (asynchronous messaging).

## What “synchronous” and “asynchronous” mean here

- **Synchronous (RPC / direct HTTP)**: the broker receives an HTTP request, then **blocks** until the downstream call completes (or times out), then returns a final answer to the client.
- **Asynchronous (RabbitMQ)**: the broker receives an HTTP request, **queues work** to RabbitMQ, returns quickly (often `202 Accepted`), and the real work happens later in a consumer (`listener-service`).

## Your two logging implementations (why they both exist)

### Logging via RabbitMQ (async)

- **Broker**: publishes a message to the topic exchange `logs_topic` with a routing key like `log.INFO`.
- **Listener**: consumes messages and forwards them to `logger-service`.

This is the “queue a log job” pipeline documented in `docs/rabbitMQ-listener-service-README.md`.

### Logging via RPC (sync)

- **Broker**: dials the logger RPC server and calls something like `RPCServer.LogInfo`.
- **Logger**: handles the RPC call immediately and returns a response string.

This is a classic request/response integration: it’s useful to learn RPC, and it’s appropriate when you *want* an immediate success/failure answer.

## Common misunderstanding: “Can RPC be asynchronous?”

RPC is *designed* for request/response, so **the normal RPC call is synchronous** from the caller’s point of view (the broker waits).

You *can* make a system “asynchronous overall” while still using RPC somewhere internally, but you need a queue boundary:

- Broker publishes to RabbitMQ (async boundary)
- Listener consumes
- Listener calls logger via RPC (sync call, but off the broker’s critical path)

In that design, **RabbitMQ provides the async behavior**; RPC is just the transport between internal services.

## When to use which (practical guidance)

- **Use RabbitMQ when**:
  - you want **fire-and-forget** behavior (client doesn’t need immediate confirmation of final work)
  - you want to **decouple** services (broker shouldn’t depend on logger availability)
  - you want **buffering / smoothing spikes** (queue absorbs bursts)
  - you want **fan-out** (multiple consumers can bind to the same exchange/topics)

- **Use RPC when**:
  - you need an **immediate answer** (success/failure/data) as part of the request
  - the operation is **part of a synchronous workflow** (e.g., authentication checks)
  - the coupling is acceptable and you prefer the simplicity of direct calls

## Why “RPC to publish to RabbitMQ” is usually unnecessary

You *can* do:

- Broker → RPC call to an “event publisher” service → publish to RabbitMQ → Listener consumes → Logger logs

But that still means:

- the broker is **waiting on RPC** (so it’s not really “async” from the broker’s perspective)
- you added an extra hop (more moving parts) to do something the broker can already do directly

This pattern only becomes attractive if you intentionally want a dedicated service that owns:

- message schemas
- routing keys and exchange names
- publish retries / observability
- keeping RabbitMQ client logic out of the broker

## Typical “choose one” rule of thumb

- If the client needs a final outcome **now** → choose **RPC** (or direct HTTP).
- If the client just needs “accepted” and work can finish **later** → choose **RabbitMQ**.

