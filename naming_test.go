package rabbitmq

import (
	"github.com/BHRK-codelabs/corekit/configkit"
	"testing"
)

func TestBuildTopologyNamesUsesConvention(t *testing.T) {
	t.Parallel()

	names := BuildTopologyNames(
		NamingConvention{
			Prefix:  "com.bhrk.shared",
			Env:     "staging",
			Domain:  "billing",
			Version: "v2",
		},
		TopologySpec{
			Routes: map[string]RouteSpec{
				"invoice-created": {MessageType: "event", Action: "created"},
				"invoice-paid":    {MessageType: "event", Action: "paid"},
			},
		},
	)

	if names.ExchangeName != "com.bhrk.shared.staging.billing.exchange" {
		t.Fatalf("unexpected exchange name: %s", names.ExchangeName)
	}
	paid, ok := names.Route("invoice-paid")
	if !ok {
		t.Fatal("expected invoice-paid route to be registered")
	}
	if paid.RoutingKey != "com.bhrk.shared.staging.billing.v2.event.paid" {
		t.Fatalf("unexpected paid routing key: %s", paid.RoutingKey)
	}
	if paid.QueueName != "com.bhrk.shared.staging.billing.v2.event.paid.queue" {
		t.Fatalf("unexpected paid queue name: %s", paid.QueueName)
	}
	if paid.DLQ.RoutingKey != "com.bhrk.shared.staging.billing.v2.event.paid.dlq" {
		t.Fatalf("unexpected paid dlq routing key: %s", paid.DLQ.RoutingKey)
	}
}

func TestBuildTopologyNamesFallsBackToDevEnv(t *testing.T) {
	t.Parallel()

	names := BuildTopologyNames(
		NamingConvention{
			Prefix:  "io.bhrk",
			Domain:  "notifications",
			Version: "v1",
		},
		TopologySpec{
			Routes: map[string]RouteSpec{
				"worker-sync": {MessageType: "command", Action: "sync"},
			},
		},
	)

	if names.ExchangeName != "io.bhrk.dev.notifications.exchange" {
		t.Fatalf("expected dev exchange fallback, got %s", names.ExchangeName)
	}
}

func TestNewTopologyNamesUsesConfigurableConvention(t *testing.T) {
	t.Parallel()

	names := NewTopologyNames(&configkit.Config{
		App: configkit.AppConfig{
			Environment: configkit.EnvStaging,
		},
		RabbitMQ: configkit.RabbitMQConfig{
			NamespacePrefix: "com.bhrk.shared",
			DomainName:      "billing",
			Version:         "v2",
		},
	}, TopologySpec{
		Routes: map[string]RouteSpec{
			"invoice-created": {MessageType: "event", Action: "created"},
		},
	})

	if names.ExchangeName != "com.bhrk.shared.staging.billing.exchange" {
		t.Fatalf("unexpected exchange name: %s", names.ExchangeName)
	}
	created, ok := names.Route("invoice-created")
	if !ok {
		t.Fatal("expected invoice-created route to be registered")
	}
	if created.RoutingKey != "com.bhrk.shared.staging.billing.v2.event.created" {
		t.Fatalf("unexpected invoice-created routing key: %s", created.RoutingKey)
	}
}

