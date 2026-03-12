package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/BHRK-codelabs/messagingkit"
	"github.com/BHRK-codelabs/corekit/canonicalkit"
	"github.com/BHRK-codelabs/corekit/contextkit"
	"github.com/BHRK-codelabs/corekit/obskit"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

type MessageHandler[T any] func(context.Context, canonicalkit.Message[T], amqp.Delivery) error

func HandleDelivery[T any](ctx context.Context, delivery amqp.Delivery, handler MessageHandler[T]) error {
	ctx = ContextFromDelivery(ctx, delivery)

	message, err := DecodeDelivery[T](delivery)
	if err != nil {
		return err
	}

	ctx = contextkit.WithMessageMeta(ctx, message.Meta)
	ctx, span := obskit.StartSpan(ctx, "messaging.rabbitmq.consume",
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.operation", "process"),
		attribute.String("messaging.destination.name", delivery.Exchange),
		attribute.String("messaging.rabbitmq.routing_key", delivery.RoutingKey),
		attribute.String("messaging.message.id", message.Meta.MessageID),
		attribute.String("messaging.message.type", message.Meta.Type),
	)
	defer span.End()

	return handler(ctx, message, delivery)
}

func DecodePayload[T any](delivery amqp.Delivery) (T, error) {
	var payload T

	message, err := DecodeDelivery[T](delivery)
	if err != nil {
		return payload, err
	}

	return message.Payload, nil
}

func RawMessage(delivery amqp.Delivery) (json.RawMessage, error) {
	var message canonicalkit.Message[json.RawMessage]
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		return nil, err
	}
	return message.Payload, nil
}

func DeliveryFromAMQP(delivery amqp.Delivery) (messagingkit.Delivery, error) {
	message, err := DecodeDelivery[json.RawMessage](delivery)
	if err != nil {
		return messagingkit.Delivery{}, err
	}

	headers := map[string]string{}
	for key, value := range delivery.Headers {
		if str, ok := value.(string); ok {
			headers[key] = str
		}
	}

	return messagingkit.Delivery{
		Meta:    message.Meta,
		Payload: message.Payload,
		Headers: headers,
		Route: messagingkit.Route{
			Key: delivery.RoutingKey,
		},
	}, nil
}

func HandleMessage(ctx context.Context, delivery amqp.Delivery, handler messagingkit.Handler) messagingkit.Result {
	result := messagingkit.Ack()
	err := HandleDelivery[json.RawMessage](ctx, delivery, func(ctx context.Context, _ canonicalkit.Message[json.RawMessage], incoming amqp.Delivery) error {
		messageDelivery, err := DeliveryFromAMQP(incoming)
		if err != nil {
			return err
		}
		messageDelivery.Route.Key = incoming.RoutingKey

		result = handler(ctx, messageDelivery)
		if result.Outcome == "" {
			result.Outcome = messagingkit.OutcomeAck
		}
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
	if err != nil {
		if result.Outcome == "" {
			return messagingkit.Discard(err)
		}
		result.Error = err
		return result
	}
	return result
}

