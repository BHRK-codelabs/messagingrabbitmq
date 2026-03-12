# messagingrabbitmq docs

This package adapts RabbitMQ to the neutral `messagingkit` capability.

It is responsible for:
- publishing canonical messages as AMQP messages
- decoding AMQP deliveries into canonical messages
- propagating trace and transversal context through AMQP headers
- mapping RabbitMQ routes and topology names

It is intended to be wired by service or worker composition code, not imported by the neutral capability itself.
