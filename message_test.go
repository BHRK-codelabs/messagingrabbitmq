package rabbitmq

import (
	"context"
	"testing"

	"github.com/BHRK-codelabs/corekit/canonicalkit"
	"github.com/BHRK-codelabs/corekit/contextkit"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestBuildPublishingProjectsContextAndMessageMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = contextkit.WithTenantID(ctx, "tenant-1")
	ctx = contextkit.WithActorID(ctx, "actor-1")
	ctx = contextkit.WithIdempotency(ctx, "idem-1", "enforce", "new")

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer:    "orders-api",
		Kind:        canonicalkit.MessageKindEvent,
		Type:        "orders.created",
		AggregateID: "ord-1",
	}, map[string]string{"id": "ord-1"})

	pub, err := BuildPublishing(ctx, message)
	if err != nil {
		t.Fatalf("build publishing: %v", err)
	}

	if pub.MessageId != message.Meta.MessageID {
		t.Fatalf("unexpected message id: %s", pub.MessageId)
	}
	if got := pub.Headers[contextkit.HeaderTenantID]; got != "tenant-1" {
		t.Fatalf("unexpected tenant header: %v", got)
	}
	if got := pub.Headers[contextkit.HeaderIdempotencyKey]; got != "idem-1" {
		t.Fatalf("unexpected idempotency header: %v", got)
	}
}

func TestContextFromDeliveryRestoresTransversalMetadata(t *testing.T) {
	t.Parallel()

	delivery := amqp.Delivery{
		MessageId: "msg-1",
		Type:      "orders.created",
		Headers: amqp.Table{
			canonicalkit.HeaderCorrelationID: "corr-1",
			canonicalkit.HeaderCausationID:   "cause-1",
			contextkit.HeaderTenantID:        "tenant-9",
			contextkit.HeaderActorID:         "actor-9",
		},
	}

	ctx := ContextFromDelivery(context.Background(), delivery)
	if correlationID, ok := contextkit.CorrelationID(ctx); !ok || correlationID != "corr-1" {
		t.Fatalf("unexpected correlation id: %v %v", correlationID, ok)
	}
	if tenantID, ok := contextkit.TenantID(ctx); !ok || tenantID != "tenant-9" {
		t.Fatalf("unexpected tenant id: %v %v", tenantID, ok)
	}
	if messageID, ok := contextkit.MessageID(ctx); !ok || messageID != "msg-1" {
		t.Fatalf("unexpected message id: %v %v", messageID, ok)
	}
}

