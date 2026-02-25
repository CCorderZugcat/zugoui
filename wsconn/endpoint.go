package wsconn

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// using the multiplex messages, provide an endpoint

// Demux is a mechanism to send and receive multiplexed messages over a websocket,
// with each endpoint having a MessageEndpoint address.
type Demux struct {
	conn      PacketConn
	cancel    func()
	done      chan struct{}
	endpoints map[MessageEndpoint]*endpoint
	lck       sync.RWMutex
}

// NewDemux creates a new Demux instance.
// Calling Close will close the underlying connection.
func NewDemux(ctx context.Context, conn PacketConn) *Demux {
	ctx, cancel := context.WithCancel(ctx)

	d := &Demux{
		conn:      conn,
		cancel:    cancel,
		done:      make(chan struct{}),
		endpoints: make(map[MessageEndpoint]*endpoint),
	}

	go func() {
		var msg []byte
		var addr net.Addr
		var err error

		for {
			msg, addr, err = d.conn.ReadMessage()
			if err != nil {
				if errors.Is(err, context.Canceled) {
					err = nil
				}
				break
			}

			d.lck.RLock()
			ep, ok := d.endpoints[addr.(MessageEndpoint)]
			d.lck.RUnlock()

			if ok {
				ep.input(msg)
			}
		}

		d.lck.Lock()
		for _, ep := range d.endpoints {
			ep.close(err)
		}
		d.lck.Unlock()

		close(d.done)
	}()

	return d
}

// NewEndpoint cereates and attaches a new endpoint with the desired address.
func (d *Demux) NewEndpoint(local MessageEndpoint) net.Conn {
	d.lck.Lock()
	defer d.lck.Unlock()

	ep := newEndpoint(local, d.conn)
	d.endpoints[local] = ep

	return ep
}

// Close stops the Demux go routine, closes the underlying websocket and any attached endpoints.
func (d *Demux) Close() error {
	d.cancel()
	d.conn.Close()
	<-d.done
	return nil
}

type endpoint struct {
	local net.Addr
	conn  net.PacketConn
	read  *io.PipeReader
	write *io.PipeWriter
}

var _ net.Conn = ((*endpoint)(nil))

func newEndpoint(local MessageEndpoint, message net.PacketConn) *endpoint {
	e := &endpoint{
		local: local,
		conn:  message,
	}
	e.read, e.write = io.Pipe()

	return e
}

func (e *endpoint) input(msg []byte) {
	e.write.Write(msg)
}

func (e *endpoint) close(err error) {
	e.write.CloseWithError(err)
}

// net.Conn implementation

func (e *endpoint) RemoteAddr() net.Addr {
	return e.local
}

func (e *endpoint) LocalAddr() net.Addr {
	return e.local
}

func (e *endpoint) Read(b []byte) (n int, err error) {
	return e.read.Read(b)
}

func (e *endpoint) Write(b []byte) (n int, err error) {
	return e.conn.WriteTo(b, e.local)
}

func (e *endpoint) Close() error {
	e.close(nil)
	return nil
}

func (m *endpoint) SetDeadline(t time.Time) error {
	return nil
}

func (m *endpoint) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *endpoint) SetWriteDeadline(t time.Time) error {
	return nil
}
