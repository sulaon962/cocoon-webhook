# cocoon-webhook

Kubernetes admission webhook for sticky scheduling of VM-backed pods in [cocoonstack](https://github.com/cocoonstack) clusters.

## Overview

- **Mutating** (`POST /mutate`) -- on Pod CREATE, derives a stable VM name from the Deployment/ReplicaSet owner chain, looks up the previously assigned node in the `cocoon-vm-affinity` ConfigMap, and patches `spec.nodeName` so the pod returns to the same worker. Writes the `cocoon.cis/vm-name` annotation when missing.
- **Validating** (`POST /validate`) -- on Deployment/StatefulSet UPDATE, blocks scale-down for cocoon-type workloads. Agents are stateful VMs; reducing replicas would destroy state. Use the Hibernation CRD to suspend individual agents instead.
- **Health check** -- served on `GET /healthz`

Recommended for multi-worker cocoon pools where restart affinity matters and Deployments that recreate VM-backed pods while expecting state continuity. Often unnecessary for single-worker labs, setups that pin workloads explicitly with `nodeName`, or setups that rely only on CocoonSet.

## Installation

### Download

Download a pre-built binary from [GitHub Releases](https://github.com/cocoonstack/cocoon-webhook/releases):

```bash
# Linux (amd64)
curl -fSL -o cocoon-webhook \
  "https://github.com/cocoonstack/cocoon-webhook/releases/latest/download/cocoon-webhook-linux-amd64"
chmod +x cocoon-webhook

# macOS (amd64)
curl -fSL -o cocoon-webhook \
  "https://github.com/cocoonstack/cocoon-webhook/releases/latest/download/cocoon-webhook-darwin-amd64"
chmod +x cocoon-webhook
```

### Build from source

```bash
git clone https://github.com/cocoonstack/cocoon-webhook.git
cd cocoon-webhook
make build        # produces ./cocoon-webhook
```

## Configuration

The binary expects TLS certificates and listens on `:8443`.

| Variable | Default | Description |
|---|---|---|
| `TLS_CERT` | `/etc/cocoon/webhook/certs/tls.crt` | Path to TLS certificate |
| `TLS_KEY` | `/etc/cocoon/webhook/certs/tls.key` | Path to TLS private key |

Package it behind a standard Kubernetes Deployment, Service, and MutatingWebhookConfiguration, or run it on a control-plane host if that fits your environment.

## Development

```bash
make build        # build binary
make test         # vet + race-detected tests with coverage
make lint         # golangci-lint for linux and darwin
make fmt          # format code
make help         # show all targets
```

## Related Projects

| Project | Role |
|---|---|
| [vk-cocoon](https://github.com/cocoonstack/vk-cocoon) | Virtual kubelet provider managing VM lifecycle |
| [cocoon-operator](https://github.com/cocoonstack/cocoon-operator) | CocoonSet and Hibernation CRDs |
| [epoch](https://github.com/cocoonstack/epoch) | Remote snapshot storage |
| [glance](https://github.com/cocoonstack/glance) | Web dashboard |

## License

[MIT](LICENSE)
