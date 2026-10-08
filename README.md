# go-evsched

`go-evsched` is a high-performance, event-driven job scheduling built in Go. It supports both in-memory job and persistent job backed by PostgreSQL.

---

## 🌟 Key Features
- **Persistent Job Scheduling:** Durable job scheduler backed by PostgreSQL.
- **In-Memory (Static) Job Scheduling:** Supports lightweight, transient tasks running in memory.
- **Transactional Outbox Pattern:** Ensures reliable, at-least-once job publishing.
- **Event-Driven Integration (RabbitMQ):** Event publishing to RabbitMQ.
- **Observability:** OpenTelemetry tracing and logging; Prometheus metrics (`/metrics`).

---

## 🛠 Tech Stack

- **Language:** Go 1.26+
- **HTTP Router:** go-chi
- **Database:** PostgreSQL with `pgx/v5`
- **Messaging:** RabbitMQ
- **Metrics:** Prometheus Client Go
- **Tracing:** OpenTelemetry SDK & Exporters

---

## 📋 Prerequisites

Ensure you have the following installed on your development machine:
- Go 1.26+ 
- Docker Compose
- Make (optional)

---

## 🚀 Getting Started

### 1. Clone the Repository
```bash
git clone https://github.com/l20forwork/go-evsched.git
cd go-evsched
```

### 2. Configure Environment
Review and adjust `default.yaml` in the root directory if necessary. You can override configuration values using environment variables (e.g., `EVSCHED_DB_HOST`, `EVSCHED_RABBITMQ_HOST`) or override default config file with command flag `--config`

### 3. Start Infrastructure (PostgreSQL & RabbitMQ)
Before running the application, start PostgreSQL and RabbitMQ
```bash
docker compose -f infra-docker-compose.yml up -d
```

### 4. Build the Application
Using the provided build script or Makefile:
```bash
  # Using Makefile
make build

# Or directly via build script
./build.sh
```
The compiled binary will be placed at `bin/linux-amd64/evsched` (or appropriate OS/arch).

### 5. Run the Server
```bash
go run ./cmd/server
# Or using the built binary:
./bin/linux-amd64/evsched
```
*(Note: If running database migrations on startup, pass `--init-db` as command argument)*
```bash
go run ./cmd/server --init-db
```

---

## 🐳 Deployment with Docker

### Building the Docker Image

```bash
  docker build -f Dockerfile.build -t go-evsched:latest .
```

### Running the Docker Container

```bash
docker run -d \
  --name evsched \
  -p 9020:9020 \
  go-evsched:latest
```

## 🔍 Health & Metrics Endpoints

- **Health Check:** `GET /health`
- **Prometheus Metrics:** `GET /metrics` (Exposes runtime, HTTP request metrics, and job execution metrics)
