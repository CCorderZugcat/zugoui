package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"

	"github.com/coder/websocket"

	"github.com/CCorderZugcat/zugoui/controller"
	"github.com/CCorderZugcat/zugoui/gzasm"
	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/samples/contact-us/model"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	distOpt := ""
	flag.StringVar(&distOpt, "dist", distOpt, "serve anything outside this prefix form /index.html")

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "usage: server dir")
		os.Exit(2)
	}

	fsys := os.DirFS(flag.Arg(0))

	const ep = "/anon/contact/app/rpc"

	mux := http.NewServeMux()
	fshandler := gzasm.New(http.FileServerFS(fsys), fsys)

	if distOpt != "" {
		// quick and hacky way to make client side routing methods happy

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fs.Stat(fsys, path.Join(".", r.URL.Path)); err == nil {
				fshandler.ServeHTTP(w, r)
				return
			}

			fd, err := fsys.Open("index.html")
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "%v\r\n", err)
				return
			}
			defer fd.Close()

			st, err := fd.Stat()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "%v\r\n", err)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Length", strconv.Itoa(int(st.Size())))
			io.Copy(w, fd)
		})
		mux.Handle(distOpt, fshandler)
	} else {
		mux.Handle("/", fshandler)
	}

	mux.HandleFunc(ep, handleRPC)

	l, err := net.Listen("tcp", "[::1]:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("listening on http://%v\n", l.Addr())
	fmt.Printf("websocket endpoint is %s\n", ep)

	s := &http.Server{Handler: mux}

	errC := make(chan error)
	go func() {
		err := s.Serve(l)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errC <- err
	}()

	select {
	case <-ctx.Done():
		fmt.Println("interrupt")
		s.Shutdown(context.WithoutCancel(ctx))
		err = <-errC
	case err = <-errC:
	}

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
	o := observable.NewModel(&m)

	// another random observable value we will bind to a property
	submitDisabled := true
	sdo := observable.NewModel(&submitDisabled)

	// we're going to fail our action once every 3rd time
	tries := 0

	// HandleActions sets the functio to execute for click actions (e.g. buttons and menus)
	// Call this first, and only once to not miss events.
	c.HandleActions(func(action string) {
		switch action {
		case "submit":
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

			o.SetValue("EmailStatus", detail["message"])

			if detail["ok"].(bool) {
				// now, let's disable the submit button
				sdo.SetValue("value", true)
			}

			// this form's logic wants an event upon completion
			if err := c.Browswer.DispatchEvent("contact-submitted", detail); err != nil {
				fmt.Fprintf(os.Stderr, "failed to dispatch submit event: %v\n", err)
			}

		case "reset":
			// a real application might have a default template
			o.SetValue("Subject", "Zugcat Inquiries")
			o.SetValue("Message", "This user cannot make up their mind.")
			o.SetValue("First", "")
			o.SetValue("Last", "")
			o.SetValue("Email", "")

			// clear the other fields
			o.SetValue("EmailStatus", "")
		}
	})

	// BindActions establishes which UI elements send actions if clicked.
	// The action name is passed to your function in HandleActions
	// "submit" in this example is commented out for a form that opts to do its own validation logic.
	// Thus, it calls "sendAction" on its own post-validation submit logic.
	// Otherwise, we can handle it automatically here without any client side code.
	if err := c.BindActions(
		"submit", "submit",
		"reset", "reset",
	); err != nil {
		fmt.Fprintf(os.Stderr, "failed to bind actions: %v\n", err)
	}

	// BindModel binds the observable model proxy with the UI.
	// Multiple models or instances of the same model may be bound.
	// Use a unique name for each instance.
	if err := c.BindModel("contactUs", o); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind model observer: %v\n", err)
	}

	// BindValue binds an arbitrary value to an arbitrary property.
	// In this case, we want to control the disabled status of the button.
	// We could also simply make it part of the model.
	if err := c.BindValue("submitDisabled", []string{"submit"}, "disabled", sdo); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind submit button: %v\n", err)
	}

	// and let's observe the model so we can update the button based on validation
	o.AddObserver("", observable.NewActionObserver(func(string, any) {
		sdo.SetValue("value", observable.ValidateSource(o) != nil)
	}))

	<-done
	fmt.Fprintf(os.Stderr, "handleRPC exiting\n")
}
