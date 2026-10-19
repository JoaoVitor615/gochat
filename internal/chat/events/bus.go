package events

import (
	"sync"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/storage"
)

// PeerPresence contains online presence activity for a peer.
type PeerPresence struct {
	PeerID     string
	LastSeenAt time.Time
}

// Bus provides a lightweight, thread-safe pub/sub mechanism for chat message and presence events.
type Bus struct {
	mu                  sync.RWMutex
	subscribers         map[chan storage.StoredMessage]struct{}
	presenceSubscribers map[chan PeerPresence]struct{}
}

// NewBus creates an initialized Bus.
func NewBus() *Bus {
	return &Bus{
		subscribers:         make(map[chan storage.StoredMessage]struct{}),
		presenceSubscribers: make(map[chan PeerPresence]struct{}),
	}
}

// Subscribe registers a new subscriber channel with a buffer for message events.
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

// Unsubscribe removes and closes the message subscriber channel.
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

// Publish distributes the message to all active message subscribers without blocking.
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

// SubscribePresence registers a new subscriber channel with a buffer for peer presence events.
func (b *Bus) SubscribePresence() chan PeerPresence {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan PeerPresence, 64)
	b.presenceSubscribers[ch] = struct{}{}
	return ch
}

// UnsubscribePresence removes and closes the presence subscriber channel.
func (b *Bus) UnsubscribePresence(ch chan PeerPresence) {
	if b == nil || ch == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.presenceSubscribers[ch]; exists {
		delete(b.presenceSubscribers, ch)
		close(ch)
	}
}

// PublishPresence distributes peer presence events to all active presence subscribers without blocking.
func (b *Bus) PublishPresence(event PeerPresence) {
	if b == nil {
		return
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.presenceSubscribers {
		select {
		case ch <- event:
		default:
			// Buffer full, drop event to prevent blocking publishers
		}
	}
}
