# cocoon-webhook

Kubernetes admission webhook for the [cocoonstack](https://github.com/cocoonstack) VM platform.

## Overview

cocoon-webhook hosts three admission endpoints:

| Endpoint | Type | Resources | What it does |
|---|---|---|---|
| `POST /mutate` | Mutating | Pod CREATE | Reserves a stable VM name + a sticky cocoon node for cocoon-tolerated pods that are not owned by a CocoonSet. |
| `POST /validate` | Validating | Deployment / StatefulSet UPDATE | Rejects scale-down on cocoon-tolerated workloads (agents are stateful VMs — use `CocoonHibernation` to suspend instead). |
| `POST /validate-cocoonset` | Validating | CocoonSet CREATE / UPDATE | Catches the cross-field business rules the CRD's OpenAPI schema cannot express (image required, toolbox name uniqueness, static-mode prerequisites). |
| `GET /healthz` | Liveness | — | Always 200 once the binary is running. |
| `GET /readyz` | Readiness | — | 200 once dependencies needed to serve admission traffic are reachable. |
| `GET /metrics` | Prometheus | — | Plain HTTP on `:9090`, separate from the admission TLS port. |

## Sticky scheduling model

The mutator answers two questions for every admitted pod:

1. **Which VM name?** Derived deterministically from the deployment slot via `meta.VMNameForDeployment` (or the bare pod name via `meta.VMNameForPod`).
2. **Which cocoon node?** The previously pinned node if there is one for that slot, otherwise picked from the pool by the least-used policy.

State lives in one **per-pool** ConfigMap (`cocoon-affinity-<pool>`) in the `cocoon-system` namespace. Each entry is keyed by `<namespace>/<deployment>/<slot>` and the value is a JSON-encoded `Reservation`. Multiple webhook replicas race safely via `RetryOnConflict`.

A background **Reaper** sweeps every per-pool ConfigMap on a 5-minute interval and releases reservations whose backing pod has been gone for more than 30 minutes.

Pool selection looks at the pod in this order:
1. `spec.nodeSelector["cocoonstack.io/pool"]`
2. `metadata.labels["cocoonstack.io/pool"]`
3. `metadata.annotations["cocoonstack.io/pool"]`
4. `default`

Cocoon nodes themselves opt into a pool by setting the `cocoonstack.io/pool=<name>` label.

## CocoonSet validation rules

The CRD ships with `+kubebuilder` enum / required / default markers, but the webhook adds the cross-field business rules:

- `spec.agent.image` must be set
- `spec.agent.replicas >= 0`
- `spec.agent.mode ∈ {clone, run}`
- `spec.agent.os ∈ {linux, windows}`
- `spec.toolboxes[*].name` unique and matches RFC 1123
- `spec.toolboxes[*]` static mode requires both `staticIP` and `staticVMID`
- `spec.toolboxes[*]` non-static modes require `image`
- `spec.snapshotPolicy ∈ {always, main-only, never}`

## Configuration

| Variable | Default | Description |
|---|---|---|
| `KUBECONFIG` | unset | Path to kubeconfig when running outside the cluster (in-cluster config used otherwise) |
| `WEBHOOK_LOG_LEVEL` | `info` | `projecteru2/core/log` level |
| `TLS_CERT` | `/etc/cocoon/webhook/certs/tls.crt` | TLS server certificate |
| `TLS_KEY` | `/etc/cocoon/webhook/certs/tls.key` | TLS server private key |
| `LISTEN_ADDR` | `:8443` | Admission listener (HTTPS) |
| `METRICS_ADDR` | `:9090` | Prometheus listener (HTTP) |

## Installation

The supported install path is `kubectl apply -k`:

```bash
kubectl apply -k github.com/cocoonstack/cocoon-webhook/config/default?ref=main
```

This installs:
- `cocoon-system` namespace
- `ServiceAccount` + `ClusterRole` (read pods/nodes; full configmap CRUD in `cocoon-system`)
- cert-manager `Issuer` + `Certificate` (`cocoon-webhook-tls`) — **cert-manager must already be installed in the cluster**
- `Deployment` (2 replicas) + `Service` (port 443 → 8443, port 9090 → 9090)
- `MutatingWebhookConfiguration` for Pod CREATE
- `ValidatingWebhookConfiguration` for Deployment/StatefulSet UPDATE and CocoonSet CREATE/UPDATE

To override the image tag or replica count, build a kustomize overlay that imports `config/default` as a base.

## Development

```bash
make all            # full pipeline: deps + fmt + lint + test + build
make build          # build cocoon-webhook binary
make test           # vet + race-detected tests
make lint           # golangci-lint on linux + darwin
make fmt            # gofumpt + goimports
make help           # show all targets
```

The Makefile detects Go workspace mode (`go env GOWORK`) and skips `go mod tidy` when active so cross-module references resolve through `go.work` without forcing a release of cocoon-common.

## Related projects

| Project | Role |
|---|---|
| [cocoon-common](https://github.com/cocoonstack/cocoon-common) | CRD types, annotation contract, shared helpers |
| [cocoon-operator](https://github.com/cocoonstack/cocoon-operator) | CocoonSet and CocoonHibernation reconcilers |
| [epoch](https://github.com/cocoonstack/epoch) | Snapshot registry and storage backend |
| [vk-cocoon](https://github.com/cocoonstack/vk-cocoon) | Virtual kubelet provider managing VM lifecycle |

## License

[MIT](LICENSE)
