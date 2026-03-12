package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/BHRK-codelabs/corekit/contextkit"
	"github.com/BHRK-codelabs/messagingkit"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RequestClient struct {
	channel   *amqp.Channel
	publisher *Publisher
	replyTo   string
	timeout   time.Duration

	consumeOnce sync.Once
	consumeErr  error

	mu      sync.Mutex
	pending map[string]chan amqp.Delivery
}

func NewRequestClient(channel *amqp.Channel, publisher *Publisher, timeout time.Duration) *RequestClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &RequestClient{
		channel:   channel,
		publisher: publisher,
		replyTo:   "amq.rabbitmq.reply-to",
		timeout:   timeout,
		pending:   make(map[string]chan amqp.Delivery),
	}
}

func (c *RequestClient) Call(ctx context.Context, route messagingkit.Route, delivery messagingkit.Delivery) (messagingkit.Delivery, error) {
	if c.channel == nil || c.publisher == nil {
		return messagingkit.Delivery{}, fmt.Errorf("request client is not configured")
	}

	correlationID, ok := contextkit.CorrelationID(ctx)
	if !ok || correlationID == "" {
		correlationID = delivery.Meta.MessageID
	}

	if err := c.ensureReplyConsumer(); err != nil {
		return messagingkit.Delivery{}, err
	}

	pub, err := BuildPublishing(ctx, messagingkit.RawMessage(delivery))
	if err != nil {
		return messagingkit.Delivery{}, err
	}
	pub.ReplyTo = c.replyTo
	pub.CorrelationId = correlationID

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	replyCh := make(chan amqp.Delivery, 1)
	c.registerPending(correlationID, replyCh)
	defer c.unregisterPending(correlationID)

	if err := c.channel.PublishWithContext(callCtx, c.publisher.exchange, route.Key, false, false, pub); err != nil {
		return messagingkit.Delivery{}, err
	}

	for {
		select {
		case <-callCtx.Done():
			return messagingkit.Delivery{}, callCtx.Err()
		case reply, ok := <-replyCh:
			if !ok {
				return messagingkit.Delivery{}, errors.New("reply channel closed")
			}
			return DeliveryFromAMQP(reply)
		}
	}
}

func (c *RequestClient) ensureReplyConsumer() error {
	c.consumeOnce.Do(func() {
		replyMessages, err := c.channel.Consume(
			c.replyTo,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			c.consumeErr = err
			return
		}

		go func() {
			for reply := range replyMessages {
				c.dispatchReply(reply)
			}
			c.closePending()
		}()
	})
	return c.consumeErr
}

func (c *RequestClient) registerPending(correlationID string, replyCh chan amqp.Delivery) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending[correlationID] = replyCh
}

func (c *RequestClient) unregisterPending(correlationID string) {
	c.mu.Lock()
	if _, ok := c.pending[correlationID]; ok {
		delete(c.pending, correlationID)
	}
	c.mu.Unlock()
}

func (c *RequestClient) dispatchReply(reply amqp.Delivery) {
	c.mu.Lock()
	replyCh := c.pending[reply.CorrelationId]
	c.mu.Unlock()
	if replyCh == nil {
		return
	}

	select {
	case replyCh <- reply:
	default:
	}
}

func (c *RequestClient) closePending() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for correlationID, replyCh := range c.pending {
		close(replyCh)
		delete(c.pending, correlationID)
	}
}

type RequestHandler func(context.Context, messagingkit.Delivery) (messagingkit.Delivery, messagingkit.Result)

func HandleRequest(ctx context.Context, channel *amqp.Channel, delivery amqp.Delivery, handler RequestHandler) messagingkit.Result {
	requestDelivery, err := DeliveryFromAMQP(delivery)
	if err != nil {
		return messagingkit.Discard(err)
	}

	response, result := handler(ContextFromDelivery(ctx, delivery), requestDelivery)
	if result.Outcome == "" {
		result.Outcome = messagingkit.OutcomeAck
	}
	if result.Error != nil {
		return result
	}

	if delivery.ReplyTo == "" {
		return result
	}

	pub, err := BuildPublishing(ctx, messagingkit.RawMessage(response))
	if err != nil {
		return messagingkit.Discard(err)
	}
	pub.CorrelationId = delivery.CorrelationId

	if err := channel.PublishWithContext(ctx, "", delivery.ReplyTo, false, false, pub); err != nil {
		return messagingkit.Discard(err)
	}
	return result
}
