//go:build js

package jsrpc

import (
	"embed"
	"encoding/gob"
	"net"
	"net/rpc"
	"sync"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/formtest"
	"github.com/CCorderZugcat/zugoui/input"
	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/observable/observabletest"
	"github.com/CCorderZugcat/zugoui/wsrpc"
)

//go:embed testdata/*
var fsys embed.FS

func endpoints(
	t testing.TB,
	server *wsrpc.Server,
) (
	b *Browser,
	bclient wsrpc.Browser,
	sclient Server,
	stop func(),
) {
	wg := &sync.WaitGroup{}

	sp_s, sp_c := net.Pipe()
	bp_s, bp_c := net.Pipe()

	bclient = wsrpc.Browser{Client: rpc.NewClient(bp_c)} // browser side client calling web server
	srpc := rpc.NewServer()
	err := srpc.Register(server)
	require.NoError(t, err)

	sclient = Server{Client: rpc.NewClient(sp_c)} // server side client calling browser
	b = New(&sclient)                             // browser side rpc server
	brpc := rpc.NewServer()
	err = brpc.Register(b)
	require.NoError(t, err)

	wg.Go(func() {
		defer sp_s.Close()
		srpc.ServeConn(sp_s)
	})
	wg.Go(func() {
		defer bp_s.Close()
		brpc.ServeConn(bp_s)
	})

	return b, bclient, sclient, func() {
		sp_c.Close()
		bp_c.Close()
		wg.Wait()
		b.Release()
	}
}

type Model struct {
	Product  string `bind:"product"`
	Quantity int    `bind:"quantity"`
}

func TestObjectBinding(t *testing.T) {
	gob.Register(&Model{})
	formtest.SetBody(t, fsys, "form.html")

	s := wsrpc.New()
	_, bclient, _, stop := endpoints(t, s)
	defer stop()

	m := &Model{
		Product:  "initial",
		Quantity: 1,
	}

	h, err := bclient.NewValueBinding("products", "products", nil, &m)
	require.NoError(t, err)
	defer bclient.Unbind(h)

	ob, ch := observabletest.New()
	defer close(ch)

	s.AddValueObserver("products", ob)

	elem, err := input.Element("product")
	require.NoError(t, err)

	// verify UI element is populated
	assert.Equal(t, m.Product, elem.Get("value").String())
	t.Log("initial:", elem.Get("value"))

	// change UI element contents (as if a user did with change event)
	elem.Set("value", "updated")
	jsglue.DispatchEvent(elem, "change", map[string]any{"bubbles": true})

	t.Log("waiting for browswer side field update")
	ob1 := <-ch

	assert.Equal(t, "Product", ob1.Key)
	assert.Equal(t, "updated", ob1.Value)

	// change field from server to browser
	wsrpc.Observer{Browser: bclient, Handle: h}.SetValue("Product", "updated again")

	// (that call was synchronous)
	assert.Equal(t, "updated again", elem.Get("value").String())

	t.Log("updated:", elem.Get("value"))
}

func TestActionBinding(t *testing.T) {
	formtest.SetBody(t, fsys, "form.html")
	elem, err := input.Element("productAction")
	require.NoError(t, err)

	s := wsrpc.New()

	ao, ac := observabletest.New()
	defer close(ac)

	s.AddActionObserver(ao)

	_, bclient, _, stop := endpoints(t, s)
	defer stop()

	binding, err := bclient.NewClickBinding("productAction", "myAction")
	require.NoError(t, err)
	defer bclient.Unbind(binding)

	elem.Call("click")

	action := <-ac
	assert.Equal(t, "myAction", action.Value)
	assert.Equal(t, "action", action.Key)

	t.Logf("action: %s=%s", action.Value, action.Key)
}

func TestEventListener(t *testing.T) {
	s := wsrpc.New()

	b, bclient, _, stop := endpoints(t, s)
	defer stop()

	ch := make(chan string, 1)

	cb := js.FuncOf(func(_ js.Value, args []js.Value) any {
		ch <- args[0].Get("detail").String()
		return nil
	})
	defer cb.Release()

	b.addEventListener("test", cb.Value)
	bclient.DispatchEvent("test", "event arg")

	arg := <-ch
	assert.Equal(t, "event arg", arg)

	b.removeEventListener("test", cb.Value)
	t.Logf("event: %s", arg)
}
