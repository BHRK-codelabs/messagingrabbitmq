package rabbitmq

import (
	"context"

	"github.com/BHRK-codelabs/messagingkit"
	"github.com/BHRK-codelabs/corekit/canonicalkit"
	"github.com/BHRK-codelabs/corekit/configkit"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	cfg      *configkit.Config
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(cfg *configkit.Config, channel *amqp.Channel, exchange string) *Publisher {
	return &Publisher{
		cfg:      cfg,
		channel:  channel,
		exchange: exchange,
	}
}

func NewQueuePublisher(cfg *configkit.Config, channel *amqp.Channel, exchange string) *Publisher {
	return NewPublisher(cfg, channel, exchange)
}

func (p *Publisher) Publish(ctx context.Context, route messagingkit.Route, delivery messagingkit.Delivery) error {
	rawMessage := canonicalkit.Message[jsonRawPayload]{
		Meta:    delivery.Meta,
		Payload: jsonRawPayload(delivery.Payload),
	}

	return PublishMessage(ctx, p.channel, p.cfg, p.exchange, RouteNames{RoutingKey: route.Key}, rawMessage)
}

type jsonRawPayload []byte

func (p jsonRawPayload) MarshalJSON() ([]byte, error) {
	return []byte(p), nil
}

