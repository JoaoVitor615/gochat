package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/JoaoVitor615/gochat/internal/chat"
	"github.com/JoaoVitor615/gochat/internal/chat/message"
)

const (
	databaseFileName  = "gochat.db"
	currentSchema     = uint64(1)
	defaultPageSize   = 50
	defaultOutboxSize = 100
)

var (
	peersBucket      = []byte("peers")
	messagesBucket   = []byte("messages")
	messageIDsBucket = []byte("message_ids")
	outboxBucket     = []byte("outbox")
	metadataBucket   = []byte("metadata")
	schemaVersionKey = []byte("schema_version")
)

// BboltStore implements Repository using a single local bbolt file.
type BboltStore struct {
	db        *bolt.DB
	closeOnce sync.Once
	closeErr  error
}

// OpenDefault opens ~/.gochat/gochat.db, alongside the local identity file.
func OpenDefault() (*BboltStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home directory: %w", err)
	}
	dataDir := filepath.Join(home, chat.DataDirectory)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("create application data directory: %w", err)
	}
	if err := os.Chmod(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("restrict application data directory permissions: %w", err)
	}
	return Open(filepath.Join(dataDir, databaseFileName))
}

// Open creates or opens a database file and initializes the schema buckets.
func Open(path string) (*BboltStore, error) {
	if path == "" {
		return nil, fmt.Errorf("open local database: %w", ErrInvalidRecord)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bbolt database: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("restrict database file permissions: %w", err)
	}
	store := &BboltStore{db: db}
	if err := store.initializeSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *BboltStore) initializeSchema() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{peersBucket, messagesBucket, messageIDsBucket, outboxBucket, metadataBucket} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("create bucket %q: %w", name, err)
			}
		}
		metadata := tx.Bucket(metadataBucket)
		storedVersion := metadata.Get(schemaVersionKey)
		if storedVersion == nil {
			var version [8]byte
			binary.BigEndian.PutUint64(version[:], currentSchema)
			return metadata.Put(schemaVersionKey, version[:])
		}
		if len(storedVersion) != 8 {
			return fmt.Errorf("invalid local database schema version")
		}
		version := binary.BigEndian.Uint64(storedVersion)
		if version > currentSchema {
			return fmt.Errorf("database schema %d is newer than supported schema %d", version, currentSchema)
		}
		if version < currentSchema {
			// Future schema changes should add explicit migrations here.
			return fmt.Errorf("database schema %d requires a migration to schema %d", version, currentSchema)
		}
		return nil
	})
}

func (s *BboltStore) Close() error {
	s.closeOnce.Do(func() {
		if s.db != nil {
			s.closeErr = s.db.Close()
		}
	})
	return s.closeErr
}

func (s *BboltStore) SavePeer(ctx context.Context, peer Peer) error {
	if peer.PeerID == "" {
		return fmt.Errorf("save peer: %w", ErrInvalidRecord)
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	if peer.AddedAt.IsZero() {
		peer.AddedAt = time.Now().UTC()
	} else {
		peer.AddedAt = peer.AddedAt.UTC()
	}
	if !peer.LastSeenAt.IsZero() {
		peer.LastSeenAt = peer.LastSeenAt.UTC()
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := checkContext(ctx); err != nil {
			return err
		}
		bucket := tx.Bucket(peersBucket)
		if existing := bucket.Get([]byte(peer.PeerID)); existing != nil {
			var saved Peer
			if err := json.Unmarshal(existing, &saved); err != nil {
				return fmt.Errorf("decode existing peer: %w", err)
			}
			peer.AddedAt = saved.AddedAt
			if peer.DisplayName == "" {
				peer.DisplayName = saved.DisplayName
			}
			if peer.LastSeenAt.IsZero() {
				peer.LastSeenAt = saved.LastSeenAt
			}
			if peer.Addresses == nil {
				peer.Addresses = saved.Addresses
			}
		}
		value, err := json.Marshal(peer)
		if err != nil {
			return fmt.Errorf("encode peer: %w", err)
		}
		return bucket.Put([]byte(peer.PeerID), value)
	})
}

