package events

import (
	"sync"

	"github.com/JoaoVitor615/gochat/internal/chat/storage"
)

// Bus provides a lightweight, thread-safe pub/sub mechanism for chat message events.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[chan storage.StoredMessage]struct{}
}

// NewBus creates an initialized Bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[chan storage.StoredMessage]struct{}),
	}
}

// Subscribe registers a new subscriber channel with a buffer.
func (b *Bus) Subscribe() chan storage.StoredMessage {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan storage.StoredMessage, 64)
	b.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes and closes the subscriber channel.
func (b *Bus) Unsubscribe(ch chan storage.StoredMessage) {
	if b == nil || ch == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.subscribers[ch]; exists {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Publish distributes the message to all active subscribers without blocking.
func (b *Bus) Publish(msg storage.StoredMessage) {
	if b == nil {
		return
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
			// Buffer full, drop event to prevent blocking publishers
		}
	}
}
