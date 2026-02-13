# Go Dispatcher

Message-driven analytics and dashboard backend that consumes requests from Redis Streams, computes widget/page data (MongoDB, Cassandra, MinIO, HTTP APIs), and dispatches results via Redis Streams, Redis buffer (stream_tech API), or HTTP POST to a live/SSE service.

Specification: see `SECRET.md` (not committed; add your copy locally).

## Architecture

- **Redis consumer** (main worker): consumes from `APP_DISPATCHER_INPUT_STREAM` with consumer groups, runs controller + strategies, dispatches to output streams / dead-letter / buffer / live service.
- **HTTP API**: optional process serving `/health` and `/ready` on `0.0.0.0:8080` (or `PORT`).

Both processes use the same configuration (environment variables).

## Build

```bash
go build -o bin/consumer ./cmd/consumer
go build -o bin/server  ./cmd/server
```

## Run

### Required environment variables (consumer)

- `APP_DISPATCHER_INPUT_STREAM` – input stream name (e.g. `go_dispatcher:requests`)
- `APP_DISPATCHER_CONSUMER_GROUP` – consumer group (e.g. `go_dispatcher`)
- `APP_DISPATCHER_CONSUMER_NAME` – unique consumer name (e.g. hostname)
- `APP_DISPATCHER_OUTPUT_STREAM_PREFIX` – output stream prefix (e.g. `go_dispatcher:out:`)
- `APP_DISPATCHER_REJECTED_STREAM` – dead-letter stream (e.g. `go_dispatcher:deadletters`)

Redis host/port default to `APP_BACKEND_REDIS_SERVER` / `APP_BACKEND_REDIS_PORT` if `APP_REDIS_HOST` / `APP_REDIS_PORT` are unset.

### Consumer

```bash
export APP_DISPATCHER_INPUT_STREAM=go_dispatcher:requests
export APP_DISPATCHER_CONSUMER_GROUP=go_dispatcher
export APP_DISPATCHER_CONSUMER_NAME=consumer1
export APP_DISPATCHER_OUTPUT_STREAM_PREFIX=go_dispatcher:out:
export APP_DISPATCHER_REJECTED_STREAM=go_dispatcher:deadletters
./bin/consumer
```

### HTTP server

```bash
export PORT=8080
# Same env as consumer (validation requires stream keys even when only running server)
./bin/server
```

Then: `curl http://localhost:8080/health` and `curl http://localhost:8080/ready`.

## Accepted events and strategies

Accepted `eventName` values and event-to-page enrichment are defined in `internal/models/events.go`. Strategies are registered in `internal/strategies/stub.go`; the current implementation uses stub strategies that return placeholder payloads. Replace or extend them with real query builders and formatters (MongoDB, Cassandra, MinIO, HTTP) per spec sections 8 and 9.

## Dependencies (FOSS)

- **Redis**: `github.com/redis/go-redis/v9` – streams, consumer groups, hashes
- **MongoDB**: `go.mongodb.org/mongo-driver` – document store
- **MinIO**: `github.com/minio/minio-go/v7` – S3-compatible object storage
- **OpenTelemetry**: `go.opentelemetry.io/otel` – tracing (optional)

Cassandra support can be added via `github.com/gocql/gocql` for CQL queries.

## License

Use per your project license.
