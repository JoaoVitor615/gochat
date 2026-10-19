# GoChat — Release Notes v0.3.1

> **Release Date:** October 2026  
> **Status:** Stable  
> **Protocol Transition:** E2EE active by default, real-time presence protocol, and dynamic address updates.

---

## 📥 Downloads

Choose the appropriate package for your operating system:

| Platform | Architecture | Package Type | Download Link |
| :--- | :--- | :--- | :--- |
| **Windows** | `x86_64 / amd64` | **Graphical Installer (`.exe`)** | [⬇️ Download Windows Installer](https://github.com/JoaoVitor615/gochat/releases/download/v0.3.1/gochat-setup-windows-amd64.exe) |
| **Windows** | `x86_64 / amd64` | Portable (`.zip`) | [⬇️ gochat-v0.3.1-windows-amd64.zip](https://github.com/JoaoVitor615/gochat/releases/download/v0.3.1/gochat-v0.3.1-windows-amd64.zip) |
| **Windows** | `ARM64` | Portable (`.zip`) | [⬇️ gochat-v0.3.1-windows-arm64.zip](https://github.com/JoaoVitor615/gochat/releases/download/v0.3.1/gochat-v0.3.1-windows-arm64.zip) |
| **Linux** | `x86_64 / amd64` | Archive (`.tar.gz`) | [⬇️ gochat-v0.3.1-linux-amd64.tar.gz](https://github.com/JoaoVitor615/gochat/releases/download/v0.3.1/gochat-v0.3.1-linux-amd64.tar.gz) |
| **Linux** | `ARM64` | Archive (`.tar.gz`) | [⬇️ gochat-v0.3.1-linux-arm64.tar.gz](https://github.com/JoaoVitor615/gochat/releases/download/v0.3.1/gochat-v0.3.1-linux-arm64.tar.gz) |
| **macOS** | `Apple Silicon (M1/M2/M3/M4)` | Archive (`.tar.gz`) | [⬇️ gochat-v0.3.1-darwin-arm64.tar.gz](https://github.com/JoaoVitor615/gochat/releases/download/v0.3.1/gochat-v0.3.1-darwin-arm64.tar.gz) |
| **macOS** | `Intel 64-bit` | Archive (`.tar.gz`) | [⬇️ gochat-v0.3.1-darwin-amd64.tar.gz](https://github.com/JoaoVitor615/gochat/releases/download/v0.3.1/gochat-v0.3.1-darwin-amd64.tar.gz) |

---

## 🚀 Release Highlights — v0.3.1

Version **0.3.1** represents a fundamental leap in security, usability, and network resilience for **GoChat**. This update introduces End-to-End Encryption (E2EE), automatic presence detection with visual terminal indicators, intelligent dynamic address resolution via Discovery, an interactive splash screen initialization, and contact sorting inspired by modern messaging apps such as WhatsApp.

---

## ✨ Features & Enhancements

### 1. 🔒 End-to-End Encryption (E2EE)
All messages are now encrypted end-to-end between peers, ensuring that no intermediary can intercept or read conversation content.
- **Elliptic Curve Diffie-Hellman (X25519)**: Native conversion of Ed25519 identity keys to Curve25519 ([RFC 7748](https://datatracker.ietf.org/doc/html/rfc7748) / [RFC 8032](https://datatracker.ietf.org/doc/html/rfc8032)), allowing shared session secrets to be derived directly from PeerIDs.
- **AES-256-GCM Cipher with HKDF**: Symmetric session keys are derived using HKDF-SHA256, and message payloads are encrypted using AES-256-GCM with cryptographically secure per-message nonces.
- **Ed25519 Digital Signatures**: Every envelope carries an Ed25519 digital signature generated with the sender's private key. The recipient verifies integrity prior to accepting and decrypting, preventing tampering and impersonation attacks.
- **Local Privacy**: Incoming messages are decrypted only in memory upon delivery and stored securely in the local [bbolt](https://github.com/etcd-io/bbolt) database.

### 2. 🟢 Real-Time Presence & Status Indicators (Online / Offline)
- **Dedicated Presence Protocol (`/gochat/presence/1.0.0`)**: A lightweight P2P stream protocol for periodic activity signaling between contacts.
- **10-Second P2P Heartbeat Ping**: GoChat pings all saved contacts every 10 seconds (as well as 500ms after startup).
- **Terminal UI (TUI) Visual Status**:
  - **`● online`** in vibrant green (`#22C55E`) when the peer responded to a ping or sent a message within the last 30 seconds.
  - **`○ visto dd/mm hh:mm`** in subtle muted gray when the peer has been offline for over 30 seconds, showing the last confirmed timestamp.
- **Instant Reactivity**: The local event bus dispatches presence updates directly to the UI without blocking or polling overhead.

### 3. 🔄 Dynamic Address Resolution (Self-Healing IPs)
Previously, contacts only retained the network addresses discovered during the initial invite exchange. If a peer switched Wi-Fi networks or restarted the application, dialing attempts failed permanently.
- **New Discovery Endpoint (`GET /peer?peer_id=...`)**: Allows peers to query the active multiaddresses announced in the peer's latest Redis heartbeat.
- **Automatic Failure Recovery**:
  - When P2P dialing fails in the messaging outbox (`messaging.Service`), the client automatically queries Discovery for fresh addresses.
  - During the presence ping loop, if a peer fails to respond, GoChat resolves their latest IP, updates the libp2p peerstore and local database, re-establishes connectivity, and flushes queued outbox messages immediately.

### 4. 💬 WhatsApp-Style Contact Sorting by Recent Activity
- **Dynamic Ordering**: The contact sidebar (`PEERS`) is automatically sorted in descending order by the timestamp of the last message sent or received.
- **Real-Time Promotion**: When a new message is received or sent, that contact is immediately promoted to the top of the list.
- **Contacts Without Messages**: Placed below active conversations, sorted chronologically by addition date.
- **Focus & Selection Preservation**: The active chat and cursor selection remain locked onto the intended peer even as the list dynamically reorders.

### 5. ⏳ Interactive Splash Screen with Braille Spinner
- **Visual Startup Feedback**: When launching the terminal UI, users are greeted by an ASCII GoChat logo and an animated Braille spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`).
- **Non-Blocking Loading**: Libp2p initialization, reflective address discovery, and Discovery server synchronization execute concurrently in the background prior to rendering the main chat interface.

---

## 📁 Key Files & Architecture

| Component | Main Files | Responsibility |
|---|---|---|
| **Cryptography (E2EE)** | `internal/chat/crypto/crypto.go`<br>`internal/chat/crypto/crypto_test.go` | X25519 key exchange, HKDF, AES-256-GCM, and Ed25519 signatures |
| **P2P Protocol** | `internal/chat/p2p/presence.go`<br>`internal/chat/p2p/presence_test.go`<br>`internal/chat/p2p/stream.go` | `/gochat/presence/1.0.0` protocol and cryptographic stream verification |
| **Discovery Server** | `internal/discovery/service/service.go`<br>`internal/discovery/handler/handler.go`<br>`internal/discovery/server.go` | `GET /peer?peer_id=...` endpoint for peer address lookups |
| **Discovery Client** | `internal/chat/client/client.go`<br>`internal/chat/client/constants.go`<br>`internal/chat/client/domain.go` | Consuming `/peer` and response deserialization |
| **Messaging & Outbox** | `internal/chat/messaging/service.go`<br>`internal/chat/messaging/service_test.go` | Address resolution fallback upon delivery failure |
| **Terminal UI (TUI)** | `internal/chat/tui/tui.go`<br>`internal/chat/tui/tui_test.go`<br>`internal/chat/cli/presence.go` | Splash screen, WhatsApp-style sorting, and presence rendering |

---

## 🧪 Test Coverage

All new features are backed by comprehensive automated test suites:
- **E2EE End-to-End Tests**: Encryption, decryption, and rejection of tampered payloads.
- **P2P Presence Tests**: Ping/ack loop, online (<30s) / offline (>30s) state transitions.
- **Dynamic Resolution Tests**: Simulated stale addresses recovering via Discovery.
- **TUI Sorting Tests**: List reordering in response to incoming and outgoing messages.
- **Discovery Endpoint Tests**: 200 OK, 404 offline, and 400 validation error scenarios.

Run the test suite:
```bash
go test -count=1 ./...
```

---

## 📦 Build Instructions

To build the binaries for version 0.3.1:

```bash
# Build GoChat client
go build -o gochat ./cmd/gochat

# Build Discovery server
go build -o discovery ./cmd/discovery
```
