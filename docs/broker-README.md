# Broker service: Docker + Compose notes

This doc explains two files used to build and run the **broker-service** in containers:

- `broker-service/broker-service.dockerfile`
- `project/docker-compose.yml`

The goal is to help you understand **what each line does**, and **why it exists** in a microservices setup.

---

## `broker-service/broker-service.dockerfile`

This Dockerfile uses a **multi-stage build**:

- **Stage 1 (“builder”)**: compile the Go service into a Linux binary
- **Stage 2 (“runtime”)**: run *only* that binary in a small base image

### Stage 1: build the Go binary

- **`FROM golang:1.25.0-alpine AS builder`**
  - **What**: Uses an Alpine-based image that includes the Go toolchain.
  - **Why**: You need Go installed to compile. You don’t want Go installed in the final runtime image.

- **`RUN mkdir /app`**
  - **What**: Creates a directory inside the container image.
  - **Why**: A predictable location to copy code into and build from.

- **`COPY . /app`**
  - **What**: Copies the build context (your source code) into `/app`.
  - **Why**: The `go build` step runs inside the container and needs the source.

- **`WORKDIR /app`**
  - **What**: Sets the default working directory for subsequent commands.
  - **Why**: Avoids having to prefix commands with `cd /app`.

- **`RUN CGO_ENABLED=0 go build -o brokerApp ./cmd/api`**
  - **What**: Compiles the Go program found at `./cmd/api` into an executable named `brokerApp`.
  - **Why `./cmd/api`**: Common Go project structure: `cmd/<service>` contains the entrypoint (`main` package).
  - **Why `CGO_ENABLED=0`**: Produces a **static** binary (no libc dependency), which runs cleanly in minimal images.

- **`RUN chmod +x /app/brokerApp`**
  - **What**: Ensures the binary is executable.
  - **Why**: Prevents permission issues when the runtime image starts.

### Stage 2: tiny runtime image

- **`FROM alpine:latest`**
  - **What**: Starts a fresh runtime stage from a small base image.
  - **Why**: Smaller images download faster and have a smaller attack surface.

- **`RUN mkdir /app`**
  - **What**: Creates a folder where the binary will live.

- **`COPY --from=builder /app/brokerApp /app`**
  - **What**: Copies only the compiled binary from the builder stage.
  - **Why**: The final image contains the app, not the compiler or source code.

- **`CMD ["/app/brokerApp"]`**
  - **What**: The default process that runs when the container starts.
  - **Why**: Containers are typically “one process”; here, it’s your broker service.

### FAQ

> **Why don't you see `/app` or `/app/brokerApp` locally?**

Because `/app` only exists inside the Docker image/container filesystem.
- During `docker build`:
  - Docker creates temporary layers (an isolated filesystem).
  - Your Dockerfile runs `RUN mkdir /app`, `COPY . /app`, and `go build -o brokerApp ...` inside that image filesystem.
  - That produces `/app/brokerApp` inside the image, not in your Mac folder.
- After the build:
  - The final runtime image only contains what you copied in that stage (here, `/app/brokerApp`).
    - Nothing is automatically copied back out to your host unless you explicitly do it (via a bind mount, `docker cp`, or a multi-stage “export” step).

---

## `project/docker-compose.yml`

Docker Compose is a way to run one or more containers together using a single config file.

### Top-level structure

- **`version: "3"`**
  - **What**: Compose file format version.

- **`services:`**
  - **What**: A list of containers (services) that Compose manages.

### The `broker-service` service

- **`broker-service:`**
  - **What**: The name of the service.
  - **Why**: Compose uses this as the DNS name on the default Compose network and for logs/management.

- **`build:`**
  - **What**: Build an image locally (instead of pulling a prebuilt image).

  - **`context: ../broker-service`**
    - **What**: The directory sent to Docker as “build context”.
    - **Why it matters**: `COPY . /app` in the Dockerfile copies **from the context**. If the context is wrong, your source code won’t be available during build.

  - **`dockerfile: ../broker-service/broker-service.dockerfile`**
    - **What**: The specific Dockerfile path to use.
    - **Why**: Your `docker-compose.yml` is in `project/`, but the Dockerfile lives in `broker-service/`.

- **`restart: always`**
  - **What**: Docker will restart the container if it exits.
  - **Why**: Useful in local dev/demos; services come back after crashes or host reboots.

- **`ports:` → `- "8080:80"`**
  - **What**: Publishes container port `80` to host port `8080`.
    - Left side: **host** (your machine) port
    - Right side: **container** port
  - **Why**: Lets you access the service at `http://localhost:8080` from your browser/curl.
  - **Common gotcha**: This only works if the app inside the container is listening on **port 80**.
    - If your Go server listens on `:8080` internally, you’d typically map `"8080:8080"` instead (or change the app to listen on `:80` in container).

- **`deploy:` / `mode: replicated` / `replicas: 1`**
  - **What**: Swarm/stack deployment settings.
  - **Important**: These are primarily used with **Docker Swarm** (`docker stack deploy`) and may be ignored by plain `docker compose up` depending on your setup.
  - **Why replicas are tricky with ports**: If you publish a fixed host port (like `8080`), you can’t have multiple replicas on the same host all trying to bind that same host port.

---

## Quick mental model (how it all fits together)

- **Dockerfile**: “How do I build and run *one* service image?”
- **Compose**: “How do I run *this* service (and later, many services) together with networking and port publishing?”

---

## Suggested next check

To confirm the port mapping is correct, look for the HTTP server listen address in `broker-service/cmd/api/main.go` (it will be something like `http.ListenAndServe(":80", ...)` or `":8080"`). The container port in Compose should match that internal listen port.

