# Topology

`messagingrabbitmq` includes topology helpers to keep naming and declaration outside application business code.

## Naming

```go
names := rabbitmq.NewTopologyNames(cfg, spec)
```

The naming helpers are driven by:
- namespace prefix
- environment
- domain
- version

## Declaration

The declaration helpers create exchanges, queues and bindings based on the neutral topology structures defined in the package.

## Boundary

- topology helpers are adapter-specific because they reflect RabbitMQ concepts
- the neutral capability remains `messagingkit`
- topology naming should stay free of business channel assumptions
# Topology

`messagingrabbitmq` supports different exchange kinds through `TopologySpec.ExchangeKind`.

## Exchange kinds

- `topic`
  Use for integration events and routing by semantic key patterns.
- `direct`
  Use for request-reply, command routing or single-target dispatch.
- `fanout`
  Use for broadcast where routing keys are intentionally ignored.

If omitted, the adapter defaults to `topic`.

## Example

```go
spec := rabbitmq.TopologySpec{
    ExchangeKind: "direct",
    Routes: map[string]rabbitmq.RouteSpec{
        "files.lookup": {
            MessageType: "files",
            Action:      "lookup",
        },
    },
}
```

For `fanout`, routing keys become empty by design.
