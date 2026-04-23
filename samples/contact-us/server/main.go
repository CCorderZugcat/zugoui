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
	"github.com/CCorderZugcat/zugoui/samples/contact-us/model"
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

	const ep = "/anon/contact/app/rpc"

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

func nope(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "%s %v\r\n", http.StatusText(statusCode), err)
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// step 1: create a web socket
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		nope(w, http.StatusInternalServerError, err)
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
	m := &model.ContactForm{
		Subject: "Zugcat Inquiries", // a default value
	}

	// NewModel creates an Observable proxy to your model's data.
	// You may create as many model and associated observable proxies as you like.
	o := controllers.New(&m)
	co := controllers.New(&model.ContactControls{})

	// we're going to fail our action once every 3rd time
	tries := 0

	// bind an action called "submit" to the button whose ID is "submit"
	c.BindAction("submit", "submit", func(string) {
		fmt.Printf("submit button clicked: %+v\n", *m)

		detail := map[string]any{
			"ok":      true,
			"message": "Sent",
		}

		// fail this occasionally
		tries++
		if (tries % 3) == 0 {
			detail["ok"] = false
			detail["message"] = "Could not make any toast"

			fmt.Printf("failing this one\n")
		} else {
			// do server side validation
			if err := observable.ValidateSource(o); err != nil {
				fmt.Printf("client sent invalid data: %v\n", err)
				detail["ok"] = false
				detail["message"] = "Form has validation errors."
			}
		}

		co.SetValue("Status", detail["message"])

		if detail["ok"].(bool) {
			// now, let's disable the submit button
			co.SetValue("SubmitDisabled", true)
		}

		// this form's logic wants an event upon completion
		if err := c.Browser.DispatchEvent("contact-submitted", detail); err != nil {
			fmt.Fprintf(os.Stderr, "failed to dispatch submit event: %v\n", err)
		}
	})

	// Bind the reset button's action
	c.BindAction("reset", "reset", func(string) {
		// a real application might have a default template
		o.SetValue("Subject", "Zugcat Inquiries")
		o.SetValue("Message", "This user cannot make up their mind.")
		o.SetValue("First", "")
		o.SetValue("Last", "")
		o.SetValue("Email", "")

		// reset the UI elements
		co.SetValue("Status", "")
		co.SetValue("SubmitDisabled", false)
	})

	// BindValue binds the observable model proxy with the UI.
	// Multiple models or instances of the same model may be bound.
	// Use a unique name for each instance.
	if err := c.BindValues("contactUs", "contact", []string{"contact"}, o); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind model observer: %v\n", err)
	}
	if err := c.BindValues("contactUI", "", nil, co); err != nil {
		fmt.Fprintf(os.Stderr, "unabel to find controls: %v\n", err)
	}

	<-done
	fmt.Fprintf(os.Stderr, "handleRPC exiting\n")
}
