# Go HTTP Server: Task API

A small, production-style HTTP server written in Go using **only the standard library**.
It exposes a JSON REST API for managing a to-do list and serves a tiny web UI.

The project is meant for learning. It shows how to:

- route requests with Go 1.22+ `net/http` patterns (`GET /tasks/{id}`)
- read and write JSON safely (size limits, unknown-field rejection)
- write middleware (request logging, panic recovery)
- protect shared data with a mutex
- configure server timeouts and graceful shutdown
- test handlers with `net/http/httptest`

---

## Requirements

- Go **1.22 or newer**, because the router uses method and path-wildcard patterns

## Quick start

```bash
cd HTTP_server
go run .                 # listens on :8080
```

Open <http://localhost:8080> in a browser to see the web UI, or use `curl`:

```bash
curl -s localhost:8080/health
```

### Configuration

| Option    | Flag      | Env var | Default    |
|-----------|-----------|---------|------------|
| Listen address | `-addr`   | `PORT` (port only) | `:8080` |
| Static files dir | `-static` | none | `static` (use `-static ""` to turn it off) |

```bash
go run . -addr 127.0.0.1:9000
PORT=3000 go run .
```

### Make targets

| Command      | What it does                     |
|--------------|----------------------------------|
| `make run`   | Run the server                   |
| `make build` | Build the binary `bin/httpserver`  |
| `make test`  | Run tests with the race detector |
| `make vet`   | Run `go vet` static analysis     |
| `make fmt`   | Format the code                  |

---

## API reference

All responses are JSON. Errors always look like this:

```json
{ "error": "description of what went wrong" }
```

### Task object

```json
{
  "id": 1,
  "title": "learn Go",
  "done": false,
  "created_at": "2026-10-23T12:00:00Z"
}
```

### Endpoints

| Method   | Path          | Description          | Success |
|----------|---------------|----------------------|---------|
| `GET`    | `/health`     | Liveness check       | `200`   |
| `GET`    | `/tasks`      | List all tasks       | `200`   |
| `POST`   | `/tasks`      | Create a task        | `201`   |
| `GET`    | `/tasks/{id}` | Get one task         | `200`   |
| `PATCH`  | `/tasks/{id}` | Partially update a task | `200` |
| `DELETE` | `/tasks/{id}` | Delete a task        | `204`   |

#### `POST /tasks`

The body must be JSON. `title` is required and cannot be blank.

```bash
curl -s -X POST localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"learn Go"}'
```

#### `PATCH /tasks/{id}`

Send only the fields you want to change. Any field you leave out stays the same.

```bash
curl -s -X PATCH localhost:8080/tasks/1 -d '{"done":true}'
curl -s -X PATCH localhost:8080/tasks/1 -d '{"title":"master Go"}'
```

#### `DELETE /tasks/{id}`

```bash
curl -i -X DELETE localhost:8080/tasks/1    # HTTP/1.1 204 No Content
```

### Status codes

| Code  | Meaning |
|-------|---------|
| `400` | Bad JSON, an unknown field, a missing or blank title, an ID that isn't a number, or a body over 1 MiB |
| `404` | No task has that ID |
| `405` | That HTTP method isn't allowed on this path. The `Allow` header lists the ones that are. |
| `500` | Unexpected server error, including a recovered panic |

---

## Project structure

```
HTTP_server/
├── main.go                         # entry point: config, server, graceful shutdown
├── internal/
│   ├── handler/
│   │   ├── handler.go              # routes + HTTP handlers + JSON helpers
│   │   └── handler_test.go         # table-driven API tests
│   ├── middleware/
│   │   └── middleware.go           # Logger, Recover, Chain
│   └── store/
│       └── store.go                # thread-safe in-memory task storage
├── static/
│   └── index.html                  # minimal web UI
├── Makefile
├── go.mod
└── README.md
```

Code under `internal/` can only be imported by this module. That keeps the public surface of the project small.

## How a request flows

```
client ──► http.Server (timeouts)
              │
              ▼
        Recover middleware   (catches panics → 500)
              │
              ▼
        Logger middleware    (logs method, path, status, duration)
              │
              ▼
        ServeMux router      ("GET /tasks/{id}" → getTask)
              │
              ▼
        Handler              (validate input, call store, write JSON)
              │
              ▼
        Store                (map + sync.RWMutex)
```

## Design notes

- **Timeouts.** `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` and `IdleTimeout` are all set. A plain `http.ListenAndServe` has no timeouts, so slow clients (as in a Slowloris attack) could hold connections open forever.
- **Graceful shutdown.** When the server gets Ctrl+C or `SIGTERM`, it stops accepting new connections and gives in-flight requests up to 10 seconds to finish.
- **Input safety.** `http.MaxBytesReader` caps request bodies at 1 MiB, and `DisallowUnknownFields` rejects JSON with fields the server doesn't expect.
- **Concurrency.** Every request runs in its own goroutine. The store therefore uses a `sync.RWMutex`, which lets many requests read at once but only one write at a time.
- **PATCH with pointers.** The update request uses `*string` and `*bool` fields. That lets the server tell a field that wasn't sent apart from one sent as `""` or `false`.
- **Storage.** Data is kept in memory only and is lost on restart. To make it persistent, swap in a database-backed store that has the same methods.

## Testing

```bash
go test -race ./...
```

The tests run the whole router with `httptest`, so no real network port is needed.

## Ideas to extend

- Persist tasks to SQLite or a JSON file
- Add authentication middleware (an API key or JWT)
- Add pagination and filtering (`GET /tasks?done=true`)
- Add a Dockerfile
- Serve over HTTPS with `srv.ListenAndServeTLS`
