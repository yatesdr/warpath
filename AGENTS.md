# AGENTS.md

Guide for AI agents working in the Shingo codebase.

## Project Overview

Shingo (信号) — material tracking and automated transport system for manufacturing plants. Manages payload flow between warehouse storage and production line stations using autonomous mobile robots.

Monorepo with **3 independent Go 1.24 modules**:

| Module | Import Path | Description |
|--------|-------------|-------------|
| `shingo-core/` | `shingocore` | Central server: dispatch, fleet management, state tracking |
| `shingo-edge/` | `shingoedge` | Shop-floor client: PLC integration, material tracking, operator workflows |
| `protocol/` | `shingo/protocol` | Shared wire protocol: JSON envelope, message types, two-phase decode |

Each module has its own `go.mod` and builds independently. The `protocol` module is vendored via `replace` directive in both core and edge.

## Build & Test Commands

```sh
# Core (has Makefile)
cd shingo-core
make build          # build for current platform
make test           # go test -v ./...
make fmt            # go fmt ./...
make vet            # go vet ./...
make all            # cross-compile linux/windows/macos

# Edge (no Makefile)
cd shingo-edge
go build ./...
go test -v ./...

# Protocol
cd protocol
go test -v ./...
```

### Run Single Test

```sh
cd shingo-core && go test -v ./dispatch -run TestHandleOrderRequest -count=1
```

### Run Applications

```sh
# Core
cd shingo-core && go run ./cmd/shingocore

# With debug logging (subsystem-filtered)
cd shingo-core && go run ./cmd/shingocore --log-debug=rds,dispatch
cd shingo-edge && go run ./cmd/shingoedge --log-debug=orders,plc
```

## CI

GitHub Actions workflows in `.github/workflows/`:
- `core.yml` — builds and tests shingo-core on push to `shingo-core/**` or `protocol/**`
- `edge.yml` — builds and tests shingo-edge on push to `shingo-edge/**` or `protocol/**`
- `protocol.yml` — builds and tests protocol module
- `release.yml` — release builds

Go version: 1.24

## Module Structure

### shingo-core Packages

```
cmd/shingocore/    # main entry point
config/            # YAML config loading
engine/            # orchestrates all subsystems
dispatch/          # order dispatch logic, node resolution
messaging/         # Kafka client, outbox pattern
rds/               # Seer RDS fleet backend implementation
fleet/             # vendor-neutral fleet interface
store/             # database access (SQLite/Postgres)
nodestate/         # node state tracking
www/               # HTTP handlers, templates, static assets
debuglog/          # filtered debug logging
```

### shingo-edge Packages

```
cmd/shingoedge/    # main entry point
config/            # YAML config loading
engine/            # orchestrates subsystems
orders/            # order state machine, operator workflows
plc/               # PLC integration, SSE client
changeover/        # production changeover handling
messaging/         # Kafka client
store/             # SQLite database access
www/               # HTTP handlers, templates
debuglog/          # filtered debug logging
```

### protocol Package

```
envelope.go        # JSON envelope structure
types.go           # message type constants
payloads.go        # payload structs for all message types
ingestor.go        # two-phase decode, message dispatch
expiry.go          # TTL/expiry handling
signing.go         # optional HMAC-SHA256 message signing
noop_handler.go    # embed to implement only needed handlers
```

## Architecture Patterns

### EventBus

Sync pub/sub within each component for inter-subsystem decoupling. Defined in `engine/events.go`, wired during `engine.Start()`. Event types include order lifecycle, fleet connectivity, bin changes.

```go
// Subscribe
engine.Events.On(engine.EventOrderDispatched, func(e any) {
    ev := e.(engine.OrderDispatchedEvent)
    // handle event
})

// Publish
engine.Events.Emit(engine.OrderDispatchedEvent{OrderID: 42, ...})
```

### Emitter Interfaces

Each package defines its own emitter interface locally to avoid import cycles:

- `dispatch.Emitter` — order lifecycle events
- `plc.Emitter` — PLC events

Engine wiring creates adapter structs that bridge these to the EventBus. See `engine/engine.go` for adapter implementations.

### Outbox Pattern

Messages written to an `outbox` DB table first, then drained to Kafka periodically via `messaging.OutboxDrainer`. Ensures at-least-once delivery even when Kafka is unavailable.

### Two-Phase Protocol Decode

`protocol.Ingestor` decodes in two phases:

1. **Phase 1**: Parse only routing fields (`v`, `type`, `id`, `dst`, `exp`) for filtering and expiry check. Payload bytes untouched.
2. **Phase 2**: If Phase 1 passes, deserialize full envelope including `p` field based on `type`.

This allows efficient filtering without paying the cost of full deserialization.

