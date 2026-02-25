package wsconn

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/coder/websocket"
)

// use the message oriented websocket connection to multiplex two way communication between
// different endpoints over the same connection.

var (
	ErrNotBinary = errors.New("websocket message was not binary")
	ErrTruncated = errors.New("received message was truncated")
	ErrZero      = errors.New("got zero sized message")
)

type PacketConn interface {
	net.PacketConn
	ReadMessage() ([]byte, net.Addr, error)
}

// MessageEndpoint is a [net.Addr] representing a multiplexed endpoint
type MessageEndpoint uint8

func (m MessageEndpoint) Network() string { return "endpoint" }
func (m MessageEndpoint) String() string  { return strconv.Itoa(int(m)) }

// Message is a PacketConn sending and receiving multiplex messages.
// Use NewDemux instead of this type directly to create streaming endpoints.
type Message struct {
	*websocket.Conn
	ctx func() context.Context
}

var _ PacketConn = ((*Message)(nil))

// NewMessage creates a PacketConn interface of our multiplexed messages.
// The source and destination addresses are of type MessageEndpoint.
// Can only have one of these per websocket.Conn instance.
func NewMessage(ctx context.Context, conn *websocket.Conn) *Message {
	return &Message{
		Conn: conn,
		ctx:  func() context.Context { return ctx },
	}
}

// ReadMessage is preferred over ReadFrom to avoid truncation.
func (m *Message) ReadMessage() (msg []byte, addr net.Addr, err error) {
	mt, buf, err := m.Conn.Read(m.ctx())
	if err != nil {
		return nil, nil, err
	}
	if mt != websocket.MessageBinary {
		return nil, nil, ErrNotBinary
	}
	if len(buf) == 0 {
		return nil, nil, ErrZero
	}
	addr = MessageEndpoint(buf[0])
	return buf[1:], addr, nil
}

// net.PacketConn implementation

func (m *Message) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	var buf []byte
	buf, addr, err = m.ReadMessage()

	n = copy(p, buf)
	if n < len(buf) {
		err = ErrTruncated
	}

	return n, addr, err
}

func (m *Message) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	msg := &bytes.Buffer{}
	msg.WriteByte(byte(addr.(MessageEndpoint)))
	msg.Write(p)
	err = m.Conn.Write(m.ctx(), websocket.MessageBinary, msg.Bytes())
	if err != nil {
		return 0, err
	}
	return len(p), err
}

func (m *Message) Close() error {
	return m.Conn.Close(websocket.StatusNormalClosure, "closed")
}

func (m *Message) LocalAddr() net.Addr {
	return nil
}

func (m *Message) SetDeadline(t time.Time) error {
	return nil
}

func (m *Message) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *Message) SetWriteDeadline(t time.Time) error {
	return nil
}
