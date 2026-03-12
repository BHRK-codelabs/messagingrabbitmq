package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/BHRK-codelabs/messagingkit"
	"github.com/BHRK-codelabs/corekit/canonicalkit"
	"github.com/BHRK-codelabs/corekit/contextkit"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestHandleDeliveryBuildsContextAndDecodesMessage(t *testing.T) {
	t.Parallel()

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer:    "payments-worker",
		Kind:        canonicalkit.MessageKindEvent,
		Type:        "payments.approved",
		AggregateID: "pay-1",
	}, map[string]string{"payment_id": "pay-1"})

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	delivery := amqp.Delivery{
		Body:       body,
		MessageId:  message.Meta.MessageID,
		Type:       message.Meta.Type,
		Exchange:   "payments.exchange",
		RoutingKey: "payments.approved",
		Headers: amqp.Table{
			canonicalkit.HeaderCorrelationID: "corr-1",
			contextkit.HeaderTenantID:        "tenant-1",
		},
	}

	err = HandleDelivery(context.Background(), delivery, func(ctx context.Context, got canonicalkit.Message[map[string]string], incoming amqp.Delivery) error {
		if incoming.RoutingKey != "payments.approved" {
			t.Fatalf("unexpected routing key: %s", incoming.RoutingKey)
		}
		if got.Meta.MessageID != message.Meta.MessageID {
			t.Fatalf("unexpected message id: %s", got.Meta.MessageID)
		}
		if got.Payload["payment_id"] != "pay-1" {
			t.Fatalf("unexpected payload: %#v", got.Payload)
		}
		if correlationID, ok := contextkit.CorrelationID(ctx); !ok || correlationID != message.Meta.CorrelationID {
			t.Fatalf("unexpected correlation id: %v %v", correlationID, ok)
		}
		if tenantID, ok := contextkit.TenantID(ctx); !ok || tenantID != "tenant-1" {
			t.Fatalf("unexpected tenant id: %v %v", tenantID, ok)
		}
		if messageID, ok := contextkit.MessageID(ctx); !ok || messageID != message.Meta.MessageID {
			t.Fatalf("unexpected message id in context: %v %v", messageID, ok)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("handle delivery: %v", err)
	}
}

func TestHandleDeliveryReturnsDecodeError(t *testing.T) {
	t.Parallel()

	delivery := amqp.Delivery{Body: []byte("not-json")}
	err := HandleDelivery[map[string]string](context.Background(), delivery, func(context.Context, canonicalkit.Message[map[string]string], amqp.Delivery) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestHandleDeliveryReturnsHandlerError(t *testing.T) {
	t.Parallel()

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer: "orders-worker",
		Kind:     canonicalkit.MessageKindCommand,
		Type:     "orders.reserve",
	}, map[string]string{"order_id": "ord-1"})

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	expected := errors.New("handler failed")
	err = HandleDelivery(context.Background(), amqp.Delivery{Body: body}, func(context.Context, canonicalkit.Message[map[string]string], amqp.Delivery) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("unexpected handler error: %v", err)
	}
}

func TestDecodePayloadReturnsPayload(t *testing.T) {
	t.Parallel()

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer: "inventory-worker",
		Kind:     canonicalkit.MessageKindEvent,
		Type:     "inventory.updated",
	}, map[string]string{"sku": "sku-1"})

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	payload, err := DecodePayload[map[string]string](amqp.Delivery{Body: body})
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["sku"] != "sku-1" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestRawMessageReturnsRawPayload(t *testing.T) {
	t.Parallel()

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer: "audit-worker",
		Kind:     canonicalkit.MessageKindEvent,
		Type:     "audit.logged",
	}, json.RawMessage(`{"severity":"high"}`))

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	raw, err := RawMessage(amqp.Delivery{Body: body})
	if err != nil {
		t.Fatalf("raw message: %v", err)
	}
	if string(raw) != `{"severity":"high"}` {
		t.Fatalf("unexpected raw payload: %s", string(raw))
	}
}

func TestDeliveryFromAMQPBuildsMessagingDelivery(t *testing.T) {
	t.Parallel()

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer: "orders-worker",
		Kind:     canonicalkit.MessageKindEvent,
		Type:     "orders.created",
	}, json.RawMessage(`{"order_id":"ord-1"}`))

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	delivery, err := DeliveryFromAMQP(amqp.Delivery{
		Body:       body,
		RoutingKey: "orders.created",
		Headers: amqp.Table{
			contextkit.HeaderTenantID: "tenant-1",
		},
	})
	if err != nil {
		t.Fatalf("delivery from amqp: %v", err)
	}

	if delivery.Route.Key != "orders.created" {
		t.Fatalf("unexpected route key: %s", delivery.Route.Key)
	}
	if string(delivery.Payload) != `{"order_id":"ord-1"}` {
		t.Fatalf("unexpected payload: %s", string(delivery.Payload))
	}
}

func TestHandleMessageReturnsAck(t *testing.T) {
	t.Parallel()

	message := canonicalkit.NewMessage(canonicalkit.MessageSpec{
		Producer: "inventory-worker",
		Kind:     canonicalkit.MessageKindEvent,
		Type:     "inventory.updated",
	}, json.RawMessage(`{"sku":"sku-1"}`))

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	result := HandleMessage(context.Background(), amqp.Delivery{Body: body, RoutingKey: "inventory.updated"}, func(ctx context.Context, delivery messagingkit.Delivery) messagingkit.Result {
		if delivery.Route.Key != "inventory.updated" {
			t.Fatalf("unexpected route key: %s", delivery.Route.Key)
		}
		return messagingkit.Ack()
	})
	if result.Outcome != messagingkit.OutcomeAck {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
}

