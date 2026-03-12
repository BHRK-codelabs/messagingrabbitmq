package rabbitmq

import (
	"github.com/BHRK-codelabs/corekit/configkit"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	DLQExchangeKey = "x-dead-letter-exchange"
	DLQRoutingKey  = "x-dead-letter-routing-key"
)

func SetupTopology(cfg *configkit.Config, ch *amqp.Channel, spec TopologySpec) (TopologyNames, error) {
	names := NewTopologyNames(cfg, spec)

	if err := ch.ExchangeDeclare(names.ExchangeName, names.ExchangeKind, true, false, false, false, nil); err != nil {
		return TopologyNames{}, err
	}
	if err := ch.ExchangeDeclare(names.DLQExchangeName, "topic", true, false, false, false, nil); err != nil {
		return TopologyNames{}, err
	}

	for _, route := range names.Routes {
		if err := declareRoute(ch, names.DLQExchangeName, route); err != nil {
			return TopologyNames{}, err
		}
	}
	for _, route := range names.Routes {
		if err := bindMainRoute(ch, names.ExchangeName, names.DLQExchangeName, route); err != nil {
			return TopologyNames{}, err
		}
	}

	return names, nil
}

func declareRoute(ch *amqp.Channel, dlqExchangeName string, route RouteNames) error {
	if _, err := ch.QueueDeclare(route.DLQ.QueueName, true, false, false, false, nil); err != nil {
		return err
	}
	return ch.QueueBind(route.DLQ.QueueName, route.DLQ.RoutingKey, dlqExchangeName, false, nil)
}

func bindMainRoute(ch *amqp.Channel, exchangeName, dlqExchangeName string, route RouteNames) error {
	args := amqp.Table{
		DLQExchangeKey: dlqExchangeName,
		DLQRoutingKey:  route.DLQ.RoutingKey,
	}

	if _, err := ch.QueueDeclare(route.QueueName, true, false, false, false, args); err != nil {
		return err
	}
	return ch.QueueBind(route.QueueName, route.RoutingKey, exchangeName, false, nil)
}

