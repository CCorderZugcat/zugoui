package controller

import (
	"context"
	"net/rpc"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/CCorderZugcat/zugoui/wsconn"
	"github.com/CCorderZugcat/zugoui/wsrpc"
)

// Start starts server side rpc and creates a Controller.
// wait on the returned channel after creating your bindings.
func Start(ctx context.Context, ws *websocket.Conn) (c *Controller, done <-chan struct{}, err error) {
	s := wsrpc.New()
	rpcServer := rpc.NewServer()

	if err := rpcServer.Register(s); err != nil {
		return nil, nil, err
	}

	mx := wsconn.NewDemux(ctx, wsconn.NewMessage(ctx, ws))

	// endpoint 0: used for server side rpc
	// endpoint 1: used for browser side rpc
	ep0, ep1 := mx.NewEndpoint(0), mx.NewEndpoint(1)

	// rpc client to the browser
	browser := wsrpc.Browser{Client: rpc.NewClient(ep1)}
	c = New(s, browser, uuid.NewString())

	doneC := make(chan struct{})

	go func() {
		defer ep0.Close()

		// rpc server called from the browser
		rpcServer.ServeConn(ep0)
		close(doneC)
	}()

	return c, doneC, nil
}
