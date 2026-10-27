package kafka

import (
	"context"
	"errors"
	"testing"

	"rent/notifier/internal/entity"
	repository "rent/notifier/internal/repository/notification"

	kafkago "github.com/segmentio/kafka-go"
)

type testReader struct{ commits int }

func (r *testReader) FetchMessage(context.Context) (kafkago.Message, error) {
	return kafkago.Message{}, errors.New("unused")
}
func (r *testReader) Close() error { return nil }
func (r *testReader) CommitMessages(context.Context, ...kafkago.Message) error {
	r.commits++
	if r.commits == 1 {
		return errors.New("commit response lost")
	}
	return nil
}

type testInbox struct {
	fail bool
	put  func(context.Context, entity.Notification) error
}

func (i *testInbox) Notify(ctx context.Context, n entity.Notification) error {
	if i.fail {
		i.fail = false
		return errors.New("database unavailable")
	}
	return i.put(ctx, n)
}

func TestCommitAfterSaveAndSafeRedelivery(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemory()
	reader := &testReader{}
	inbox := &testInbox{fail: true, put: store.Put}
	consumer := &consumer{reader: reader, notifications: inbox}
	message := kafkago.Message{Value: []byte(`{"id":"event","user_id":"user","order_id":"order","state":"confirmed"}`)}
	if err := consumer.process(ctx, message); err == nil || reader.commits != 0 {
		t.Fatal("offset committed before persistence")
	}
	if err := consumer.process(ctx, message); err == nil {
		t.Fatal("commit failure ignored")
	}
	if err := consumer.process(ctx, message); err != nil {
		t.Fatal(err)
	}
	count, err := store.PendingCount(ctx)
	if err != nil || count != 1 || reader.commits != 2 {
		t.Fatalf("redelivery: pending=%d commits=%d err=%v", count, reader.commits, err)
	}
}
