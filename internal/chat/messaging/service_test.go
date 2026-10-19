package messaging_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/crypto"
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/JoaoVitor615/gochat/internal/chat/messaging"
	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func createTestPeer(t *testing.T, dir string) (*identity.Identity, storage.Repository) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	libPriv, _, err := libp2pcrypto.KeyPairFromStdKey(&priv)
	if err != nil {
		t.Fatalf("convert key for libp2p: %v", err)
	}
	pID, err := peer.IDFromPrivateKey(libPriv)
	if err != nil {
		t.Fatalf("derive peer ID: %v", err)
	}

	id := &identity.Identity{
		PrivateKey: priv,
		PublicKey:  pub,
		PeerID:     pID.String(),
	}

	dbPath := filepath.Join(dir, pID.String()+".db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}

	return id, store
}

func TestMessaging_E2EEEndToEnd(t *testing.T) {
	tempDir := t.TempDir()

	aliceID, aliceStore := createTestPeer(t, tempDir)
	defer aliceStore.Close()

	bobID, bobStore := createTestPeer(t, tempDir)
	defer bobStore.Close()

	aliceHost, err := p2p.NewHost(aliceID, 0)
	if err != nil {
		t.Fatalf("start alice host: %v", err)
	}
	defer aliceHost.Close()

	bobHost, err := p2p.NewHost(bobID, 0)
	if err != nil {
		t.Fatalf("start bob host: %v", err)
	}
	defer bobHost.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bobReceived := make(chan storage.StoredMessage, 1)

	// Bob listens for incoming streams with his private key
	p2p.SetStreamHandler(bobHost, func(stream network.Stream) {
		err := p2p.HandleStream(ctx, stream, bobStore, bobID.PeerID, func(sm storage.StoredMessage) {
			bobReceived <- sm
		}, bobID.PrivateKey)
		if err != nil {
			t.Errorf("bob handle stream error: %v", err)
		}
	})

	// Alice sets up messaging service with her private key
	aliceMessenger := messaging.New(aliceHost, aliceID.PeerID, aliceStore, nil, aliceID.PrivateKey)

	// Register Bob's addresses with Alice
	bobAddrsInfo, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    bobHost.ID(),
		Addrs: bobHost.Addrs(),
	})
	if err != nil {
		t.Fatalf("bob addr info to p2p addrs: %v", err)
	}
	bobAddrs := make([]string, len(bobAddrsInfo))
	for i, addr := range bobAddrsInfo {
		bobAddrs[i] = addr.String()
	}
	if err := aliceMessenger.AddPeerAddresses(bobID.PeerID, bobAddrs); err != nil {
		t.Fatalf("add bob addresses: %v", err)
	}

	secretContent := "Hello Bob, this is a secret E2EE message!"
	res, err := aliceMessenger.Send(ctx, bobID.PeerID, secretContent)
	if err != nil {
		t.Fatalf("alice send failed: %v", err)
	}
	if res.Status != message.DeliveryDelivered {
		t.Fatalf("expected status %s, got %s (err: %v)", message.DeliveryDelivered, res.Status, res.DeliveryError)
	}

	// Verify Bob received the message
	select {
	case received := <-bobReceived:
		if received.Envelope.Content != secretContent {
			t.Errorf("bob received content %q, want %q", received.Envelope.Content, secretContent)
		}
		if received.Envelope.Encrypted {
			t.Errorf("stored message should have Encrypted: false after decryption")
		}
		if received.Envelope.Signature == "" {
			t.Errorf("expected signature to be present on envelope")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for bob to receive message")
	}

	// Verify Alice's store has plaintext
	storedAlice, err := aliceStore.GetMessage(ctx, res.Message.ID)
	if err != nil {
		t.Fatalf("get alice message: %v", err)
	}
	if storedAlice.Envelope.Content != secretContent {
		t.Errorf("alice store has content %q, want %q", storedAlice.Envelope.Content, secretContent)
	}
}

