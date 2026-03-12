package rabbitmq

import (
	"context"
	"testing"

	"github.com/BHRK-codelabs/corekit/canonicalkit"
	"github.com/BHRK-codelabs/messagingkit"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestBuildTopologyNamesUsesDirectExchangeWhenRequested(t *testing.T) {
	t.Parallel()

	names := BuildTopologyNames(NamingConvention{
		Prefix:  "io.bhrk",
		Env:     "local",
		Domain:  "files",
		Version: "v1",
	}, TopologySpec{
		ExchangeKind: "direct",
		Routes: map[string]RouteSpec{
			"files.get": {MessageType: "files", Action: "get"},
		},
	})

	if names.ExchangeKind != "direct" {
		t.Fatalf("unexpected exchange kind: %s", names.ExchangeKind)
	}
	route, ok := names.Route("files.get")
	if !ok {
		t.Fatal("expected route")
	}
	if route.RoutingKey == "" {
		t.Fatal("expected routing key")
	}
}

func TestBuildTopologyNamesUsesEmptyRoutingKeyForFanout(t *testing.T) {
	t.Parallel()

	names := BuildTopologyNames(NamingConvention{
		Prefix: "io.bhrk",
		Env:    "local",
		Domain: "files",
	}, TopologySpec{
		ExchangeKind: "fanout",
		Routes: map[string]RouteSpec{
			"files.broadcast": {MessageType: "files", Action: "broadcast"},
		},
	})

	route, ok := names.Route("files.broadcast")
	if !ok {
		t.Fatal("expected route")
	}
	if route.RoutingKey != "" {
		t.Fatalf("expected empty routing key for fanout, got %q", route.RoutingKey)
	}
}

func TestHandleRequestReturnsAckWhenNoReplyTo(t *testing.T) {
	t.Parallel()

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer: "files-api",
		Kind:     canonicalkit.MessageKindCommand,
		Type:     "files.lookup",
	}, map[string]string{"id": "f-1"})

	pub, err := BuildPublishing(context.Background(), message)
	if err != nil {
		t.Fatalf("build publishing: %v", err)
	}

	result := HandleRequest(context.Background(), nil, amqp.Delivery{
		Body:          pub.Body,
		Headers:       pub.Headers,
		MessageId:     pub.MessageId,
		Type:          pub.Type,
		CorrelationId: "corr-1",
	}, func(ctx context.Context, delivery messagingkit.Delivery) (messagingkit.Delivery, messagingkit.Result) {
		return delivery, messagingkit.Ack()
	})

	if result.Outcome != messagingkit.OutcomeAck {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
}
