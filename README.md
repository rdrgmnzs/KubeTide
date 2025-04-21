# KubeTide 🌊

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/kubetide/kubetide)](https://github.com/rdrgmnzs/kubetide/releases)
[![License](https://img.shields.io/github/license/kubetide/kubetide)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/rdrgmnzs/kubetide)](https://goreportcard.com/report/github.com/rdrgmnzs/kubetide)

Scheduled rollout restarts for Kubernetes workloads, controlled by annotations. Like the tides, your deployments refresh on a predictable schedule.

KubeTide is a Kubernetes controller that performs scheduled rollout restarts on Deployments, StatefulSets, and DaemonSets using simple cron expressions. Just annotate your resources, and KubeTide handles the rest.

## Key Features

- ⏰ Cron-based scheduling with timezone support
- 🔄 Automated rollout restarts for Deployments, StatefulSets, and DaemonSets
- 🏷️ Simple annotation-driven configuration
- 🐳 Lightweight controller with minimal resource footprint
- 📊 Prometheus metrics and health endpoints
- 🔒 Built with security in mind (non-root, minimal permissions)

## Installation

### Using Helm

```bash
# Add the Helm repository
helm pull oci://ghcr.io/rdrgmnzs/kubetide/charts/kubetide --version 0.1.0

# Install the chart
helm install kubetide kubetide-0.1.0.tgz --namespace kubetide --create-namespace
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/rdrgmnzs/kubetide.git
cd kubetide

# Install CRDs and resources
kubectl apply -f deploy/manifests
```

## Usage

Add the following annotations to your resources to enable scheduled restarts:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  annotations:
    kubetide.io/schedule: "0 2 * * *"   # Run at 2:00 AM daily
    kubetide.io/timezone: "UTC"         # Optional: specify a timezone
spec:
  # ... your deployment spec
```

### Annotations

| Annotation | Description | Example | Required |
|------------|-------------|---------|----------|
| `kubetide.io/schedule` | Cron expression for the restart schedule | `"0 2 * * *"` | Yes |
| `kubetide.io/timezone` | Timezone for the schedule | `"America/New_York"` | No (defaults to controller timezone) |

## Configuration

The following table lists the configurable parameters for the KubeTide Helm chart:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/rdrgmnzs/kubetide` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `timezone` | Default timezone for controller | `UTC` |
| `leaderElection.enabled` | Enable leader election | `true` |
| `resources.limits.cpu` | CPU limits | `200m` |
| `resources.limits.memory` | Memory limits | `256Mi` |
| `resources.requests.cpu` | CPU requests | `100m` |
| `resources.requests.memory` | Memory requests | `128Mi` |
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.name` | Service account name | `""` (defaults to release name) |
| `metrics.enabled` | Enable Prometheus metrics | `true` |
| `metrics.serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | `false` |

## Examples

### Daily restart at midnight (UTC)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  annotations:
    kubetide.io/schedule: "0 0 * * *"
spec:
  # ... deployment spec
```

### Weekly restart on Sundays at 1:30 AM in New York timezone

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  annotations:
    kubetide.io/schedule: "30 1 * * 0"
    kubetide.io/timezone: "America/New_York"
spec:
  # ... deployment spec
```

### Monthly restart on the 1st day at 4:15 AM

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: database
  annotations:
    kubetide.io/schedule: "15 4 1 * *"
spec:
  # ... statefulset spec
```

## Why KubeTide?

Regular application restarts can help prevent memory leaks, clear caches, and ensure your applications stay healthy. KubeTide makes scheduled restarts effortless and reliable.

## Monitoring

KubeTide exposes the following Prometheus metrics:

- `kubetide_restart_total`: Total number of restarts by resource type
- `kubetide_error_total`: Total number of errors encountered
- `kubetide_schedule_drift_seconds`: Time drift between scheduled and actual restart time
- `kubetide_resource_processing_latency_seconds`: Time taken to process a resource operation

## Development

### Prerequisites

- Go 1.24+
- Docker
- kubectl
- Helm

### Building the Controller

```bash
# Build the controller binary
make build

# Run tests
make test

# Build Docker image
make docker-build
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/FeatureDescription`)
3. Commit your Changes (`git commit -m 'Add some Feature Description'`)
4. Push to the Branch (`git push origin feature/FeatureDescription`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.