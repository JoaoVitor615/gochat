package p2p

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const maxMessageSize = 64 * 1024

var ErrUnexpectedFrame = errors.New("unexpected P2P frame type")

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
	return c.SendFrame(message.Frame{Type: message.FrameTypeMessage, Message: &msg})
}

// SendAck writes an ACK after the referenced message has been committed locally.
func (c *Connection) SendAck(messageID string) error {
	return c.SendFrame(message.Frame{Type: message.FrameTypeAck, Ack: &message.Ack{MessageID: messageID}})
}

// SendFrame serializes and writes one protocol frame.
func (c *Connection) SendFrame(frame message.Frame) error {
	if c == nil || c.stream == nil {
		return errors.New("connection is not initialized")
	}

	if err := validateFrame(frame); err != nil {
		return err
	}
	payload, err := json.Marshal(frame)
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
	frame, err := c.ReceiveFrame()
	if err != nil {
		return message.Message{}, err
	}
	if frame.Type != message.FrameTypeMessage || frame.Message == nil {
		return message.Message{}, fmt.Errorf("receive message: %w: %s", ErrUnexpectedFrame, frame.Type)
	}
	return *frame.Message, nil
}

// ReceiveFrame reads and validates one protocol frame.
func (c *Connection) ReceiveFrame() (message.Frame, error) {
	if c == nil || c.reader == nil {
		return message.Frame{}, errors.New("connection is not initialized")
	}

	payload, err := c.reader.ReadSlice('\n')
	if err != nil {
		if errors.Is(err, bufio.ErrBufferFull) {
			return message.Frame{}, fmt.Errorf("received frame exceeds maximum size of %d bytes", maxMessageSize)
		}
		if errors.Is(err, io.EOF) && len(payload) > 0 {
			return message.Frame{}, errors.New("received incomplete frame")
		}
		return message.Frame{}, fmt.Errorf("read frame: %w", err)
	}

	var frame message.Frame
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&frame); err != nil {
		return message.Frame{}, fmt.Errorf("deserialize frame: %w", err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return message.Frame{}, err
	}
	if err := validateFrame(frame); err != nil {
		return message.Frame{}, err
	}
	return frame, nil
}

func validateFrame(frame message.Frame) error {
	switch frame.Type {
	case message.FrameTypeMessage:
		if frame.Message == nil || frame.Ack != nil {
			return errors.New("invalid message frame")
		}
	case message.FrameTypeAck:
		if frame.Ack == nil || frame.Ack.MessageID == "" || frame.Message != nil {
			return errors.New("invalid ACK frame")
		}
	default:
		return fmt.Errorf("unknown frame type %q", frame.Type)
	}
	return nil
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
