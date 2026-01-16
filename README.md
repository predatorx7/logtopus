# Logtopus 🐙

<img src="https://raw.githubusercontent.com/predatorx7/logtopus/refs/heads/main/public/logtopus.png" width="140" height="150" align="center" />

High-performance, modular log ingestion and query system.

## Features
- **High Ingestion Throughput**: Non-blocking in-memory buffering.
- **Multiple Subscribers**:
  - **ClickHouse**: High-volume, analytical storage (optional).
  - **File**: Local file storage.
- **Query Service**: Separate HTTP service to query logs from ClickHouse or File.
- **OpenAPI Docs**: Integrated API documentation.
- **Authentication**: HMAC-SHA256 based API Key authentication.

## Services & Commands
The project consists of several components located in the `cmd/` directory:

| Service | Source | Description |
| :--- | :--- | :--- |
| **Ingestor** | `cmd/ingestor` | The main HTTP service for accepting logs. |
| **Query Service** | `cmd/query-service` | HTTP service for querying logs from the backend. |
| **API Key Gen** | `cmd/apikey-gen` | CLI tool to generate HMAC-SHA256 API keys. |
| **Setup DB** | `cmd/setup-db` | Tool to initialize ClickHouse tables. |

## Quick Start

You can run Logtopus using **Docker Compose** (recommended for ease) or **Locally** (for development).

### Option A: Docker Compose

#### 1. File-Only Mode (Lightweight)
Use this mode for local testing without ClickHouse.
```bash
docker-compose -f docker-compose.file.yml up --build
```
- **Ingestion Service**: `http://localhost:8080`
- **Query Service**: `http://localhost:8081`

#### 2. ClickHouse Mode (Full Power)
Use this mode for production-grade setup with deep analytical capabilities.
```bash
# Start Services
docker-compose -f docker-compose.clickhouse.yml up -d --build

# Initialize Database (Run once)
make setup-db
```
- **Ingestion Service**: `http://localhost:8080`
- **Query Service**: `http://localhost:8081`

### Option B: Local Development

Prerequisites: Go 1.25+

#### 1. Build Binaries
```bash
make build
# Binaries will be in build/bin/
```

#### 2. Run Ingestion Service
**File Mode (Default):**
```bash
export ENABLE_FILE_LOGGING=true
export FILE_LOG_DIR=./logs
./build/bin/logtopus
```

**ClickHouse Mode:**
Ensure ClickHouse is running locally (e.g., via `docker run`).
```bash
export ENABLE_CLICKHOUSE=true
export CLICKHOUSE_DSN="clickhouse://default:password@localhost:9000/logtopus"
./build/bin/logtopus
```

#### 3. Run Query Service
**File Mode:**
```bash
export SEARCH_DIR=./logs
export QUERY_PORT=8081
./build/bin/query-service
```

**ClickHouse Mode:**
```bash
export CLICKHOUSE_DSN="clickhouse://default:password@localhost:9000/logtopus"
export QUERY_PORT=8081
./build/bin/query-service
```

## Usage

### Ingest Logs
Requires valid API Key (see `AUTH_SECRET` in `.env` or set environment variable).

1. **Generate Key**:
   ```bash
   go run ./cmd/apikey-gen -client my-client -secret change-me-in-prod-secret-key-123
   # Output example: my-client.a1b2c3d4...
   ```

2. **Send Logs**:
   ```bash
   curl -X POST http://localhost:8080/v1/logs \
     -H "X-API-Key: <YOUR_KEY>" \
     -d '[{"message":"hello", "level":"INFO"}]'
   ```

### Query Logs

**ClickHouse (Default):**
```bash
curl "http://localhost:8081/v1/logs?limit=5"
```

**File Store:**
```bash
curl "http://localhost:8081/v1/logs?subscriber_type=file&limit=5"
```

### Documentation
- Ingest API: `http://localhost:8080/openapi.yaml`
- Query API: `http://localhost:8081/openapi.yaml`

## Development Commands
The `Makefile` provides several helpers:

- `make build`: Build all binaries.
- `make fmt`: Format all Go code.
- `make test`: Run unit tests.
- `make setup-db`: Initialize ClickHouse database (requires env vars).
- `make docker-up`: Start full stack via Docker.
- `make docker-logs`: Tail Docker logs.
