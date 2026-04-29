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

### 3) RabbitMQ DNS: use the Service DNS name

If your broker can’t resolve the RabbitMQ hostname, you’ll see errors like:

`dial tcp: lookup rabbitmq on ...:53: no such host`

This is almost always because your broker dial string doesn’t match the **Kubernetes Service name**.
Inside the cluster, the easiest hostname to use is the Service’s `metadata.name` (optionally with namespace).

In your project:

- RabbitMQ Service name (in `project/k8s/rabbitmq.yml`): `rabbitmq-service`
- So the broker should dial: `amqp://guest:guest@rabbitmq-service`

Once the hostname matches, the broker stops crashing/restarting and connects to RabbitMQ successfully.