func (s *BboltStore) GetPeer(ctx context.Context, peerID string) (Peer, error) {
	var peer Peer
	if err := checkContext(ctx); err != nil {
		return peer, err
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket(peersBucket).Get([]byte(peerID))
		if value == nil {
			return ErrNotFound
		}
		return json.Unmarshal(value, &peer)
	})
	if err != nil {
		return Peer{}, fmt.Errorf("get peer: %w", err)
	}
	return peer, nil
}

func (s *BboltStore) ListPeers(ctx context.Context) ([]Peer, error) {
	peers := make([]Peer, 0)
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(peersBucket).ForEach(func(_, value []byte) error {
			if err := checkContext(ctx); err != nil {
				return err
			}
			var peer Peer
			if err := json.Unmarshal(value, &peer); err != nil {
				return fmt.Errorf("decode peer: %w", err)
			}
			peers = append(peers, peer)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list peers: %w", err)
	}
	return peers, nil
}

func (s *BboltStore) DeletePeer(ctx context.Context, peerID string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := checkContext(ctx); err != nil {
			return err
		}
		bucket := tx.Bucket(peersBucket)
		if bucket.Get([]byte(peerID)) == nil {
			return ErrNotFound
		}
		// Message history is retained independently of the contacts list.
		return bucket.Delete([]byte(peerID))
	})
}

func (s *BboltStore) SaveMessage(ctx context.Context, stored StoredMessage) error {
	if err := validateStoredMessage(stored); err != nil {
		return err
	}
	stored.Envelope.CreatedAt = stored.Envelope.CreatedAt.UTC()
	if err := checkContext(ctx); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := checkContext(ctx); err != nil {
			return err
		}
		return saveMessage(tx, stored)
	})
}

func (s *BboltStore) GetMessage(ctx context.Context, messageID string) (StoredMessage, error) {
	var stored StoredMessage
	if err := checkContext(ctx); err != nil {
		return stored, err
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		key := tx.Bucket(messageIDsBucket).Get([]byte(messageID))
		if key == nil {
			return ErrNotFound
		}
		value := tx.Bucket(messagesBucket).Get(key)
		if value == nil {
			return ErrNotFound
		}
		return json.Unmarshal(value, &stored)
	})
	if err != nil {
		return StoredMessage{}, fmt.Errorf("get message: %w", err)
	}
	return stored, nil
}

// ListMessages returns the newest page strictly before before, in chronological order.
// A zero before value starts at the newest message. A non-positive limit uses 50.
func (s *BboltStore) ListMessages(ctx context.Context, peerID string, before time.Time, limit int) ([]StoredMessage, error) {
	if peerID == "" {
		return nil, fmt.Errorf("list messages: %w", ErrInvalidRecord)
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	messages := make([]StoredMessage, 0, limit)
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(messagesBucket)
		cursor := bucket.Cursor()
		prefix := conversationPrefix(peerID)
		var key, value []byte
		if before.IsZero() {
			// The successor of the prefix sorts after every key in this conversation.
			upper := append(bytes.Clone(prefix[:len(prefix)-1]), prefix[len(prefix)-1]+1)
			key, value = cursor.Seek(upper)
			if key == nil {
				key, value = cursor.Last()
			} else {
				key, value = cursor.Prev()
			}
		} else {
			upper := append(bytes.Clone(prefix), timestampBytes(before)...)
			key, value = cursor.Seek(upper)
			if key == nil {
				key, value = cursor.Last()
			} else {
				key, value = cursor.Prev()
			}
		}
		for key != nil && bytes.HasPrefix(key, prefix) && len(messages) < limit {
			if err := checkContext(ctx); err != nil {
				return err
			}
			var stored StoredMessage
			if err := json.Unmarshal(value, &stored); err != nil {
				return fmt.Errorf("decode message: %w", err)
			}
			messages = append(messages, stored)
			key, value = cursor.Prev()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}

func (s *BboltStore) UpdateMessageStatus(ctx context.Context, messageID string, status message.DeliveryStatus) error {
	if !validStatus(status) {
		return fmt.Errorf("update message status: %w", ErrInvalidRecord)
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := checkContext(ctx); err != nil {
			return err
		}
		ids := tx.Bucket(messageIDsBucket)
		key := ids.Get([]byte(messageID))
		if key == nil {
			return ErrNotFound
		}
		messages := tx.Bucket(messagesBucket)
		value := messages.Get(key)
		if value == nil {
			return ErrNotFound
		}
		var stored StoredMessage
		if err := json.Unmarshal(value, &stored); err != nil {
			return fmt.Errorf("decode message: %w", err)
		}
		stored.Status = status
		updated, err := json.Marshal(stored)
		if err != nil {
			return fmt.Errorf("encode message: %w", err)
		}
		if err := messages.Put(key, updated); err != nil {
			return err
		}
		if status == message.DeliveryDelivered {
			return tx.Bucket(outboxBucket).Delete(outboxKey(stored.Envelope))
		}
		return nil
	})
}

func (s *BboltStore) QueueMessage(ctx context.Context, conversationPeerID string, envelope message.Message) error {
	stored := StoredMessage{ConversationPeerID: conversationPeerID, Envelope: envelope, Status: message.DeliveryPending}
	if err := validateStoredMessage(stored); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := checkContext(ctx); err != nil {
			return err
		}
		if err := saveMessage(tx, stored); err != nil {
			return err
		}
		entry := OutboxEntry{ConversationPeerID: conversationPeerID, Message: envelope}
		value, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("encode outbox entry: %w", err)
		}
		return tx.Bucket(outboxBucket).Put(outboxKey(envelope), value)
	})
}

