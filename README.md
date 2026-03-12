# messagingrabbitmq

`messagingrabbitmq` is the RabbitMQ adapter for `capabilities/messagingkit`.

It provides:
- canonical message publishing
- AMQP delivery decoding
- context and tracing propagation through AMQP headers
- adapter handling for `messagingkit`
- topology naming and declaration helpers
- request-reply helpers
- exchange kind support for `topic`, `direct` and `fanout`

## Package structure

- `message.go`: publishing and AMQP context/header propagation
- `consumer.go`: delivery handling and adapter-facing message processing
- `queueing.go`: `messagingkit` publishing integration
- `naming.go`: neutral topology naming model
- `topology.go`: config-backed topology naming construction
- `declare.go`: exchange/queue declaration helpers
- `requestreply.go`: request-reply client and handler helpers

## Local docs

- [Overview](OVERVIEW.md)
- [Usage](USAGE.md)
- [Topology](TOPOLOGY.md)

## Design notes

- this package is an adapter, not part of the neutral messaging capability
- it implements the single `messagingkit` port
- queueing and eventing semantics should be layered on top of `messagingkit`, not reintroduced as parallel transport ports here
