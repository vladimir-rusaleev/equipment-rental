package kafka

import (
	"context"
	"encoding/json"
	"time"

	"rent/internal/eventbus"
	"rent/rental/internal/entity"

	kafkago "github.com/segmentio/kafka-go"
)

type client struct{ writer *kafkago.Writer }

func New(config eventbus.Config) *client {
	return &client{writer: &kafkago.Writer{
		Addr:         kafkago.TCP(config.Brokers...),
		Topic:        config.Topic,
		Balancer:     &kafkago.Hash{},
		RequiredAcks: kafkago.RequireAll,
		Async:        false,
		BatchSize:    1,
		MaxAttempts:  1,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}}
}
func (c *client) Close() error { return c.writer.Close() }

func (c *client) Notify(ctx context.Context, n entity.Notification) error {
	event := eventbus.OrderEvent{ID: n.ID, UserID: n.UserID, OrderID: n.OrderID, State: n.State}
	if err := event.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.writer.WriteMessages(ctx, kafkago.Message{Key: []byte(n.OrderID), Value: data})
}