func (s *BboltStore) ListOutbox(ctx context.Context, limit int) ([]OutboxEntry, error) {
	if limit <= 0 {
		limit = defaultOutboxSize
	}
	entries := make([]OutboxEntry, 0, limit)
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		cursor := tx.Bucket(outboxBucket).Cursor()
		for _, value := cursor.First(); value != nil && len(entries) < limit; _, value = cursor.Next() {
			if err := checkContext(ctx); err != nil {
				return err
			}
			var entry OutboxEntry
			if err := json.Unmarshal(value, &entry); err != nil {
				return fmt.Errorf("decode outbox entry: %w", err)
			}
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list outbox: %w", err)
	}
	return entries, nil
}

func saveMessage(tx *bolt.Tx, stored StoredMessage) error {
	messages := tx.Bucket(messagesBucket)
	ids := tx.Bucket(messageIDsBucket)
	id := []byte(stored.Envelope.ID)
	key := messageKey(stored)
	if previous := ids.Get(id); previous != nil {
		var old StoredMessage
		if err := json.Unmarshal(messages.Get(previous), &old); err != nil {
			return fmt.Errorf("decode existing message: %w", err)
		}
		if old.ConversationPeerID != stored.ConversationPeerID {
			return ErrMessageIDExists
		}
		if !bytes.Equal(previous, key) {
			if err := messages.Delete(previous); err != nil {
				return err
			}
		}
	}
	value, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}
	if err := messages.Put(key, value); err != nil {
		return err
	}
	return ids.Put(id, key)
}

func validateStoredMessage(stored StoredMessage) error {
	if stored.ConversationPeerID == "" || stored.Envelope.ID == "" || stored.Envelope.CreatedAt.IsZero() || !validStatus(stored.Status) {
		return fmt.Errorf("save message: %w", ErrInvalidRecord)
	}
	return nil
}

func validStatus(status message.DeliveryStatus) bool {
	switch status {
	case message.DeliveryPending, message.DeliverySent, message.DeliveryDelivered:
		return true
	default:
		return false
	}
}

func messageKey(stored StoredMessage) []byte {
	key := conversationPrefix(stored.ConversationPeerID)
	key = append(key, timestampBytes(stored.Envelope.CreatedAt)...)
	key = append(key, 0)
	return append(key, stored.Envelope.ID...)
}

func conversationPrefix(peerID string) []byte {
	return append([]byte(peerID), 0)
}

func timestampBytes(timestamp time.Time) []byte {
	ordered := uint64(timestamp.UTC().UnixNano()) ^ (uint64(1) << 63)
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], ordered)
	return encoded[:]
}

func outboxKey(envelope message.Message) []byte {
	key := timestampBytes(envelope.CreatedAt)
	key = append(key, 0)
	return append(key, envelope.ID...)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("storage: context is nil")
	}
	return ctx.Err()
}

var _ Repository = (*BboltStore)(nil)
