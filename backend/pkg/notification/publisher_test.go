package notification

import (
	"context"
	"testing"
)

func TestPublisherFunc(t *testing.T) {
	wantID := int64(42)
	publisher := PublisherFunc(func(_ context.Context, message Message) (int64, error) {
		if message.Title != "title" {
			t.Fatalf("unexpected title: %s", message.Title)
		}
		return wantID, nil
	})
	id, err := publisher.Publish(context.Background(), Message{Title: "title"})
	if err != nil {
		t.Fatal(err)
	}
	if id != wantID {
		t.Fatalf("PublisherFunc returned %d, want %d", id, wantID)
	}
}

func TestPublishWithoutDefaultPublisher(t *testing.T) {
	defaultPublisher.Store(nil)
	if _, err := Publish(context.Background(), Message{}); err == nil {
		t.Fatal("expected an error when the default publisher is not initialized")
	}
}
