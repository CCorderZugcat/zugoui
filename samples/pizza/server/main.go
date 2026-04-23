package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/coder/websocket"

	"github.com/CCorderZugcat/zugoui/controller"
	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
	"github.com/CCorderZugcat/zugoui/observable/controllers/scroll"
	"github.com/CCorderZugcat/zugoui/samples/pizza/model"
	"github.com/CCorderZugcat/zugoui/samples/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	listenOpt := "localhost:"

	flag.StringVar(&listenOpt, "listen", listenOpt, "listen address")

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "usage: server dir")
		os.Exit(2)
	}

	fsys := os.DirFS(flag.Arg(0))

	const ep = "/user/pizza/app/rpc"

	mux := http.NewServeMux()
	mux.HandleFunc(ep, handleRPC)

	l, err := net.Listen("tcp", listenOpt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("listening on http://%v\n", l.Addr())
	fmt.Printf("websocket endpoint is %s\n", ep)

	err = server.Serve(ctx, l, fsys, mux)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v\r\n", err)
		return
	}
	defer ws.CloseNow()

	c, done, err := controller.Start(ctx, ws)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start controller: %v\n", err)
		return
	}
	defer c.Release()

	m := &model.Pizza{
		Size: "medium",
		Toppings: []*model.Topping{
			{
				Show:    true,
				Topping: "broccoli",
			},
		},
	}

	buttons := controllers.New(&model.Buttons{})
	defer buttons.Release()

	o := controllers.New(&m)
	defer o.Release()

	/*
		p := observable.NewPathObserver("*", o)
		p.AddObserver("", observable.NewActionObserver(func(string, any) {
			fmt.Printf("PIZZA: size=%s\n", m.Size)
			for n, topping := range m.Toppings {
				if topping == nil {
					fmt.Printf("\t%d is nil\n", n)
					continue
				}
				fmt.Printf("\t%d: %s\n", n, topping.Topping)
			}
			fmt.Printf("\n")
		}))
	*/

	toppingsView := o.Value("Toppings").(*scroll.Scroll)
	toppings := toppingsView.Source().(observable.MutableSource)

	c.BindAction("pizza.add", "add", func(string) {
		toppingsView.Insert("add")
		observable.SetKeyPath(toppingsView, "0.Show", true)
	})

	c.BindAction("pizza.up", "up", toppingsView.Up)
	c.BindAction("pizza.down", "down", toppingsView.Down)

	c.BindAction("pizza.toppings.0.remove", "remove.0", func(string) {
		toppings.RemoveValueAt(0)
	})
	c.BindAction("pizza.toppings.1.remove", "remove.1", func(string) {
		toppings.RemoveValueAt(1)
	})
	c.BindAction("pizza.toppings.2.remove", "remove.2", func(string) {
		toppings.RemoveValueAt(2)
	})

	canUp, _ := observable.NewBinding(
		"canUp", toppingsView,
		"Up", buttons,
		"isZero",
	)
	defer canUp.Release()

	canDown, _ := observable.NewBinding(
		"canDown", toppingsView,
		"Down", buttons,
		"isZero",
	)
	defer canDown.Release()

	if err := c.BindValues("pizza", "pizza", []string{"pizza"}, o); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind model observer: %v\n", err)
	}
	if err := c.BindValues("controls", "", []string{"pizza"}, buttons); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind model controls: %v\n", err)
	}

	<-done
	fmt.Fprintf(os.Stderr, "handleRPC exiting\n")
}
