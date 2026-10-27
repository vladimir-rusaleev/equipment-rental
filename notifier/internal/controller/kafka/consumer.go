package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"rent/internal/eventbus"
	"rent/internal/metrics"
	"rent/notifier/internal/entity"

	kafkago "github.com/segmentio/kafka-go"
)

type messageReader interface {
	FetchMessage(context.Context) (kafkago.Message, error)
	CommitMessages(context.Context, ...kafkago.Message) error
	Close() error
}
type notificationService interface {
	Notify(context.Context, entity.Notification) error
}
type consumer struct {
	reader        messageReader
	notifications notificationService
}

func New(config eventbus.Config, notifications notificationService) *consumer {
	return &consumer{notifications: notifications, reader: kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: config.Brokers, Topic: config.Topic, GroupID: config.Group,
		StartOffset: kafkago.FirstOffset, CommitInterval: 0,
		MinBytes: 1, MaxBytes: 1 << 20, QueueCapacity: 10, MaxWait: 500 * time.Millisecond,
		Dialer: &kafkago.Dialer{Timeout: 3 * time.Second},
	})}
}
func (c *consumer) Close() error { return c.reader.Close() }

func (c *consumer) Run(ctx context.Context) {
	for ctx.Err() == nil {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("fetch Kafka event", "error", err)
			if !retry(ctx) {
				return
			}
			continue
		}
		// Retry this message before fetching the next offset.
		for ctx.Err() == nil {
			attempt, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = c.process(attempt, message)
			cancel()
			metrics.NotificationAttempt("kafka_consume", err)
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				return
			}
			slog.Warn("process Kafka event", "partition", message.Partition, "offset", message.Offset, "error", err)
			if !retry(ctx) {
				return
			}
		}
	}
}

func (c *consumer) process(ctx context.Context, message kafkago.Message) error {
	var event eventbus.OrderEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("decode event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}
	if err := c.notifications.Notify(ctx, entity.Notification{ID: event.ID, UserID: event.UserID, OrderID: event.OrderID, State: event.State}); err != nil {
		return err
	}
	return c.reader.CommitMessages(ctx, message)
}

func retry(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