### Database Dialect Abstraction

Core's `store.DB` uses `?` placeholders with `Rebind()` to convert to `$1, $2, ...` for PostgreSQL. SQLite uses `?` natively.

```go
query := store.Q(`SELECT * FROM orders WHERE id = ? AND status = ?`)
// Returns query as-is for SQLite, or with $1, $2 for Postgres
```

### Fleet Backend Interface

`fleet.Backend` in `shingo-core/fleet/fleet.go` abstracts the robot fleet system. Current implementation: Seer RDS in `rds/` package. Additional capabilities via optional interfaces:

- `fleet.RobotLister` — robot status and control
- `fleet.SceneSyncer` — scene/map data sync
- `fleet.NodeOccupancyProvider` — location occupancy

## Messaging

Kafka with dual topics:

| Topic | Direction | Purpose |
|-------|-----------|---------|
| `shingo.orders` | Edge → Core | Order requests, heartbeats, registration |
| `shingo.dispatch` | Core → Edge | Acknowledgements, status updates, errors |

Consumer groups:
- Core: `shingo-core` on `shingo.orders`
- Edge: `shingo-edge-{station_id}` on `shingo.dispatch`

Full protocol spec: `docs/wire-protocol.md`

## Testing Patterns

### Temp SQLite DBs

```go
func testDB(t *testing.T) *store.DB {
    t.Helper()
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "test.db")
    db, err := store.Open(&config.DatabaseConfig{
        Driver: "sqlite",
        SQLite: config.SQLiteConfig{Path: dbPath},
    })
    if err != nil {
        t.Fatalf("open test db: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}
```

### Mock Interfaces

Mock emitter/backend interfaces to test without external dependencies:

```go
// dispatch/dispatcher_test.go
type mockEmitter struct {
    received   []emitReceived
    dispatched []emitDispatched
    // ...
}

type mockBackend struct{}
func (m *mockBackend) CreateTransportOrder(req fleet.TransportOrderRequest) (fleet.TransportOrderResult, error) {
    return fleet.TransportOrderResult{}, fmt.Errorf("mock: not connected")
}
```

## Domain Terms

| Term | Description |
|------|-------------|
| **Node** | Physical floor location (storage, line-side, staging, lane slot) |
| **Bin** | Physical container tracked at a node (`store.Bin`) |
| **BinType** | Container class (size, form factor) |
| **Payload** | Template defining bin contents and UOP capacity (`store.Payload`) |
| **Manifest** | JSON parts list on a bin (`bins.manifest`); payload_manifest = template items from payload |
| **Order** | Transport request (retrieve/store/move): edge→core→fleet |
| **Station** | An edge instance identity (`{namespace}.{line_id}`) |
| **Process** | Production area (DB: `production_lines`) |
| **Style** | End-item type (DB: `job_styles`) |

See `docs/terminology.md` for vendor term mappings.

## Conventions

- **Config**: YAML (`shingocore.yaml` / `shingoedge.yaml`). Never hardcode defaults in code.
- **Database migrations**: Run on startup via `store.migrate()`
- **Static assets/templates**: Use `go:embed`
- **Auth**: gorilla/sessions + bcrypt
- **Protocol TTL** (`exp` field): Absolute UTC timestamp, not relative duration
- **No AI co-author references in commits**

## Naming Conventions

- **Go interfaces/structs**: ShinGo terms (`store.Bin`, `store.Blueprint`, `store.Payload`, `store.Node`)
- **JSON API fields**: `snake_case` (`bin_type_id`, `manifest_confirmed`, `node_id`)
- **HTML/CSS classes**: `kebab-case` (`tile-loc`, `occupancy-modal`)

## Configuration

Example config: `shingo-core/shingocore.example.yaml`

Key sections:
- `database`: SQLite or Postgres connection
- `rds`: Fleet backend URL, poll interval, timeout
- `web`: HTTP server host/port, session secret
- `messaging`: Kafka brokers, topics, station ID

## Documentation

| Document | Location |
|----------|----------|
| Data model | `docs/data-model.md` |
| Bins & blueprints | `docs/payloads.md` |
| Terminology | `docs/terminology.md` |
| Wire protocol spec | `docs/wire-protocol.md` |
| Core architecture | `shingo-core/docs/architecture.md` |
| REST API reference | `shingo-core/docs/api-reference.md` |
| Configuration | `shingo-core/docs/configuration.md` |

## Start Scripts

Repository root contains convenience scripts:

```sh
./start-core.sh          # Pull latest, run core
./start-core-debug.sh    # Run with --log-debug flag
./start-edge.sh          # Pull latest, run edge
./start-edge-debug.sh    # Run with --log-debug flag
```
