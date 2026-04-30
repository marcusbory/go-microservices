# Kubernetes: Labels, Selectors, and Services

This note focuses on one of the most confusing (but crucial) parts of Kubernetes networking:
how a **Service** knows which **Pods** to route traffic to.

---

## 1) Labels vs. object names

Kubernetes has *two different kinds of identifiers* that often look similar at first:

- `metadata.name`
  - The name of a Kubernetes object (Deployment, Service, etc.).
  - Example in your manifest: `metadata.name: mongo-service`.
  - This does **not** automatically get copied to Pods.

- Labels (set under `metadata.labels` / `spec.template.metadata.labels`)
  - Arbitrary `key: value` tags applied to objects—most importantly Pods.
  - Example in your manifest: `app: mongo`.

When you configure a Service, the Service selector matches **Pod labels**, not `metadata.name`.

---

## 2) What `spec.selector` on a Service actually does

For a Service, this section:

```yaml
spec:
  selector:
    <label-key>: <label-value>
```

means:

> “Find all Pods whose labels match these key/value pairs, and route traffic to them.”

So the Service is basically: **selector (labels) -> matching Pods -> endpoints**.

---

## 3) How the Deployment ties into the Service (labels propagation)

The important part is that a **Deployment creates Pods**, and those Pods inherit labels from the Deployment template:

```yaml
spec:
  template:
    metadata:
      labels:
        app: mongo
```

So, in practice:

1. Deployment starts Pods
2. Pods get `app: mongo` labels
3. Service selector `app: mongo` matches those Pods
4. Traffic to the Service gets routed to the matching Pods

---

## 4) Applying this to your `project/k8s/mongo.yml`

In your current file:

- Deployment template labels include:
  - `app: mongo`
- Service selector includes:
  - `app: mongo`

That means the Service will route traffic to the Mongo Pods created by that Deployment.

### Common confusion: “name: mongo” vs “app: mongo”

The key thing is: the selector key must match the **label key actually present on Pods**.

- If you wrote `selector: { name: mongo }` in the Service, you would also need the Pods to have `name: mongo` as a label.
- If your Pods only have `app: mongo`, then selecting by `name: mongo` matches nothing.

---

## 5) For `mongo-deployment`, are the `labels` cosmetic for now?

> In the file, `metadata.labels` exists in two different places, so they do different things.

`metadata.labels` - Deployment Object
- Live on the deployment resource itself
- Mostly for organisation / filtering / tooling (e.g. `kubectl get deploy -l app=mongo`). Don't directly affect service routing.
- Currently, this is *cosmetic*.

`spec.template.metadata.labels` - Pod Template
- Matters for actual runtime networking
- Kubernetes will copy these into the Pods' labels
- Service selector uses **Pod labels**, so it only works with `app: mongo` labels on Pods.

---

## Quick sanity checks (optional but helpful)

These commands are the fastest way to confirm label/selector wiring in a real cluster:

- View Pod labels:
  - `kubectl get pods --show-labels`
- View Service selector:
  - `kubectl get svc mongo-service -o yaml`

If your Service selector matches the Pod labels, routing will work.

---

## FAQ / Clarifications about K8s

### 1) Kubernetes needs an image, not a Dockerfile

In a Kubernetes manifest you can only set:

- `image: <registry>/<repo>:<tag>`

You cannot point Kubernetes at a local `Dockerfile` (unlike Docker Compose). The workflow is:

1. Build your image (using your `Dockerfile`)
```sh
docker login
docker build -f <path/to/dockerfile> -t DOCKERHUB_USER/broker:latest . # context
docker push DOCKERHUB_USER/broker:latest
```
2. Make it available to the cluster (typically by pushing to a registry)
3. Reference the pushed image in the Pod spec (`containers[].image`)

### 2) `containerPort` is not a host port mapping

`containerPort` documents the port your container listens on. It does **not** publish that port to the host.

The actual routing happens via a Kubernetes `Service`:

- `spec.ports[].port`: the port clients use *inside the cluster*
- `spec.ports[].targetPort`: the port on the Pod/container to send traffic to

So if Docker Compose used something like:

- `"8080:80"` (host `8080` → container `80`)

Then in Kubernetes you usually want the Service to target the **container’s listening port** (`80`), not the Docker Compose host port (`8080`).

For example, in your broker service:

- `broker-service/cmd/api/main.go` sets `WEB_PORT = "80"` → the broker listens on `:80`
- therefore your Kubernetes Deployment/Service should align to `containerPort: 80` and `targetPort: 80`

Extra Info on Ports (from perspective of a Service):
- `nodePort` [Optional, else uses 30000-32767] - The port where external traffic will come in on (only if is `NodePort` Service).
- `port` - Port of this **Service**. Internal traffic will use this port instead of nodePort.
- `targetPort` - Target Port of **Pod** to forward traffic to.

### 3) RabbitMQ DNS: use the Service DNS name

If your broker can’t resolve the RabbitMQ hostname, you’ll see errors like:

`dial tcp: lookup rabbitmq on ...:53: no such host`

This is almost always because your broker dial string doesn’t match the **Kubernetes Service name**.
Inside the cluster, the easiest hostname to use is the Service’s `metadata.name` (optionally with namespace).

In your project:

- RabbitMQ Service name (in `project/k8s/rabbitmq.yml`): `rabbitmq-service`
- So the broker should dial: `amqp://guest:guest@rabbitmq-service`

Once the hostname matches, the broker stops crashing/restarting and connects to RabbitMQ successfully.

Other examples include:
- `listener-service/main.go`: `amqp://guest:guest@rabbitmq-service` as per above (listener processes MQ)
- `logger-service/cmd/api/main.go`: `MONGO_URI` needs to be updated to `mongo-service` endpoint to match K8s DNS

### 4) Connecting a K8s Pod to a *local* docker-compose Postgres

If Postgres is running via Docker Compose on your machine and published on `localhost:5432`, a Kubernetes Pod
won't be able to reach it via the Docker Compose service name (e.g. `postgres-service`) unless Kubernetes
also has a Service with that name.

This repo includes `project/k8s/postgres-external.yml`, which creates a Kubernetes Service named
`postgres-service` that resolves to `host.minikube.internal` (for minikube on macOS).

Under the hood, this is an `ExternalName` Service, which is best thought of as a **DNS alias** inside the
cluster:
- It provides a stable in-cluster hostname (`postgres-service`) for clients to use
- It does **not** create/load-balance to Pod endpoints (there are no Pods behind it)
- Clients connect **directly** to the external host after DNS resolution

Apply it:

```sh
kubectl apply -f project/k8s/postgres-external.yml
```

Then your Pod DSN can remain:

`host=postgres-service port=5432 ...`

Notes:
- If you are using **Docker Desktop Kubernetes**, you likely want `host.docker.internal` instead.
- If you are using **kind**, you typically need to use the host gateway IP or run Postgres inside the cluster.

### 5) Ingress + Ingress Controller (NGINX): exposing Services publicly

An **Ingress** is a Kubernetes resource that defines HTTP/HTTPS routing rules (hostnames + paths) for traffic
coming **into** the cluster.

An **Ingress Controller** (e.g. NGINX Ingress) is the component that actually enforces those rules:
- It runs inside the cluster and watches `Ingress` resources
- It accepts external traffic (usually via a `LoadBalancer` or a `NodePort` Service in front of the controller)
- It routes requests to the correct Kubernetes `Service` based on `spec.rules[].host` and `spec.rules[].http.paths[]`

In this repo, `project/ingress.yml` maps public hostnames to internal Services:
- `host: front-end.info` → `service.name: front-end-service` on `port: 8081` (from `project/k8s/front-end.yml`)
- `host: broker-service.info` → `service.name: broker-service` on `port: 80` (from `project/k8s/broker.yml`)

#### 5.1 Why `broker-service.info` exists (and how it relates to `BROKER_URL`)

There are two different “names” involved:
- **Service DNS name (internal)**: `http://broker-service`
  - Resolvable only inside the cluster (CoreDNS)
  - Intended for pod-to-pod traffic
- **Ingress hostname (public)**: `http://broker-service.info`
  - A public DNS name that should resolve to the Ingress Controller’s external IP
  - Intended for traffic coming from outside the cluster (e.g. your browser)

The frontend sets `BROKER_URL` to `http://broker-service.info` so browser-based JavaScript can reach the broker
through the Ingress. If you set `BROKER_URL` to `http://broker-service`, your browser won’t be able to resolve
that hostname (it only exists inside Kubernetes).

#### 5.2 Security note: this makes the broker publicly reachable

If your Ingress has a `broker-service.info` rule, then the `broker-service` is effectively exposed as a **public API**
endpoint (anyone who can reach the Ingress can send requests to it).

Whether that’s acceptable depends on your broker endpoints:
- If they are authenticated, rate-limited, and intended for public use, exposing them can be fine.
- If they are “internal-only” endpoints, exposing them is risky.

Safer patterns:
- **Expose only the frontend publicly**, and keep broker internal (remove the `broker-service.info` host rule).
  - If the frontend still needs to trigger broker actions, do it server-side (frontend backend → `http://broker-service`)
    rather than browser JS → public broker hostname.
- If you must expose the broker, add controls at the broker and/or Ingress (TLS, auth, rate limiting, WAF rules).

#### 5.3 Why does `front-end.yml` still need to use `http://broker-service.info`?

Even though the frontend is *hosted* in Kubernetes, the `fetch(...)` calls in your HTML template run in the
**user’s browser**, not inside the frontend Pod.

So:
- Browser → `front-end.info` (Ingress) → `front-end-service` works (page delivery)
- Browser → `broker-service.info` (Ingress) → `broker-service` works (API call)
- Browser → `broker-service` does **not** work, because `broker-service` is cluster-only DNS

If you want to avoid making the broker publicly reachable, change the flow so the browser calls only the
frontend, and the frontend (server-side, from inside the cluster) calls `http://broker-service`.

### 6) What is `minikube tunnel` for?

In **minikube**, `Service` objects of type `LoadBalancer` don’t get a real cloud load balancer automatically, so
their `EXTERNAL-IP` often stays `<pending>`.

Running:

```sh
minikube tunnel
```

creates a network route from your machine to the minikube cluster and assigns a reachable external IP to those
`LoadBalancer` Services. Keep it running while you want to access the service externally (it may prompt for
sudo/admin privileges).