//go:build js

// Package browser implements the main loop for the browser side wasm code
package browser

import (
	"context"
	"fmt"
	"net/rpc"
	"net/url"
	"os"
	"path"
	"sync"
	"syscall/js"

	"github.com/coder/websocket"

	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/jsrpc"
	"github.com/CCorderZugcat/zugoui/wsconn"
)

// Main runs the main loop of the browser side of things.
// It is _critical_ your main package calls gob.Register() on your shared model types.
// This call is blocking, and should be the last thing your main package's main function calls.
// exit with non-0 to the system if error is not nil.
func Main(ctx context.Context, endpoint string) (err error) {
	defer func() {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	readyCond := &sync.Cond{}
	readyCond.L = &sync.Mutex{}

	// wait for the document to be ready, if it isn't yet
	document := js.Global().Get("document")
	document.Call("addEventListener", "DOMContentLoaded",
		jsglue.FuncOfOnce(
			func(_ js.Value, _ []js.Value) any {
				readyCond.L.Lock()
				defer readyCond.L.Unlock()
				readyCond.Signal()

				return nil
			},
		),
	)

	for document.Get("readyState").String() == "loading" {
		fmt.Println("still loading")
		readyCond.Wait()
	}

	if endpoint == "" {
		endpoint = "rpc"
	}

	// from the location this page loaded, connect back at the same path+endpoint
	window := js.Global().Get("window").Get("location")
	wspath := path.Join(window.Get("pathname").String(), endpoint)
	u, err := url.Parse(window.Get("origin").String() + wspath)
	if err != nil {
		return err
	}

	ws, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "websocket.Dial(%v) failed\n", u.String())
		return err
	}

	fmt.Println("websocket connected")

	// create a Demux for the two endpoints
	dmx := wsconn.NewDemux(ctx, wsconn.NewMessage(ctx, ws))
	defer dmx.Close()

	ep0 := dmx.NewEndpoint(0)
	ep1 := dmx.NewEndpoint(1)
	defer ep0.Close()
	defer ep1.Close()

	// create the rpc service
	browser := jsrpc.New(jsrpc.Server{Client: rpc.NewClient(ep0)})
	defer browser.Destroy()

	ready := js.Global().Get("zugouiReady")
	if ready.Type() == js.TypeFunction {
		ready.Invoke(browser.JsObject())
	}

	rpcServer := rpc.NewServer()
	if err := rpcServer.Register(browser); err != nil {
		fmt.Fprintln(os.Stderr, "failed to register rpc server")
		return err
	}

	// ..and serve it for the lifetime of this page
	fmt.Println("listening")
	rpcServer.ServeConn(ep1)
	return nil
}
