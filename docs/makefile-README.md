# Makefile commands (project/)

This repo uses a Makefile at `project/Makefile` to provide short, memorable commands for common workflows (running containers, building images, building/running the front-end binary).

If you run `make` from the repo root, it’s common to add a small root `Makefile` that forwards to `project/Makefile` (e.g. `make -C project up_build`). If you don’t have that forwarding Makefile yet, run these commands from the `project/` directory.

---

## Variables

- **`COMPOSE`**
  - Set to `docker compose`.
  - Used so you can change the Compose command in one place if needed.

- **`FRONT_END_BINARY`**
  - Name of the front-end binary (`frontApp`).

- **`BROKER_BINARY`**
  - Name of the broker binary (`brokerApp`).
  - Note: in the current setup the broker binary is built *inside Docker* (via the broker Dockerfile), so this variable is mainly informational.

---

## Commands

## `make up`

- **What it does**: starts the Compose services in the background.
- **Command**: `$(COMPOSE) up -d`
- **When to use**: you already built images recently and just want to start containers quickly.
- **Gotcha**: won’t rebuild images unless you also pass `--build`.

## `make up_build`

- **What it does**: stops any running containers, then rebuilds images (as needed) and starts everything.
- **Commands**:
  - `$(COMPOSE) down`
  - `$(COMPOSE) up --build -d`
- **When to use**: after changing Go code / Dockerfiles and you want a clean “rebuild + start”.
- **Note**: with this repo’s broker setup, compilation happens inside Docker (multi-stage Dockerfile), not on your host.

## `make down`

- **What it does**: stops and removes the running containers.
- **Command**: `$(COMPOSE) down`
- **Important**: this does **not** delete images. If you want to remove images too, you’d use `docker compose down --rmi local` (outside this Makefile).

## `make build_broker`

- **What it does**: builds only the `broker-service` image.
- **Command**: `$(COMPOSE) build broker-service`
- **When to use**: you changed only broker code/Dockerfile and want a faster build than rebuilding everything.

## `make build`

- **What it does**: builds all Compose images.
- **Command**: `$(COMPOSE) build`
- **When to use**: you changed multiple services (or want to warm the cache before `up`).

## `make build_front`

- **What it does**: builds the front-end Go binary into the repo-level `bin/` directory.
- **Command**: `cd ../front-end && env CGO_ENABLED=0 go build -o ../bin/${FRONT_END_BINARY} ./cmd/web`
- **Output**: `bin/frontApp`
- **When to use**: you changed front-end Go code and want to rebuild the binary.

## `make start`

- **What it does**: builds the front-end binary (via `build_front`) and then starts it.
- **Key behavior**: runs the binary **from the `front-end/` directory**:
  - `cd ../front-end && ../bin/${FRONT_END_BINARY} &`
- **Why that matters**: the front-end loads templates from disk using relative paths like `./cmd/web/templates/...`, so it must run with a working directory where those paths exist.

## `make stop`

- **What it does**: stops the running front-end process by finding it and sending SIGTERM.
- **Command**: `pkill -SIGTERM -f "../bin/${FRONT_END_BINARY}"`
- **Gotcha**: `pkill -f` matches the command line; if you have multiple similar processes, it may stop more than you intend.

---

## Notes / FAQ

> **Why is my Makefile different from the course's?**

The course provided `Makefile` is placed under `docs/assets/` ([original Makefile](assets/Makefile)).

They both run `go build` to produce a Linux binary, but they do it in different environments:

- **Makefile `build_broker` (course style)**
  - Runs on your host machine.
  - Builds a Linux binary by setting `GOOS=linux` (so it can run in a Linux container).
  - Produces the binary in your repo directory (whatever `${BROKER_BINARY}` points to).
  - Then Compose starts containers, often reusing that already-built local binary.

- **Dockerfile build stage (`broker-service/broker-service.dockerfile`)**
  - Runs inside Docker during `docker build`.
  - Uses the `golang:*` builder image to compile the binary inside the image filesystem at `/app/brokerApp`.
  - You won’t see the binary on your host unless you copy it out.

**Build on host (course style)** tends to be faster iteration and you can “see” binaries locally, but you must ensure you’re producing a Linux-compatible binary and your host tooling is consistent.

**Build in Docker (this repo’s broker approach)** is more reproducible and doesn’t require Go installed on the host, but builds can be slower and binaries live inside the image.

> **Why keep the (frontend) binary in `bin/` and not in `front-end/`?**

It’s mostly an organization choice:

- Keeping binaries in a single top-level `bin/` makes it easy to find and clean them (`rm -rf bin/`), and keeps source folders free of compiled artifacts.
- The important part is *how you run the binary*: because templates are loaded from disk via relative paths, you typically run it from `front-end/` (as the Makefile does) unless you switch to embedded templates (`embed`) or a configurable template directory.

> **How to delete `brokerApp` binary inside Docker image?**

- Deleting a container (`docker compose down`) removes the running container, but the image (and its `/app/brokerApp`) still exists on your machine.
- To remove the binary too, you need to remove the image (or prune images), e.g. `docker compose down --rmi local` (or `docker image prune`).

> **When do I need to delete the `brokerApp` binary inside Docker image?**

You usually don’t need to “delete the binary” manually.

- Inside the newly built image: yes—each build produces a fresh `/app/brokerApp` for that image.
- On your machine overall: old images (and their old `/app/brokerApp`) can still remain on disk until you remove/prune them. They aren’t “in-place overwritten”; Docker keeps old layers/images around for caching and rollback.

Only delete / prune when:
- Want to reclaim disk space or force a totally clean rebuild
  - `docker compose down --rmi local`
  - `docker image prune`
  - `docker system prune` - Broader cleanup, but may remove more than this project's scope