func TestMessaging_TamperedMessageRejected(t *testing.T) {
	tempDir := t.TempDir()

	aliceID, aliceStore := createTestPeer(t, tempDir)
	defer aliceStore.Close()

	bobID, bobStore := createTestPeer(t, tempDir)
	defer bobStore.Close()

	aliceHost, err := p2p.NewHost(aliceID, 0)
	if err != nil {
		t.Fatalf("start alice host: %v", err)
	}
	defer aliceHost.Close()

	bobHost, err := p2p.NewHost(bobID, 0)
	if err != nil {
		t.Fatalf("start bob host: %v", err)
	}
	defer bobHost.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bobReceived := make(chan storage.StoredMessage, 1)

	p2p.SetStreamHandler(bobHost, func(stream network.Stream) {
		_ = p2p.HandleStream(ctx, stream, bobStore, bobID.PeerID, func(sm storage.StoredMessage) {
			bobReceived <- sm
		}, bobID.PrivateKey)
	})

	bobDecodedID, err := peer.Decode(bobID.PeerID)
	if err != nil {
		t.Fatalf("decode bob peer id: %v", err)
	}

	// Connect Alice to Bob
	bobAddrsInfo, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    bobHost.ID(),
		Addrs: bobHost.Addrs(),
	})
	if err != nil {
		t.Fatalf("bob addr info to p2p addrs: %v", err)
	}
	addrInfo, _ := peer.AddrInfoFromP2pAddr(bobAddrsInfo[0])
	if err := aliceHost.Connect(ctx, *addrInfo); err != nil {
		t.Fatalf("connect alice to bob: %v", err)
	}

	// Encrypt a message
	encryptedContent, err := crypto.Encrypt([]byte("Hello Bob!"), bobID.PeerID)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	tamperedMsg := message.Message{
		ID:              "tampered-msg-id-1",
		SenderPeerID:    aliceID.PeerID,
		RecipientPeerID: bobID.PeerID,
		CreatedAt:       time.Now().UTC(),
		LamportClock:    1,
		Content:         encryptedContent,
		Encrypted:       true,
		ProtocolVersion: message.CurrentProtocolVersion,
	}
	// Sign the original
	tamperedMsg.Signature = crypto.SignMessage(aliceID.PrivateKey, tamperedMsg)

	// Tamper with the content after signing
	tamperedMsg.Content = encryptedContent[:len(encryptedContent)-4] + "AAAA"

	_, err = p2p.SendMessageAndWaitForAck(ctx, aliceHost, bobDecodedID, tamperedMsg)
	// Bob should reject it, and not send ACK
	if err == nil {
		t.Fatal("expected SendMessageAndWaitForAck to fail due to rejected tampered message")
	}

	// Verify Bob never stored or notified the message
	select {
	case <-bobReceived:
		t.Fatal("bob should not have accepted tampered message")
	case <-time.After(500 * time.Millisecond):
		// Expected timeout
	}

	_, err = bobStore.GetMessage(ctx, tamperedMsg.ID)
	if err == nil {
		t.Fatal("bob store should not contain tampered message")
	}
}

func TestMessaging_DynamicAddressResolution(t *testing.T) {
	tempDir := t.TempDir()

	aliceID, aliceStore := createTestPeer(t, tempDir)
	defer aliceStore.Close()

	bobID, bobStore := createTestPeer(t, tempDir)
	defer bobStore.Close()

	aliceHost, err := p2p.NewHost(aliceID, 0)
	if err != nil {
		t.Fatalf("start alice host: %v", err)
	}
	defer aliceHost.Close()

	bobHost, err := p2p.NewHost(bobID, 0)
	if err != nil {
		t.Fatalf("start bob host: %v", err)
	}
	defer bobHost.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bobReceived := make(chan storage.StoredMessage, 1)
	p2p.SetStreamHandler(bobHost, func(stream network.Stream) {
		_ = p2p.HandleStream(ctx, stream, bobStore, bobID.PeerID, func(sm storage.StoredMessage) {
			bobReceived <- sm
		}, bobID.PrivateKey)
	})

	aliceMessenger := messaging.New(aliceHost, aliceID.PeerID, aliceStore, nil, aliceID.PrivateKey)

	// Bob's actual addresses
	bobAddrsInfo, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    bobHost.ID(),
		Addrs: bobHost.Addrs(),
	})
	if err != nil {
		t.Fatalf("bob addr info to p2p addrs: %v", err)
	}
	bobAddrs := make([]string, len(bobAddrsInfo))
	for i, addr := range bobAddrsInfo {
		bobAddrs[i] = addr.String()
	}

	// Register a dummy/invalid address for Bob initially
	dummyAddr := "/ip4/127.0.0.1/tcp/59999/p2p/" + bobID.PeerID
	_ = aliceMessenger.AddPeerAddresses(bobID.PeerID, []string{dummyAddr})

	resolverCalled := false
	updatedPeerID := ""
	var updatedAddrs []string

	aliceMessenger.SetResolver(func(ctx context.Context, peerID string) ([]string, error) {
		resolverCalled = true
		if peerID == bobID.PeerID {
			return bobAddrs, nil
		}
		return nil, nil
	}, func(peerID string, addrs []string) {
		updatedPeerID = peerID
		updatedAddrs = addrs
	})

	result, err := aliceMessenger.Send(ctx, bobID.PeerID, "Hello via resolved address!")
	if err != nil {
		t.Fatalf("send message failed: %v", err)
	}
	if result.DeliveryError != nil {
		t.Fatalf("delivery error: %v", result.DeliveryError)
	}
	if result.Status != message.DeliveryDelivered {
		t.Fatalf("expected status delivered, got %s", result.Status)
	}
	if !resolverCalled {
		t.Fatal("expected resolver to be called upon dial failure")
	}
	if updatedPeerID != bobID.PeerID || len(updatedAddrs) == 0 {
		t.Fatalf("expected onUpdated callback with bobID and addrs, got peer %s, addrs %v", updatedPeerID, updatedAddrs)
	}

	select {
	case msg := <-bobReceived:
		if msg.Envelope.Content != "Hello via resolved address!" {
			t.Fatalf("expected content 'Hello via resolved address!', got %q", msg.Envelope.Content)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for bob to receive message")
	}
}


