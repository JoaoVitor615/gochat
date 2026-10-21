package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
)

func TestCryptographicOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_file.dat")

	originalContent := bytes.Repeat([]byte("SENSITIVE_GOCHAT_DATA_TOP_SECRET_1234567890"), 500) // ~22 KB
	if err := os.WriteFile(filePath, originalContent, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Overwrite using AES-256 with random key
	if err := CryptographicOverwrite(filePath); err != nil {
		t.Fatalf("CryptographicOverwrite failed: %v", err)
	}

	// Verify file still exists and size is preserved
	stat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat file after overwrite: %v", err)
	}
	if stat.Size() != int64(len(originalContent)) {
		t.Fatalf("expected file size %d, got %d", len(originalContent), stat.Size())
	}

	// Read overwritten contents
	overwritten, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read overwritten file: %v", err)
	}

	// Verify original sensitive text is nowhere to be found
	if bytes.Contains(overwritten, []byte("SENSITIVE_GOCHAT_DATA")) {
		t.Fatal("overwritten file still contains sensitive plaintext data")
	}
	if bytes.Contains(overwritten, []byte("TOP_SECRET")) {
		t.Fatal("overwritten file still contains TOP_SECRET plaintext")
	}

	// Verify byte differences (should be almost 100% different)
	matching := 0
	for i := range originalContent {
		if originalContent[i] == overwritten[i] {
			matching++
		}
	}
	if matching > len(originalContent)/2 {
		t.Fatalf("ciphertext has abnormally high match with plaintext: %d/%d", matching, len(originalContent))
	}
}

func TestCryptographicErasureFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "to_delete.dat")

	if err := os.WriteFile(filePath, []byte("Some secret data"), 0600); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if err := CryptographicErasureFile(filePath); err != nil {
		t.Fatalf("CryptographicErasureFile failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, got err: %v", err)
	}

	// Calling again on deleted file should return nil
	if err := CryptographicErasureFile(filePath); err != nil {
		t.Fatalf("expected nil for non-existent file, got: %v", err)
	}
}

func TestBboltStore_CryptographicErasure(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "gochat_wipe_test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 1. Populate with a peer
	peerID := "12D3KooWTestPeerForWipe"
	if err := store.SavePeer(ctx, Peer{
		PeerID:      peerID,
		DisplayName: "Secret Alice",
		AddedAt:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SavePeer failed: %v", err)
	}

	// 2. Populate with messages
	msgID := "wipe-msg-1"
	if err := store.SaveMessage(ctx, StoredMessage{
		ConversationPeerID: peerID,
		Envelope: message.Message{
			ID:              msgID,
			SenderPeerID:    "12D3KooWLocalMe",
			RecipientPeerID: peerID,
			CreatedAt:       time.Now().UTC(),
			LamportClock:    1,
			Content:         "Top Secret Private Conversation",
		},
		Status: message.DeliveryDelivered,
	}); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// 3. Populate outbox (QueueMessage also records in messagesBucket with DeliveryPending)
	if err := store.QueueMessage(ctx, peerID, message.Message{
		ID:              "outbox-msg-1",
		SenderPeerID:    "12D3KooWLocalMe",
		RecipientPeerID: peerID,
		CreatedAt:       time.Now().UTC(),
		Content:         "Pending outbox confidential",
	}); err != nil {
		t.Fatalf("QueueMessage failed: %v", err)
	}

	// Verify data exists before wipe
	peers, err := store.ListPeers(ctx)
	if err != nil || len(peers) != 1 {
		t.Fatalf("expected 1 peer before wipe, got %d (err: %v)", len(peers), err)
	}

	msgs, err := store.ListMessages(ctx, peerID, nil, 10)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("expected 2 messages before wipe, got %d (err: %v)", len(msgs), err)
	}

	// 4. Perform Cryptographic Erasure
	if err := store.CryptographicErasure(); err != nil {
		t.Fatalf("CryptographicErasure failed: %v", err)
	}

	// 5. Verify the file is completely removed from the filesystem
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("expected database file %s to be deleted, got err: %v", dbPath, err)
	}

	// 6. Verify that re-opening the database initializes a fresh, empty database
	newStore, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open new database after erasure: %v", err)
	}
	defer newStore.Close()

	freshPeers, err := newStore.ListPeers(ctx)
	if err != nil {
		t.Fatalf("ListPeers on fresh db failed: %v", err)
	}
	if len(freshPeers) != 0 {
		t.Fatalf("expected 0 peers in fresh db, got %d", len(freshPeers))
	}

	freshMsgs, err := newStore.ListMessages(ctx, peerID, nil, 10)
	if err != nil {
		t.Fatalf("ListMessages on fresh db failed: %v", err)
	}
	if len(freshMsgs) != 0 {
		t.Fatalf("expected 0 messages in fresh db, got %d", len(freshMsgs))
	}

	outbox, err := newStore.ListOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("ListOutbox on fresh db failed: %v", err)
	}
	if len(outbox) != 0 {
		t.Fatalf("expected 0 outbox entries in fresh db, got %d", len(outbox))
	}
}

func TestCryptographicErasure_EdgeCases(t *testing.T) {
	// Empty path
	if err := CryptographicOverwrite(""); err == nil {
		t.Error("expected error for empty file path in CryptographicOverwrite")
	}

	// Nil store
	var nilStore *BboltStore
	if err := nilStore.CryptographicErasure(); err == nil {
		t.Error("expected error for nil store in CryptographicErasure")
	}

	// 0-byte file
	tempDir := t.TempDir()
	emptyFile := filepath.Join(tempDir, "empty.dat")
	if err := os.WriteFile(emptyFile, []byte{}, 0600); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	if err := CryptographicErasureFile(emptyFile); err != nil {
		t.Fatalf("CryptographicErasureFile on empty file failed: %v", err)
	}
	if _, err := os.Stat(emptyFile); !os.IsNotExist(err) {
		t.Errorf("expected empty file to be deleted")
	}
}
