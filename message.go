package rabbitmq

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/BHRK-codelabs/corekit/canonicalkit"
	"github.com/BHRK-codelabs/corekit/configkit"
	"github.com/BHRK-codelabs/corekit/contextkit"
	"github.com/BHRK-codelabs/corekit/obskit"
	amqp "github.com/rabbitmq/amqp091-go"
)

func BuildPublishing[T any](ctx context.Context, message canonicalkit.Message[T]) (amqp.Publishing, error) {
	body, err := json.Marshal(message)
	if err != nil {
		return amqp.Publishing{}, err
	}

	ctx = contextkit.WithMessageMeta(ctx, message.Meta)
	headers := amqp.Table{}
	headers = obskit.InjectAMQPHeaders(ctx, headers)
	injectAMQPHeadersFromContext(ctx, headers)

	return amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Headers:      headers,
		Body:         body,
		MessageId:    message.Meta.MessageID,
		Type:         message.Meta.Type,
	}, nil
}

func PublishMessage[T any](ctx context.Context, ch *amqp.Channel, cfg *configkit.Config, exchange string, route RouteNames, message canonicalkit.Message[T]) error {
	pub, err := BuildPublishing(ctx, message)
	if err != nil {
		return err
	}

	publishCtx := ctx
	cancel := func() {}
	if cfg != nil && cfg.RabbitMQ.PublishTimeout > 0 {
		publishCtx, cancel = context.WithTimeout(ctx, cfg.RabbitMQ.PublishTimeout)
	}
	defer cancel()

	return ch.PublishWithContext(publishCtx, exchange, route.RoutingKey, false, false, pub)
}

func ContextFromDelivery(ctx context.Context, delivery amqp.Delivery) context.Context {
	ctx = obskit.ExtractAMQPContext(ctx, delivery.Headers)
	ctx = contextFromAMQPHeaders(ctx, delivery.Headers)

	if delivery.MessageId != "" {
		ctx = contextkit.WithMessageMeta(ctx, canonicalkit.Metadata{
			MessageID: delivery.MessageId,
			Type:      delivery.Type,
		})
	}
	return ctx
}

func DecodeDelivery[T any](delivery amqp.Delivery) (canonicalkit.Message[T], error) {
	var message canonicalkit.Message[T]
	err := json.Unmarshal(delivery.Body, &message)
	return message, err
}

func injectAMQPHeadersFromContext(ctx context.Context, headers amqp.Table) {
	if correlationID, ok := contextkit.CorrelationID(ctx); ok {
		headers[canonicalkit.HeaderCorrelationID] = correlationID
	}
	if messageID, ok := contextkit.MessageID(ctx); ok {
		headers[canonicalkit.HeaderMessageID] = messageID
	}
	if causationID, ok := contextkit.CausationID(ctx); ok {
		headers[canonicalkit.HeaderCausationID] = causationID
	}
	if producer, ok := contextkit.Producer(ctx); ok {
		headers[canonicalkit.HeaderProducer] = producer
	}
	if messageKind, ok := contextkit.MessageKind(ctx); ok {
		headers[canonicalkit.HeaderMessageKind] = messageKind
	}
	if messageType, ok := contextkit.MessageType(ctx); ok {
		headers[canonicalkit.HeaderMessageType] = messageType
	}
	if aggregateID, ok := contextkit.AggregateID(ctx); ok {
		headers[canonicalkit.HeaderAggregateID] = aggregateID
	}
	if tenantID, ok := contextkit.TenantID(ctx); ok {
		headers[contextkit.HeaderTenantID] = tenantID
	}
	if actorID, ok := contextkit.ActorID(ctx); ok {
		headers[contextkit.HeaderActorID] = actorID
	}
	if idempotencyKey, ok := contextkit.IdempotencyKey(ctx); ok {
		headers[contextkit.HeaderIdempotencyKey] = idempotencyKey
	}
	if idempotencyMode, ok := contextkit.IdempotencyMode(ctx); ok {
		headers[contextkit.HeaderIdempotencyMode] = idempotencyMode
	}
}

func contextFromAMQPHeaders(ctx context.Context, headers amqp.Table) context.Context {
	httpHeaders := http.Header{}
	for key, value := range headers {
		if str, ok := value.(string); ok {
			httpHeaders[key] = []string{str}
		}
	}
	return contextkit.FromHTTPHeaders(ctx, httpHeaders)
}

