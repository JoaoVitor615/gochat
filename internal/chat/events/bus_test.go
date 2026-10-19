package events

import (
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
)

func TestBus_PublishAndSubscribe(t *testing.T) {
	bus := NewBus()

	sub := bus.Subscribe()
	defer bus.Unsubscribe(sub)

	msg := storage.StoredMessage{
		ConversationPeerID: "peer-1",
		Envelope: message.Message{
			ID:      "test-id",
			Content: "hello",
		},
		Status: message.DeliveryDelivered,
	}

	bus.Publish(msg)

	select {
	case received := <-sub:
		if received.Envelope.ID != "test-id" {
			t.Fatalf("expected test-id, got %s", received.Envelope.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	// Test Unsubscribe
	bus.Unsubscribe(sub)
	_, ok := <-sub
	if ok {
		t.Fatal("expected subscriber channel to be closed")
	}

	// Publishing after unsubscribe should not panic
	bus.Publish(msg)
}

func TestBus_NilSafety(t *testing.T) {
	var bus *Bus

	// None of these should panic
	if ch := bus.Subscribe(); ch != nil {
		t.Fatal("expected nil channel from nil bus")
	}
	if pch := bus.SubscribePresence(); pch != nil {
		t.Fatal("expected nil channel from nil bus")
	}
	bus.Unsubscribe(nil)
	bus.UnsubscribePresence(nil)
	bus.Publish(storage.StoredMessage{})
	bus.PublishPresence(PeerPresence{})
}

func TestBus_PresencePublishAndSubscribe(t *testing.T) {
	bus := NewBus()

	sub := bus.SubscribePresence()
	defer bus.UnsubscribePresence(sub)

	now := time.Now().UTC()
	event := PeerPresence{
		PeerID:     "peer-xyz",
		LastSeenAt: now,
	}

	bus.PublishPresence(event)

	select {
	case received := <-sub:
		if received.PeerID != "peer-xyz" {
			t.Fatalf("expected peer-xyz, got %s", received.PeerID)
		}
		if !received.LastSeenAt.Equal(now) {
			t.Fatalf("expected %v, got %v", now, received.LastSeenAt)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for presence event")
	}

	// Test UnsubscribePresence
	bus.UnsubscribePresence(sub)
	_, ok := <-sub
	if ok {
		t.Fatal("expected presence subscriber channel to be closed")
	}

	// Publishing after unsubscribe should not panic
	bus.PublishPresence(event)
}

