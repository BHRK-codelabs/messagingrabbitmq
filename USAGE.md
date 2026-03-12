# Usage

## Publish a canonical message

```go
err := rabbitmq.PublishMessage(ctx, ch, cfg, exchange, route, message)
```

## Build an AMQP publishing directly

```go
publishing, err := rabbitmq.BuildPublishing(ctx, message)
```

## Handle an incoming delivery

```go
err := rabbitmq.HandleDelivery(ctx, delivery, func(ctx context.Context, message canonicalkit.Message[MyPayload], raw amqp.Delivery) error {
    // process message
    return nil
})
```

## Adapter-level handling for messagingkit

```go
result := rabbitmq.HandleMessage(ctx, delivery, handler)
```

## Request-reply

Use `direct` when you want a command/request style interaction with a single expected responder.

```go
client := rabbitmq.NewRequestClient(channel, publisher, 5*time.Second)
reply, err := client.Call(ctx, messagingkit.Route{Key: "files.lookup"}, requestDelivery)
```

On the consumer side:

```go
result := rabbitmq.HandleRequest(ctx, channel, delivery, func(ctx context.Context, request messagingkit.Delivery) (messagingkit.Delivery, messagingkit.Result) {
    return responseDelivery, messagingkit.Ack()
})
```

## Decode payload

```go
payload, err := rabbitmq.DecodePayload[MyPayload](delivery)
```
