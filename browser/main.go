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
	"time"

	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/jsrpc"
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

	// create the rpc service
	// "server" refers to the client connection to the web server
	// "browser" refers to the rpc service running in the browswer

	server := &jsrpc.Server{}
	browser := jsrpc.New(server)
	defer browser.Destroy()

	rpcServer := rpc.NewServer()
	if err := rpcServer.Register(browser); err != nil {
		fmt.Fprintln(os.Stderr, "failed to register rpc server")
		return err
	}

	for {
		if err := connection(ctx, u.String(), server, rpcServer, func() {
			ready := js.Global().Get("zugouiReady")
			if ready.Type() == js.TypeFunction {
				ready.Invoke(browser.JsObject())
			}
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Println("connection lost")
		}
		time.Sleep(15 * time.Second)
	}
}
