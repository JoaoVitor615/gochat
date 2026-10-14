package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	bolt "go.etcd.io/bbolt"
)

func TestBboltStore_LamportOrdering(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	peerID := "12D3KooWPeerA"

	// Message 1 has a LATER wall-clock timestamp (12:00:05), but LamportClock = 1
	// Message 2 has an EARLIER wall-clock timestamp (12:00:00), but LamportClock = 2
	tLater := time.Date(2026, 1, 1, 12, 0, 5, 0, time.UTC)
	tEarlier := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	msg1 := StoredMessage{
		ConversationPeerID: peerID,
		Envelope: message.Message{
			ID:              "msg-1",
			SenderPeerID:    "12D3KooWMe",
			RecipientPeerID: peerID,
			CreatedAt:       tLater,
			LamportClock:    1,
			Content:         "first by clock",
			ProtocolVersion: 2,
		},
		Status: message.DeliveryDelivered,
	}

	msg2 := StoredMessage{
		ConversationPeerID: peerID,
		Envelope: message.Message{
			ID:              "msg-2",
			SenderPeerID:    peerID,
			RecipientPeerID: "12D3KooWMe",
			CreatedAt:       tEarlier,
			LamportClock:    2,
			Content:         "second by clock",
			ProtocolVersion: 2,
		},
		Status: message.DeliveryDelivered,
	}

	if err := store.SaveMessage(ctx, msg1); err != nil {
		t.Fatalf("SaveMessage msg1: %v", err)
	}
	if err := store.SaveMessage(ctx, msg2); err != nil {
		t.Fatalf("SaveMessage msg2: %v", err)
	}

	messages, err := store.ListMessages(ctx, peerID, nil, 10)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	// msg1 should be listed before msg2 despite tLater > tEarlier!
	if messages[0].Envelope.ID != "msg-1" || messages[1].Envelope.ID != "msg-2" {
		t.Fatalf("expected [msg-1, msg-2], got [%s, %s]", messages[0].Envelope.ID, messages[1].Envelope.ID)
	}
}

func TestBboltStore_GetLatestLamportClock(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	peerID := "12D3KooWPeerB"

	// Initially 0
	clock, err := store.GetLatestLamportClock(ctx, peerID)
	if err != nil {
		t.Fatalf("GetLatestLamportClock empty: %v", err)
	}
	if clock != 0 {
		t.Fatalf("expected clock 0, got %d", clock)
	}

	// After queueing a message with LamportClock = 3
	msg := message.Message{
		ID:              "msg-3",
		SenderPeerID:    "12D3KooWMe",
		RecipientPeerID: peerID,
		CreatedAt:       time.Now().UTC(),
		LamportClock:    3,
		Content:         "test",
		ProtocolVersion: 2,
	}
	if err := store.QueueMessage(ctx, peerID, msg); err != nil {
		t.Fatalf("QueueMessage: %v", err)
	}

	clock, err = store.GetLatestLamportClock(ctx, peerID)
	if err != nil {
		t.Fatalf("GetLatestLamportClock after queue: %v", err)
	}
	if clock != 3 {
		t.Fatalf("expected clock 3, got %d", clock)
	}

	// After saving received message with LamportClock = 7
	msgReceived := StoredMessage{
		ConversationPeerID: peerID,
		Envelope: message.Message{
			ID:              "msg-7",
			SenderPeerID:    peerID,
			RecipientPeerID: "12D3KooWMe",
			CreatedAt:       time.Now().UTC(),
			LamportClock:    7,
			Content:         "reply",
			ProtocolVersion: 2,
		},
		Status: message.DeliveryDelivered,
	}
	if err := store.SaveMessage(ctx, msgReceived); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	clock, err = store.GetLatestLamportClock(ctx, peerID)
	if err != nil {
		t.Fatalf("GetLatestLamportClock after save: %v", err)
	}
	if clock != 7 {
		t.Fatalf("expected clock 7, got %d", clock)
	}
}

func TestBboltStore_Migration_V1_to_V2(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// 1. Manually construct schema 1 database
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("bolt.Open failed: %v", err)
	}
	peerID := "12D3KooWPeerC"
	msgID := "legacy-1"
	createdAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{peersBucket, messagesBucket, messageIDsBucket, outboxBucket, metadataBucket} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		var version [8]byte
		binary.BigEndian.PutUint64(version[:], 1)
		if err := tx.Bucket(metadataBucket).Put(schemaVersionKey, version[:]); err != nil {
			return err
		}

		// Insert message with legacy key: prefix + timestampBytes + 0 + id
		prefix := append([]byte(peerID), 0)
		legacyKey := append(bytes.Clone(prefix), timestampBytes(createdAt)...)
		legacyKey = append(legacyKey, 0)
		legacyKey = append(legacyKey, []byte(msgID)...)

		stored := StoredMessage{
			ConversationPeerID: peerID,
			Envelope: message.Message{
				ID:              msgID,
				SenderPeerID:    "12D3KooWMe",
				RecipientPeerID: peerID,
				CreatedAt:       createdAt,
				LamportClock:    0,
				Content:         "legacy message",
				ProtocolVersion: 2,
			},
			Status: message.DeliveryDelivered,
		}
		val, _ := json.Marshal(stored)
		if err := tx.Bucket(messagesBucket).Put(legacyKey, val); err != nil {
			return err
		}
		return tx.Bucket(messageIDsBucket).Put([]byte(msgID), legacyKey)
	})
	if err != nil {
		t.Fatalf("populate schema 1 failed: %v", err)
	}
	_ = db.Close()

	// 2. Open using store.Open which triggers migration to schema 2
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open with migration failed: %v", err)
	}
	defer store.Close()

	// 3. Verify message was migrated and is retrievable
	ctx := context.Background()
	retrieved, err := store.GetMessage(ctx, msgID)
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}
	if retrieved.Envelope.Content != "legacy message" {
		t.Fatalf("unexpected content: %s", retrieved.Envelope.Content)
	}

	messages, err := store.ListMessages(ctx, peerID, nil, 10)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(messages) != 1 || messages[0].Envelope.ID != msgID {
		t.Fatalf("expected 1 migrated message, got %d", len(messages))
	}
}
