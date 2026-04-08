# 🪐 Orbit

> Lightweight, single-node container orchestration and automation — powered by your existing Docker Engine.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/requires-Docker%20Engine-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/engine/install/)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20arm-lightgrey)](#)

---

## Why Orbit?

Managing containers today is fragmented:

| Tool                      | Limitation                                                      |
| ------------------------- | --------------------------------------------------------------- |
| Docker Compose            | Long-running services only — no cron or event support           |
| Cron + Docker CLI         | Manual, error-prone, and lacks visibility                       |
| Kubernetes                | Powerful, but too heavy for single-node or edge setups          |
| Airflow / Nomad / Rundeck | Complex, require databases, or don't manage containers natively |

**Orbit combines all of these capabilities into a single, lightweight binary.**

---

## Features

### 🖥 Services

- Long-running containers: web apps, databases, workers, dashboards
- Auto-start on boot, restart on failure, health checks

### ⏱ Jobs

- Short-lived containers on a cron schedule
- Examples: backups, cleanup tasks, report generation

### ⚡ Triggers

Run containers in response to events:

- **HTTP webhook**
- **MQTT message**
- **File system change**
- **Custom internal events**

### 🛠 Unified Management

- CLI and REST API for automation and monitoring
- Logs, status, and execution history in one place
- Declarative YAML configuration for reproducibility

---

## Architecture

Orbit acts as the brain — Docker is the executor.

```
User
  └─> CLI / REST API
        └─> Orbit Daemon
              └─> Docker Engine
                    └─> Containers / Jobs
```

| Component              | Role                                                                            |
| ---------------------- | ------------------------------------------------------------------------------- |
| **CLI**                | Human-friendly commands: `orbit up`, `orbit down`, `orbit logs`, etc.           |
| **REST API**           | Integrations and automation                                                     |
| **Orchestrator**       | Core brain: reconciles desired vs. actual state, manages services/jobs/triggers |
| **Scheduler**          | Handles cron-style job execution                                                |
| **Event Router**       | Listens for events and routes them to containers/jobs                           |
| **Docker Adapter**     | Abstracts Docker API calls                                                      |
| **State Store**        | Stores metadata, trigger/job registrations, execution history                   |
| **Log & Event Stream** | Collects logs, job runs, trigger executions, and failures                       |

---

## How It Works

### Startup

1. Load configuration and saved state
2. Connect to Docker Engine
3. Reconcile desired state with actual containers
4. Start services, scheduler, and event listeners

### Service Lifecycle

1. Define service in YAML
2. Orbit creates the container and monitors health
3. Restarts on failure based on configured policy
4. Updates the container if configuration changes

### Job Execution

1. Scheduler determines it's time to run
2. Orbit launches a one-off container
3. Captures logs and exit code
4. Stores the execution result

### Trigger Execution

1. Event occurs (webhook, MQTT, file change, etc.)
2. Event router matches the trigger
3. Orbit runs the target container or job
4. Stores the execution result

---

## Configuration

Orbit uses a single declarative YAML file:

```yaml
services:
    web:
        image: nginx:latest
        ports:
            - "8080:80"
        restart: always

jobs:
    nightly-backup:
        image: backup:latest
        schedule: "0 2 * * *"

triggers:
    file-upload:
        type: file
        path: /data/incoming
        target: process-upload
```

---

## State Management

Orbit is fully self-contained — no external databases required.

| Layer                 | Purpose                  |
| --------------------- | ------------------------ |
| **SQLite**            | Durable metadata storage |
| **JSON / YAML files** | Configuration            |
| **Docker Engine**     | Runtime container state  |

---

## Getting Started

### Prerequisites

- [Docker Engine](https://docs.docker.com/engine/install/) installed and running

### Install

```bash
# Download the Orbit binary
curl -fsSL https://get.orbit.sh | sh
```

### Usage

**1. Define your workloads** in `orbit.yml` (see [Configuration](#configuration) above).

**2. Start Orbit:**

```bash
orbit up
```

**3. View logs:**

```bash
orbit logs
```

**4. Check status:**

```bash
orbit status
```

**5. Stop all workloads:**

```bash
orbit down
```

---

## Key Advantages

- 🪶 **Lightweight** — single binary, runs on Raspberry Pi, old laptops, or cloud VMs
- ⚡ **Event-driven automation** built in from day one
- 📄 **Declarative configuration** for predictable, reproducible operations
- 🔧 **Unified CLI and API** for all workload types
- 🐳 **Uses Docker Engine** — no runtime replacement needed

---

## Roadmap

- [ ] Web UI for real-time monitoring
- [ ] Additional triggers: Kafka, cloud events, IoT protocols
- [ ] Optional multi-node orchestration

---

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

---

## License

[MIT](LICENSE) © Orbit Contributors
