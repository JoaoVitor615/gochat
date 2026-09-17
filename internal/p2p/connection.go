package p2p

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/JoaoVitor615/gochat/internal/message"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const maxMessageSize = 64 * 1024

// Connection represents a GoChat protocol stream to a remote peer.
type Connection struct {
	stream  network.Stream
	peerID  peer.ID
	reader  *bufio.Reader
	writeMu sync.Mutex
}

func newConnection(stream network.Stream) *Connection {
	return &Connection{
		stream: stream,
		peerID: stream.Conn().RemotePeer(),
		reader: bufio.NewReaderSize(stream, maxMessageSize),
	}
}

// Send serializes and writes one message to the stream.
func (c *Connection) Send(msg message.Message) error {
	if c == nil || c.stream == nil {
		return errors.New("connection is not initialized")
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("serialize message: %w", err)
	}
	if len(payload)+1 > maxMessageSize {
		return fmt.Errorf("message exceeds maximum size of %d bytes", maxMessageSize)
	}

	payload = append(payload, '\n')
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	for len(payload) > 0 {
		written, err := c.stream.Write(payload)
		if err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}

	return nil
}

// Receive reads and deserializes one message from the stream.
func (c *Connection) Receive() (message.Message, error) {
	if c == nil || c.reader == nil {
		return message.Message{}, errors.New("connection is not initialized")
	}

	payload, err := c.reader.ReadSlice('\n')
	if err != nil {
		if errors.Is(err, bufio.ErrBufferFull) {
			return message.Message{}, fmt.Errorf("received message exceeds maximum size of %d bytes", maxMessageSize)
		}
		if errors.Is(err, io.EOF) && len(payload) > 0 {
			return message.Message{}, errors.New("received incomplete message")
		}
		return message.Message{}, fmt.Errorf("read message: %w", err)
	}

	var msg message.Message
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&msg); err != nil {
		return message.Message{}, fmt.Errorf("deserialize message: %w", err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return message.Message{}, err
	}

	return msg, nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("received multiple JSON values in one message")
		}
		return fmt.Errorf("deserialize message: %w", err)
	}

	return nil
}

// Close closes the underlying libp2p stream.
func (c *Connection) Close() error {
	if c == nil || c.stream == nil {
		return errors.New("connection is not initialized")
	}

	return c.stream.Close()
}
