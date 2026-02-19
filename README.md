# Shingo

Material tracking and automated transport system for manufacturing plants. Manages the flow of payloads (bins, containers, raw materials) between warehouse storage and production line stations using autonomous mobile robots.

## Modules

| Module | Description |
|--------|-------------|
| **shingo-core** | Central server. Receives orders from edge nodes, resolves source/destination, dispatches transport orders to the robot fleet, and tracks fulfillment. Supports SQLite and PostgreSQL. |
| **shingo-edge** | Shop-floor client. Runs at each production line. Tracks PLC counters, manages payload inventory, handles operator order workflows (retrieve, store, move), and communicates with core via messaging. |
| **protocol** | Shared wire protocol. Defines the JSON envelope format, message types, payload schemas, two-phase decode, and TTL-based expiry used by both core and edge. |

## Structure

```
shingo/
  protocol/       shared Go module (shingo/protocol)
  shingo-core/    central server (module: shingocore)
  shingo-edge/    shop-floor client (module: shingoedge)
  docs/           wire protocol specification
```

## Building

Each module builds independently. Go 1.24+.

```sh
cd shingo-core && go build ./...
cd shingo-edge && go build ./...
cd protocol    && go test ./...
```

## Messaging

Core and edge communicate over Kafka using a unified JSON envelope protocol with dual topics:

- `shingo.orders` -- edge to core (order requests, heartbeats)
- `shingo.dispatch` -- core to edge (acknowledgements, status updates)

See [docs/wire-protocol.md](docs/wire-protocol.md) for the full specification.

## License

Proprietary. See [LICENSE](LICENSE).
