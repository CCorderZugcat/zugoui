package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/coder/websocket"

	"github.com/CCorderZugcat/zugoui/controller"
	"github.com/CCorderZugcat/zugoui/observable"
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

func syncToppings(m, buttons observable.MutableSource, toppings []*model.Topping, offset int) {
	setTopping := func(n int) {
		keyPath := fmt.Sprintf("Toppings.%d", n-offset)
		if n < len(toppings) {
			m.SetValue(keyPath, toppings[n])
		} else {
			m.SetValue(keyPath, &model.Topping{})
		}
	}

	buttons.SetValue("Up", offset == 0)
	buttons.SetValue("Down", (len(toppings)-offset) <= 3)

	for i := range 3 {
		setTopping(offset + i)
	}
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// step 1: create a web socket
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v\r\n", err)
		return
	}
	defer ws.CloseNow()

	// step 2: start a controller.
	c, done, err := controller.Start(ctx, ws)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start controller: %v\n", err)
		return
	}
	defer c.Release()

	// step 3: establish your model and its data.
	m := &model.Pizza{
		Size: "medium",
	}

	buttons := observable.NewModel(&model.Buttons{})

	// NewModel creates an Observable proxy to your model's data.
	// You may create as many model and associated observable proxies as you like.
	o := observable.NewModel(&m)
	defer o.Release()

	toppingOffset := 0
	toppings := make([]*model.Topping, 1)
	toppings[0] = &model.Topping{
		Show:    true,
		Topping: "broccoli",
	}
	syncToppings(o, buttons, toppings, toppingOffset)

	o.AddObserver("", observable.NewActionObserver(func(string, any) {
		fmt.Printf("PIZZA: size=%s\n", m.Size)
		for _, topping := range toppings {
			fmt.Printf("\t%s\n", topping.Topping)
		}
		fmt.Printf("\n")
	}))

	// HandleActions sets the functio to execute for click actions (e.g. buttons and menus)
	// Call this first, and only once to not miss events.
	c.HandleActions(func(action string) {
		switch action {
		case "add":
			toppingOffset = 0
			toppings = append([]*model.Topping{{Show: true}}, toppings...)

		case "up":
			if toppingOffset > 0 {
				toppingOffset--
			}

		case "down":
			if toppingOffset < (len(toppings) - 3) {
				toppingOffset++
			}

		case "remove.0", "remove.1", "remove.2":
			n, _ := strconv.Atoi(strings.TrimPrefix(action, "remove."))
			n += toppingOffset

			if n < len(toppings) {
				copy(toppings[n:], toppings[n+1:])
				toppings = toppings[:len(toppings)-1]
			}
		}

		syncToppings(o, buttons, toppings, toppingOffset)
	})

	// BindActions establishes which UI elements send actions if clicked.
	// The action name is passed to your function in HandleActions
	// "submit" in this example is commented out for a form that opts to do its own validation logic.
	// Thus, it calls "sendAction" on its own post-validation submit logic.
	// Otherwise, we can handle it automatically here without any client side code.
	if err := c.BindActions(
		"pizza.add", "add",
		"pizza.up", "up",
		"pizza.down", "down",
		"pizza.toppings.0.remove", "remove.0",
		"pizza.toppings.1.remove", "remove.1",
		"pizza.toppings.2.remove", "remove.2",
	); err != nil {
		fmt.Fprintf(os.Stderr, "failed to bind actions: %v\n", err)
	}

	// BindValue binds the observable model proxy with the UI.
	// Multiple models or instances of the same model may be bound.
	// Use a unique name for each instance.
	if err := c.BindValues("pizza", "pizza", []string{"pizza"}, o); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind model observer: %v\n", err)
	}
	if err := c.BindValues("controls", "", []string{"pizza"}, buttons); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind model controls: %v\n", err)
	}

	<-done
	fmt.Fprintf(os.Stderr, "handleRPC exiting\n")
}
